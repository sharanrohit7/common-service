package blobclient

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// AzureBlobClient implements BlobClient using Azure Blob Storage.
type AzureBlobClient struct {
	client      *azblob.Client
	logger      logging.Logger
	accountName string
}

// NewAzureBlobClient creates a new Azure Blob Storage client.
// accountName: Azure storage account name
// accountKey: Azure storage account key (optional if using managed identity)
// useManagedIdentity: if true, uses managed identity instead of account key
func NewAzureBlobClient(accountName, accountKey string, useManagedIdentity bool, logger logging.Logger) (*AzureBlobClient, error) {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)

	var client *azblob.Client

	if useManagedIdentity || accountKey == "" {
		// Use managed identity (for Azure environments) or default credentials
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure credential: %w", err)
		}
		client, err = azblob.NewClient(serviceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure blob client: %w", err)
		}
	} else {
		// Use shared key authentication
		cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure blob client: %w", err)
		}
	}

	return &AzureBlobClient{
		client:      client,
		logger:      logger,
		accountName: accountName,
	}, nil
}

// Upload uploads data to Azure Blob Storage.
func (a *AzureBlobClient) Upload(ctx context.Context, container, blobName string, data io.Reader, contentType string) (string, error) {
	logger := a.logger.With(
		logging.NewField("operation", "blob.upload"),
		logging.NewField("container", container),
		logging.NewField("blob", blobName),
	)

	logger.Info("Starting blob upload")

	// Ensure container exists
	_, err := a.client.CreateContainer(ctx, container, nil)
	if err != nil {
		// Container might already exist, which is fine
		logger.Debug("Container create result (may already exist)", logging.NewField("error", err.Error()))
	}

	uploadOptions := &azblob.UploadStreamOptions{}
	// Note: ContentType can be set via HTTPHeaders if needed
	// For now, using basic upload - extend as needed

	_, err = a.client.UploadStream(ctx, container, blobName, data, uploadOptions)
	if err != nil {
		logger.Error("Failed to upload blob", logging.NewField("error", err))
		return "", fmt.Errorf("failed to upload blob: %w", err)
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", a.accountName, container, blobName)
	logger.Info("Blob upload successful", logging.NewField("url", url))

	return url, nil
}

// Get retrieves a blob from Azure Blob Storage.
func (a *AzureBlobClient) Get(ctx context.Context, container, blobName string) (io.ReadCloser, error) {
	logger := a.logger.With(
		logging.NewField("operation", "blob.get"),
		logging.NewField("container", container),
		logging.NewField("blob", blobName),
	)

	logger.Info("Retrieving blob")

	downloadResponse, err := a.client.DownloadStream(ctx, container, blobName, nil)
	if err != nil {
		logger.Error("Failed to download blob", logging.NewField("error", err))
		return nil, fmt.Errorf("failed to download blob: %w", err)
	}

	logger.Info("Blob retrieved successfully")
	return downloadResponse.Body, nil
}

// Delete deletes a blob from Azure Blob Storage.
func (a *AzureBlobClient) Delete(ctx context.Context, container, blobName string) error {
	logger := a.logger.With(
		logging.NewField("operation", "blob.delete"),
		logging.NewField("container", container),
		logging.NewField("blob", blobName),
	)

	logger.Info("Deleting blob")

	_, err := a.client.DeleteBlob(ctx, container, blobName, nil)
	if err != nil {
		logger.Error("Failed to delete blob", logging.NewField("error", err))
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	logger.Info("Blob deleted successfully")
	return nil
}

// Exists checks if a blob exists in Azure Blob Storage.
func (a *AzureBlobClient) Exists(ctx context.Context, container, blobName string) (bool, error) {
	// Try a minimal download to check existence
	_, err := a.client.DownloadStream(ctx, container, blobName, &azblob.DownloadStreamOptions{
		Range: azblob.HTTPRange{
			Offset: 0,
			Count:  1,
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "BlobNotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check blob existence: %w", err)
	}

	return true, nil
}

// List lists blobs in a container with optional prefix.
func (a *AzureBlobClient) List(ctx context.Context, container, prefix string) ([]BlobInfo, error) {
	logger := a.logger.With(
		logging.NewField("operation", "blob.list"),
		logging.NewField("container", container),
		logging.NewField("prefix", prefix),
	)

	logger.Info("Listing blobs")

	listOptions := &azblob.ListBlobsFlatOptions{
		Prefix: &prefix,
	}

	pager := a.client.NewListBlobsFlatPager(container, listOptions)

	var blobs []BlobInfo
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error("Failed to list blobs", logging.NewField("error", err))
			return nil, fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, item := range page.Segment.BlobItems {
			blobInfo := BlobInfo{
				Name:        *item.Name,
				Size:        *item.Properties.ContentLength,
				ContentType: "",
				URL:         fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", a.accountName, container, *item.Name),
			}

			if item.Properties.ContentType != nil {
				blobInfo.ContentType = *item.Properties.ContentType
			}

			if item.Properties.LastModified != nil {
				blobInfo.LastModified = item.Properties.LastModified.Format(time.RFC3339)
			}

			blobs = append(blobs, blobInfo)
		}
	}

	logger.Info("Blob listing completed", logging.NewField("count", len(blobs)))
	return blobs, nil
}

// Note: For local development, you can use Azurite emulator:
// 1. Install: npm install -g azurite
// 2. Run: azurite --silent --location ~/azurite --debug ~/azurite/debug.log
// 3. Set connection string to: DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6rqj1nZA==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;
// 4. Update NewAzureBlobClient to accept connection string as an alternative
