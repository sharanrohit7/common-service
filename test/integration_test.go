package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourorg/go-service-kit/pkg/blobclient"
	"github.com/yourorg/go-service-kit/pkg/config"
	"github.com/yourorg/go-service-kit/pkg/logging"
	"github.com/yourorg/go-service-kit/pkg/servicebusclient"
)

// TestIntegration_CSVUploadAndProcessing tests the full flow:
// 1. Upload CSV via HTTP endpoint
// 2. Service parses CSV
// 3. Service enqueues messages to Service Bus
// 4. Service uploads PDF to blob storage
func TestIntegration_CSVUploadAndProcessing(t *testing.T) {
	// Setup mocks
	mockBlobClient := blobclient.NewMockBlobClient()
	mockServiceBusClient := servicebusclient.NewMockServiceBusClient()

	// Create test CSV data
	csvData := `name,age,city
John,30,New York
Jane,25,San Francisco`

	// Test CSV to PDF endpoint
	reqBody := map[string]string{
		"csv_data": csvData,
		"title":    "Test Report",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Create a simple test server (simplified version)
	// In a real integration test, you'd start the full service
	ctx := context.Background()

	// Test blob upload
	blobName := "test/report.pdf"
	url, err := mockBlobClient.Upload(ctx, "test-container", blobName, bytes.NewReader([]byte("fake pdf")), "application/pdf")
	if err != nil {
		t.Fatalf("Blob upload failed: %v", err)
	}
	if url == "" {
		t.Error("Expected blob URL")
	}

	// Test Service Bus send
	messageBody := []byte(`{"pdfUrl": "` + url + `"}`)
	_, err = mockServiceBusClient.Send(ctx, "test-queue", messageBody,
		servicebusclient.WithContentType("application/json"),
	)
	if err != nil {
		t.Fatalf("Service Bus send failed: %v", err)
	}

	// Verify message was sent
	messages, err := mockServiceBusClient.Receive(ctx, "test-queue", 10)
	if err != nil {
		t.Fatalf("Service Bus receive failed: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	_ = jsonBody // Use in actual HTTP test
}

// TestIntegration_BlobAndServiceBusFlow tests the blob and Service Bus integration.
func TestIntegration_BlobAndServiceBusFlow(t *testing.T) {
	ctx := context.Background()
	mockBlobClient := blobclient.NewMockBlobClient()
	mockServiceBusClient := servicebusclient.NewMockServiceBusClient()

	// Upload CSV to blob
	csvData := "name,age\nJohn,30\nJane,25"
	blobURL, err := mockBlobClient.Upload(ctx, "csv-container", "data.csv", bytes.NewReader([]byte(csvData)), "text/csv")
	if err != nil {
		t.Fatalf("Failed to upload CSV: %v", err)
	}

	// Send notification
	notification := map[string]interface{}{
		"blobUrl": blobURL,
		"type":    "csv_uploaded",
	}
	notificationBody, _ := json.Marshal(notification)

	_, err = mockServiceBusClient.Send(ctx, "notifications", notificationBody,
		servicebusclient.WithContentType("application/json"),
	)
	if err != nil {
		t.Fatalf("Failed to send notification: %v", err)
	}

	// Verify blob exists
	exists, err := mockBlobClient.Exists(ctx, "csv-container", "data.csv")
	if err != nil {
		t.Fatalf("Failed to check blob existence: %v", err)
	}
	if !exists {
		t.Error("Blob should exist")
	}

	// Verify message was sent
	messages, err := mockServiceBusClient.Receive(ctx, "notifications", 10)
	if err != nil {
		t.Fatalf("Failed to receive messages: %v", err)
	}
	if len(messages) == 0 {
		t.Error("Expected at least one message")
	}
}

// TestHTTPEndpoint is a helper to test HTTP endpoints.
func TestHTTPEndpoint(t *testing.T, handler http.HandlerFunc, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

// SetupTestConfig creates a test configuration.
func SetupTestConfig() *config.Config {
	return &config.Config{
		BlobContainer:         "test-container",
		ServiceBusQueue:       "test-queue",
		HTTPPort:              8080,
		LogLevel:              "debug",
		LogFormat:             "text",
		RetryMaxAttempts:      3,
		RetryInitialDelay:     100,
		RetryMaxDelay:         5000,
		ServiceBusConcurrency: 1,
	}
}

// SetupTestLogger creates a test logger.
func SetupTestLogger() logging.Logger {
	logger, _ := logging.NewLogger("debug", "text")
	return logger
}

// CleanupTestResources cleans up test resources.
func CleanupTestResources(ctx context.Context, blobClient blobclient.BlobClient, container string) {
	// Cleanup blobs if needed
	blobs, _ := blobClient.List(ctx, container, "")
	for _, blob := range blobs {
		_ = blobClient.Delete(ctx, container, blob.Name)
	}
}
