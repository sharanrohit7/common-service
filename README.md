# go-service-kit

A production-ready Go shared library for building microservices with Azure cloud services integration. This kit provides reusable, DI-friendly building blocks for blob storage, messaging, HTTP services, CSV processing, PDF generation, and more.

## Features

- **Azure Integration**: Blob Storage and Service Bus clients with pluggable interfaces
- **HTTP Service**: Gin-based HTTP server with middleware (logging, request ID, recovery, validation)
- **Structured Logging**: Centralized logging with zap, context-aware logging
- **Error Handling**: Typed errors with HTTP status code mapping
- **CSV Utilities**: Streaming CSV parser with validation hooks
- **PDF Generation**: PDF creation utilities using gofpdf
- **Database Interface**: Pluggable database interface with Postgres implementation
- **Retry Logic**: Exponential backoff retry utilities
- **Configuration**: Flexible config loading from environment variables, JSON, or YAML
- **Testing**: Mock implementations for all external dependencies

## Quick Start

### Installation

```bash
go get github.com/yourorg/go-service-kit
```

### Basic Usage

```go
package main

import (
    "context"
    "github.com/yourorg/go-service-kit/pkg/config"
    "github.com/yourorg/go-service-kit/pkg/logging"
    "github.com/yourorg/go-service-kit/pkg/blobclient"
    "github.com/yourorg/go-service-kit/pkg/servicebusclient"
    "github.com/yourorg/go-service-kit/pkg/httpservice"
)

func main() {
    // Load configuration
    cfg, _ := config.LoadConfigFromEnv()
    
    // Create logger
    logger, _ := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)
    defer logging.Sync(logger)
    
    // Create blob client
    blobClient, _ := blobclient.NewAzureBlobClient(
        cfg.BlobStorageAccountName,
        cfg.BlobStorageAccountKey,
        false, // useManagedIdentity
        logger,
    )
    
    // Create Service Bus client
    serviceBusClient, _ := servicebusclient.NewAzureServiceBusClient(
        cfg.ServiceBusNamespace,
        cfg.ServiceBusKeyName,
        cfg.ServiceBusKeyValue,
        false, // useManagedIdentity
        logger,
    )
    
    // Create HTTP server
    server, _ := httpservice.NewServer(httpservice.ServerConfig{
        Port:    cfg.HTTPPort,
        Logger:  logger,
    }, &MyHandler{})
    
    // Start server
    server.Start()
}
```

## Configuration

Configuration can be loaded from environment variables, JSON, or YAML files.

### Environment Variables

```bash
# Blob Storage
export BLOB_STORAGE_ACCOUNT_NAME="mystorageaccount"
export BLOB_STORAGE_ACCOUNT_KEY="your-key"
export BLOB_CONTAINER="my-container"

# Service Bus
export SERVICE_BUS_NAMESPACE="mynamespace"
export SERVICE_BUS_KEY_NAME="RootManageSharedAccessKey"
export SERVICE_BUS_KEY_VALUE="your-key"
export SERVICE_BUS_QUEUE="my-queue"

# HTTP Server
export HTTP_PORT=8080
export HTTP_READ_TIMEOUT=30
export HTTP_WRITE_TIMEOUT=30

# Logging
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

### JSON/YAML Config File

```json
{
  "blob": {
    "storage_account_name": "mystorageaccount",
    "container": "my-container"
  },
  "service_bus": {
    "namespace": "mynamespace",
    "queue": "my-queue"
  },
  "http": {
    "port": 8080
  }
}
```

```go
cfg, err := config.LoadConfigFromFile("config.yaml")
```

## Package Overview

### pkg/blobclient

Blob storage client with Azure implementation and mock for testing.

```go
// Create client
client, _ := blobclient.NewAzureBlobClient(accountName, accountKey, false, logger)

// Upload
url, _ := client.Upload(ctx, "container", "blob.txt", data, "text/plain")

