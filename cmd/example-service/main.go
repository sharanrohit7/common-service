package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/blobclient"
	"github.com/yourorg/go-service-kit/pkg/config"
	"github.com/yourorg/go-service-kit/pkg/csvutil"
	"github.com/yourorg/go-service-kit/pkg/errors"
	"github.com/yourorg/go-service-kit/pkg/httpservice"
	"github.com/yourorg/go-service-kit/pkg/logging"
	"github.com/yourorg/go-service-kit/pkg/pdfutil"
	"github.com/yourorg/go-service-kit/pkg/servicebusclient"
)

type App struct {
	config          *config.Config
	logger          logging.Logger
	blobClient      blobclient.BlobClient
	serviceBusClient servicebusclient.ServiceBusClient
	server          *httpservice.Server
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	
	// Create logger
	logger, err := logging.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logging.Sync(logger)
	
	logger.Info("Starting example service", logging.NewField("version", cfg.AppVersion))
	
	// Create blob client (use mock for local development)
	var blobClient blobclient.BlobClient
	if cfg.BlobStorageAccountName == "" {
		logger.Info("Using mock blob client (no account name configured)")
		blobClient = blobclient.NewMockBlobClient()
	} else {
		blobClient, err = blobclient.NewAzureBlobClient(
			cfg.BlobStorageAccountName,
			cfg.BlobStorageAccountKey,
			false,
			logger,
		)
		if err != nil {
			logger.Error("Failed to create blob client", logging.NewField("error", err))
			os.Exit(1)
		}
	}
	
	// Create Service Bus client (use mock for local development)
	var serviceBusClient servicebusclient.ServiceBusClient
	if cfg.ServiceBusNamespace == "" {
		logger.Info("Using mock Service Bus client (no namespace configured)")
		serviceBusClient = servicebusclient.NewMockServiceBusClient()
	} else {
		serviceBusClient, err = servicebusclient.NewAzureServiceBusClient(
			cfg.ServiceBusNamespace,
			cfg.ServiceBusKeyName,
			cfg.ServiceBusKeyValue,
			false,
			logger,
		)
		if err != nil {
			logger.Error("Failed to create Service Bus client", logging.NewField("error", err))
			os.Exit(1)
		}
	}
	
	// Create app
	app := &App{
		config:           cfg,
		logger:           logger,
		blobClient:       blobClient,
		serviceBusClient: serviceBusClient,
	}
	
	// Create HTTP server
	server, err := httpservice.NewServer(httpservice.ServerConfig{
		Port:         cfg.HTTPPort,
		ReadTimeout:  time.Duration(cfg.HTTPReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTPWriteTimeout) * time.Second,
		IdleTimeout: time.Duration(cfg.HTTPIdleTimeout) * time.Second,
		Logger:       logger,
	}, app)
	if err != nil {
		logger.Error("Failed to create server", logging.NewField("error", err))
		os.Exit(1)
	}
	
	app.server = server
	
	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Server error", logging.NewField("error", err))
			os.Exit(1)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Info("Shutting down server")
	
	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", logging.NewField("error", err))
	}
}

// Register implements the httpservice.Handler interface.
func (a *App) Register(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/csv-to-pdf", a.handleCSVToPDF)
		api.POST("/upload-csv", a.handleUploadCSV)
	}
}

// CSVToPDFRequest represents the request for CSV to PDF conversion.
type CSVToPDFRequest struct {
	CSVData string `json:"csv_data" binding:"required"`
	Title   string `json:"title"`
}

