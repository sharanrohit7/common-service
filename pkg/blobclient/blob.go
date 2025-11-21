package blobclient

import (
	"context"
	"io"
)

// BlobClient defines the interface for blob storage operations.
type BlobClient interface {
	// Upload uploads data to blob storage and returns the URL.
	Upload(ctx context.Context, container, blobName string, data io.Reader, contentType string) (url string, err error)
	
	// Get retrieves a blob from storage.
	Get(ctx context.Context, container, blobName string) (io.ReadCloser, error)
	
	// Delete deletes a blob from storage.
	Delete(ctx context.Context, container, blobName string) error
	
	// Exists checks if a blob exists.
	Exists(ctx context.Context, container, blobName string) (bool, error)
	
	// List lists blobs in a container with optional prefix.
	List(ctx context.Context, container, prefix string) ([]BlobInfo, error)
}

// BlobInfo contains information about a blob.
type BlobInfo struct {
	Name         string
	Size         int64
	ContentType  string
	LastModified string
	URL          string
}

// UploadOptions contains optional parameters for upload operations.
type UploadOptions struct {
	ContentType string
	AccessTier  string // Hot, Cool, Archive
	Metadata    map[string]string
}