// Get
reader, _ := client.Get(ctx, "container", "blob.txt")
defer reader.Close()

// For testing
mockClient := blobclient.NewMockBlobClient()
```

### pkg/servicebusclient

Service Bus client with Azure implementation, consumer, and mock.

```go
// Create client
client, _ := servicebusclient.NewAzureServiceBusClient(namespace, keyName, keyValue, false, logger)

// Send message
messageID, _ := client.Send(ctx, "my-queue", []byte("message"),
    servicebusclient.WithContentType("application/json"),
)

// Receive messages
messages, _ := client.Receive(ctx, "my-queue", 10)

// Consumer with concurrency
consumer, _ := servicebusclient.NewConsumer(azureClient, config, handler)
consumer.Start(ctx)
```

### pkg/httpservice

HTTP server with Gin, middleware, and validation.

```go
// Create server
server, _ := httpservice.NewServer(httpservice.ServerConfig{
    Port:   8080,
    Logger: logger,
}, &MyHandler{})

// Handler implementation
type MyHandler struct{}

func (h *MyHandler) Register(router *gin.Engine) {
    router.POST("/api/data", h.handleData)
}

func (h *MyHandler) handleData(c *gin.Context) {
    var req MyRequest
    if !httpservice.ValidateJSON(c, &req) {
        return
    }
    httpservice.SuccessResponse(c, gin.H{"status": "ok"})
}
```

### pkg/csvutil

CSV parsing with validation and streaming support.

```go
parser := csvutil.NewParser(csvutil.DefaultParserConfig())
parser.Parse(reader, func(rowNum int, headers []string, row []string) error {
    // Process row
    return nil
})
```

### pkg/pdfutil

PDF generation utilities.

```go
// Generate report
pdfBytes, _ := pdfutil.GenerateReport("Title", headers, rows)

// Generate payslip
pdfBytes, _ := pdfutil.GeneratePayslip("John Doe", "123", "2024-01", 5000, 500, 4500)
```

### pkg/logging

Structured logging with zap.

```go
logger, _ := logging.NewLogger("info", "json")
logger.Info("Message", logging.NewField("key", "value"))

// Context-aware logging
ctx = logging.WithLogger(ctx, logger)
logger := logging.FromContext(ctx)
```

### pkg/errors

Typed errors with HTTP status mapping.

```go
// Create error
err := errors.NewBadRequestError("Invalid input")

// Convert to HTTP response
c.JSON(err.HTTPStatus, gin.H{"error": err.Message})
```

## Running the Example Service

The example service demonstrates CSV upload, PDF generation, blob storage, and Service Bus messaging.

### Prerequisites

- Go 1.21+
- (Optional) Azure Storage Account and Service Bus namespace for production
- (Optional) Azurite for local blob storage emulation

### Local Development (with Mocks)

```bash
# Set environment variables (optional - mocks will be used if not set)
export HTTP_PORT=8080
export LOG_LEVEL=debug
export LOG_FORMAT=text

# Run the service
go run ./cmd/example-service
```

### With Azure Services

```bash
export BLOB_STORAGE_ACCOUNT_NAME="your-account"
export BLOB_STORAGE_ACCOUNT_KEY="your-key"
export BLOB_CONTAINER="my-container"
export SERVICE_BUS_NAMESPACE="your-namespace"
export SERVICE_BUS_KEY_NAME="RootManageSharedAccessKey"
export SERVICE_BUS_KEY_VALUE="your-key"
export SERVICE_BUS_QUEUE="my-queue"
export HTTP_PORT=8080

go run ./cmd/example-service
```

### Using Azurite (Local Blob Storage Emulator)

```bash
# Install Azurite
npm install -g azurite

# Run Azurite
azurite --silent --location ~/azurite

