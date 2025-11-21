package blobclient

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestMockBlobClient_Upload(t *testing.T) {
	client := NewMockBlobClient()
	ctx := context.Background()
	
	data := strings.NewReader("test data")
	url, err := client.Upload(ctx, "test-container", "test-blob", data, "text/plain")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	if url == "" {
		t.Error("Expected URL to be returned")
	}
}

func TestMockBlobClient_Get(t *testing.T) {
	client := NewMockBlobClient()
	ctx := context.Background()
	
	// Upload first
	data := strings.NewReader("test data")
	_, err := client.Upload(ctx, "test-container", "test-blob", data, "text/plain")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	// Get it back
	reader, err := client.Get(ctx, "test-container", "test-blob")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()
	
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	
	if string(content) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(content))
	}
}

func TestMockBlobClient_Exists(t *testing.T) {
	client := NewMockBlobClient()
	ctx := context.Background()
	
	exists, err := client.Exists(ctx, "test-container", "test-blob")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Expected blob to not exist")
	}
	
	// Upload and check again
	data := strings.NewReader("test data")
	_, err = client.Upload(ctx, "test-container", "test-blob", data, "text/plain")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	exists, err = client.Exists(ctx, "test-container", "test-blob")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected blob to exist")
	}
}

func TestMockBlobClient_List(t *testing.T) {
	client := NewMockBlobClient()
	ctx := context.Background()
	
	// Upload some blobs
	data1 := strings.NewReader("data1")
	data2 := strings.NewReader("data2")
	
	_, err := client.Upload(ctx, "test-container", "blob1", data1, "text/plain")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	_, err = client.Upload(ctx, "test-container", "blob2", data2, "text/plain")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	blobs, err := client.List(ctx, "test-container", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	
	if len(blobs) != 2 {
		t.Errorf("Expected 2 blobs, got %d", len(blobs))
	}
}

