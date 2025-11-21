package blobclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// MockBlobClient is an in-memory implementation of BlobClient for testing.
type MockBlobClient struct {
	blobs map[string]map[string][]byte // container -> blobName -> data
	mu    sync.RWMutex
}

// NewMockBlobClient creates a new mock blob client.
func NewMockBlobClient() *MockBlobClient {
	return &MockBlobClient{
		blobs: make(map[string]map[string][]byte),
	}
}

// Upload uploads data to the mock storage.
func (m *MockBlobClient) Upload(ctx context.Context, container, blobName string, data io.Reader, contentType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.blobs[container] == nil {
		m.blobs[container] = make(map[string][]byte)
	}
	
	blobData, err := io.ReadAll(data)
	if err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}
	
	m.blobs[container][blobName] = blobData
	url := fmt.Sprintf("mock://%s/%s", container, blobName)
	
	return url, nil
}

// Get retrieves a blob from the mock storage.
func (m *MockBlobClient) Get(ctx context.Context, container, blobName string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.blobs[container] == nil {
		return nil, fmt.Errorf("container not found: %s", container)
	}
	
	data, exists := m.blobs[container][blobName]
	if !exists {
		return nil, fmt.Errorf("blob not found: %s/%s", container, blobName)
	}
	
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Delete deletes a blob from the mock storage.
func (m *MockBlobClient) Delete(ctx context.Context, container, blobName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.blobs[container] == nil {
		return fmt.Errorf("container not found: %s", container)
	}
	
	delete(m.blobs[container], blobName)
	return nil
}

// Exists checks if a blob exists in the mock storage.
func (m *MockBlobClient) Exists(ctx context.Context, container, blobName string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.blobs[container] == nil {
		return false, nil
	}
	
	_, exists := m.blobs[container][blobName]
	return exists, nil
}

// List lists blobs in a container with optional prefix.
func (m *MockBlobClient) List(ctx context.Context, container, prefix string) ([]BlobInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.blobs[container] == nil {
		return []BlobInfo{}, nil
	}
	
	var blobs []BlobInfo
	for name, data := range m.blobs[container] {
		if prefix == "" || len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			blobs = append(blobs, BlobInfo{
				Name:        name,
				Size:        int64(len(data)),
				ContentType: "application/octet-stream",
				URL:         fmt.Sprintf("mock://%s/%s", container, name),
			})
		}
	}
	
	return blobs, nil
}