# Use connection string
export BLOB_STORAGE_ACCOUNT_NAME="devstoreaccount1"
export BLOB_STORAGE_ACCOUNT_KEY="Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6rqj1nZA=="
# Note: Update blob client to use http://127.0.0.1:10000 for local emulator
```

### Example API Calls

```bash
# Convert CSV to PDF
curl -X POST http://localhost:8080/api/v1/csv-to-pdf \
  -H "Content-Type: application/json" \
  -d '{
    "csv_data": "name,age\nJohn,30\nJane,25",
    "title": "Employee Report"
  }' \
  --output report.pdf

# Upload CSV file
curl -X POST http://localhost:8080/api/v1/upload-csv \
  -F "csv_file=@data.csv"
```

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint
```

## Building

```bash
# Build example service
make build

# Build Docker image
make docker-build
```

## Architecture

The kit follows clean architecture principles:

- **Interfaces First**: All external dependencies are abstracted behind interfaces
- **Dependency Injection**: No global state, all dependencies injected via constructors
- **Pluggable Implementations**: Easy to swap Azure implementations with mocks or alternatives
- **Context Propagation**: All operations support context.Context for cancellation and timeouts
- **Structured Logging**: Centralized logging with context-aware fields

## Local Development Tips

### Using Mocks

For local development without Azure services, use the provided mocks:

```go
// Use mock blob client
blobClient := blobclient.NewMockBlobClient()

// Use mock Service Bus client
serviceBusClient := servicebusclient.NewMockServiceBusClient()
```

### Using Emulators

- **Azurite**: Azure Blob Storage emulator (npm install -g azurite)
- **Azure Service Bus Emulator**: Not officially available, use mocks or Azure Storage Queue as alternative

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make test` and `make lint`
6. Submit a pull request

## License

MIT License - see LICENSE file for details

## Example: Complete Microservice

Here's a complete example showing how to wire everything together:

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/yourorg/go-service-kit/pkg/config"
    "github.com/yourorg/go-service-kit/pkg/logging"
    "github.com/yourorg/go-service-kit/pkg/blobclient"
    "github.com/yourorg/go-service-kit/pkg/servicebusclient"
    "github.com/yourorg/go-service-kit/pkg/httpservice"
)

func main() {
    // Load config
    cfg, _ := config.LoadConfigFromEnv()
    
    // Create logger
    logger, _ := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)
    defer logging.Sync(logger)
    
    // Create clients (use mocks if credentials not provided)
    var blobClient blobclient.BlobClient
    if cfg.BlobStorageAccountName == "" {
        blobClient = blobclient.NewMockBlobClient()
    } else {
        blobClient, _ = blobclient.NewAzureBlobClient(
            cfg.BlobStorageAccountName,
            cfg.BlobStorageAccountKey,
            false,
            logger,
        )
    }
    
    var serviceBusClient servicebusclient.ServiceBusClient
    if cfg.ServiceBusNamespace == "" {
        serviceBusClient = servicebusclient.NewMockServiceBusClient()
    } else {
        serviceBusClient, _ = servicebusclient.NewAzureServiceBusClient(
            cfg.ServiceBusNamespace,
            cfg.ServiceBusKeyName,
            cfg.ServiceBusKeyValue,
            false,
            logger,
        )
    }
    
    // Create HTTP server
    server, _ := httpservice.NewServer(httpservice.ServerConfig{
        Port:         cfg.HTTPPort,
        ReadTimeout:  time.Duration(cfg.HTTPReadTimeout) * time.Second,
        WriteTimeout: time.Duration(cfg.HTTPWriteTimeout) * time.Second,
        IdleTimeout: time.Duration(cfg.HTTPIdleTimeout) * time.Second,
        Logger:       logger,
    }, &MyHandler{
        blobClient:      blobClient,
        serviceBusClient: serviceBusClient,
        logger:          logger,
    })
    
    // Start server
    go server.Start()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    server.Shutdown(ctx)
}
```

## Support

For issues and questions, please open an issue on GitHub.