// handleCSVToPDF handles CSV to PDF conversion.
func (a *App) handleCSVToPDF(c *gin.Context) {
	var req CSVToPDFRequest
	if !httpservice.ValidateJSON(c, &req) {
		return
	}
	
	logger := a.logger.With(
		logging.NewField("operation", "csv_to_pdf"),
		logging.NewField("path", c.Request.URL.Path),
	)
	
	// Parse CSV
	parser := csvutil.NewParser(csvutil.DefaultParserConfig())
	csvReader := bytes.NewReader([]byte(req.CSVData))
	
	var headers []string
	var rows [][]string
	
	err := parser.Parse(csvReader, func(rowNum int, hdrs []string, row []string) error {
		if rowNum == 1 {
			headers = hdrs
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		logger.Error("Failed to parse CSV", logging.NewField("error", err))
		httpservice.HandleError(c, errors.NewValidationError("Invalid CSV: "+err.Error()))
		return
	}
	
	// Generate PDF
	title := req.Title
	if title == "" {
		title = "CSV Report"
	}
	
	pdfBytes, err := pdfutil.GenerateReport(title, headers, rows)
	if err != nil {
		logger.Error("Failed to generate PDF", logging.NewField("error", err))
		httpservice.HandleError(c, errors.NewInternalError("Failed to generate PDF: "+err.Error()))
		return
	}
	
	// Upload to blob storage
	blobName := fmt.Sprintf("reports/%d.pdf", time.Now().Unix())
	url, err := a.blobClient.Upload(
		c.Request.Context(),
		a.config.BlobContainer,
		blobName,
		bytes.NewReader(pdfBytes),
		"application/pdf",
	)
	if err != nil {
		logger.Error("Failed to upload PDF", logging.NewField("error", err))
		httpservice.HandleError(c, errors.NewInternalError("Failed to upload PDF: "+err.Error()))
		return
	}
	
	// Send notification to Service Bus
	messageBody := []byte(fmt.Sprintf(`{"pdfUrl": "%s", "title": "%s"}`, url, title))
	_, err = a.serviceBusClient.Send(
		c.Request.Context(),
		a.config.ServiceBusQueue,
		messageBody,
		servicebusclient.WithContentType("application/json"),
	)
	if err != nil {
		logger.Warn("Failed to send notification", logging.NewField("error", err))
		// Don't fail the request if notification fails
	}
	
	// Return PDF as response
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// handleUploadCSV handles CSV file upload, parses it, and enqueues messages.
func (a *App) handleUploadCSV(c *gin.Context) {
	file, err := c.FormFile("csv_file")
	if err != nil {
		httpservice.HandleError(c, errors.NewBadRequestError("CSV file is required"))
		return
	}
	
	logger := a.logger.With(
		logging.NewField("operation", "upload_csv"),
		logging.NewField("filename", file.Filename),
	)
	
	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		logger.Error("Failed to open file", logging.NewField("error", err))
		httpservice.HandleError(c, errors.NewInternalError("Failed to open file: "+err.Error()))
		return
	}
	defer src.Close()
	
	// Upload to blob storage
	blobName := fmt.Sprintf("csv/%s", file.Filename)
	url, err := a.blobClient.Upload(
		c.Request.Context(),
		a.config.BlobContainer,
		blobName,
		src,
		"text/csv",
	)
	if err != nil {
		logger.Error("Failed to upload CSV", logging.NewField("error", err))
		httpservice.HandleError(c, errors.NewInternalError("Failed to upload CSV: "+err.Error()))
		return
	}
	
	// Re-read file for parsing (reset to beginning)
	src.Seek(0, io.SeekStart)
	
	// Parse CSV
	parser := csvutil.NewParser(csvutil.DefaultParserConfig())
	rowCount := 0
	
	err = parser.Parse(src, func(rowNum int, headers []string, row []string) error {
		rowCount++
		
		// Send each row as a message to Service Bus
		// In production, you might batch these
		rowData := fmt.Sprintf(`{"row": %d, "data": %v, "blobUrl": "%s"}`, rowNum, row, url)
		_, sendErr := a.serviceBusClient.Send(
			c.Request.Context(),
			a.config.ServiceBusQueue,
			[]byte(rowData),
			servicebusclient.WithContentType("application/json"),
			servicebusclient.WithProperties(map[string]interface{}{
				"rowNumber": rowNum,
				"blobUrl":   url,
			}),
		)
		if sendErr != nil {
			logger.Warn("Failed to send message", logging.NewField("error", sendErr), logging.NewField("row", rowNum))
		}
		
		return nil
	})
	if err != nil {
		logger.Error("Failed to parse CSV", logging.NewField("error", err))
		httpservice.HandleError(c, errors.NewValidationError("Invalid CSV: "+err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"blob_url":  url,
		"row_count": rowCount,
		"message":   "CSV uploaded and processed",
	})
}

