package examples

import (
	"context"
	"fmt"
	"io"

	"github.com/yourorg/go-service-kit/pkg/blobclient"
	"github.com/yourorg/go-service-kit/pkg/config"
	"github.com/yourorg/go-service-kit/pkg/logging"
	"github.com/yourorg/go-service-kit/pkg/servicebusclient"
)

// ExampleBlobClientUsage demonstrates how to create and use a blob client.
func ExampleBlobClientUsage(cfg *config.Config, logger logging.Logger) (blobclient.BlobClient, error) {
	// Create Azure Blob client
	blobClient, err := blobclient.NewAzureBlobClient(
		cfg.BlobStorageAccountName,
		cfg.BlobStorageAccountKey,
		false, // useManagedIdentity
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}
	
	// For testing, you can use the mock client:
	// blobClient := blobclient.NewMockBlobClient()
	
	return blobClient, nil
}

// ExampleServiceBusClientUsage demonstrates how to create and use a Service Bus client.
func ExampleServiceBusClientUsage(cfg *config.Config, logger logging.Logger) (servicebusclient.ServiceBusClient, error) {
	// Create Azure Service Bus client
	serviceBusClient, err := servicebusclient.NewAzureServiceBusClient(
		cfg.ServiceBusNamespace,
		cfg.ServiceBusKeyName,
		cfg.ServiceBusKeyValue,
		false, // useManagedIdentity
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Service Bus client: %w", err)
	}
	
	// For testing, you can use the mock client:
	// serviceBusClient := servicebusclient.NewMockServiceBusClient()
	
	return serviceBusClient, nil
}

// ExampleUploadAndSend demonstrates uploading a blob and sending a message.
func ExampleUploadAndSend(
	ctx context.Context,
	blobClient blobclient.BlobClient,
	serviceBusClient servicebusclient.ServiceBusClient,
	container, blobName, queueName string,
	data io.Reader,
) error {
	// Upload to blob storage
	url, err := blobClient.Upload(ctx, container, blobName, data, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}
	
	// Send message to Service Bus with the blob URL
	messageBody := []byte(fmt.Sprintf(`{"blobUrl": "%s", "blobName": "%s"}`, url, blobName))
	_, err = serviceBusClient.Send(ctx, queueName, messageBody,
		servicebusclient.WithContentType("application/json"),
		servicebusclient.WithProperties(map[string]interface{}{
			"blobContainer": container,
			"blobName":      blobName,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	
	return nil
}

// ExampleWireComponents demonstrates how to wire all components together.
func ExampleWireComponents() error {
	// Load configuration
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Create logger
	logger, err := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer logging.Sync(logger)
	
	// Create blob client
	blobClient, err := ExampleBlobClientUsage(cfg, logger)
	if err != nil {
		return err
	}
	
	// Create Service Bus client
	serviceBusClient, err := ExampleServiceBusClientUsage(cfg, logger)
	if err != nil {
		return err
	}
	
	// Use clients...
	_ = blobClient
	_ = serviceBusClient
	
	return nil
}

