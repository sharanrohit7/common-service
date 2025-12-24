package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// NewRelicClient wraps the New Relic agent.
type NewRelicClient struct {
	app         *newrelic.Application
	logger      logging.Logger
	serviceName string
	enabled     bool
}

// NewRelicConfig holds New Relic configuration.
type NewRelicConfig struct {
	LicenseKey  string
	AppName     string
	ServiceName string // Name of the service (e.g., "api_gateway", "auth_service")
	Enabled     bool
}

// NewNewRelicClient creates a new New Relic client.
func NewNewRelicClient(cfg NewRelicConfig, logger logging.Logger) (*NewRelicClient, error) {
	if !cfg.Enabled || cfg.LicenseKey == "" {
		logger.Info("New Relic disabled or license key not provided")
		return &NewRelicClient{
			enabled:     false,
			logger:      logger,
			serviceName: cfg.ServiceName,
		}, nil
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(cfg.AppName),
		newrelic.ConfigLicense(cfg.LicenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create New Relic application: %w", err)
	}

	logger.Info("New Relic client initialized",
		logging.NewField("app_name", cfg.AppName),
		logging.NewField("service", cfg.ServiceName),
	)

	return &NewRelicClient{
		app:         app,
		logger:      logger,
		serviceName: cfg.ServiceName,
		enabled:     true,
	}, nil
}

// RecordTransaction records a transaction in New Relic.
func (n *NewRelicClient) RecordTransaction(ctx context.Context, name string, durationMs int64, statusCode int, traceID, requestID string) {
	if !n.enabled || n.app == nil {
		return
	}

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		txn = n.app.StartTransaction(name)
		defer txn.End()
	}

	txn.SetName(name)
	txn.AddAttribute("trace_id", traceID)
	txn.AddAttribute("request_id", requestID)
	txn.AddAttribute("status_code", statusCode)
	txn.AddAttribute("duration_ms", durationMs)
	txn.AddAttribute("service", n.serviceName)

	if statusCode >= 500 {
		txn.NoticeError(fmt.Errorf("HTTP %d", statusCode))
	}
}

// RecordCustomEvent records a custom event in New Relic.
func (n *NewRelicClient) RecordCustomEvent(eventType string, attributes map[string]interface{}) {
	if !n.enabled || n.app == nil {
		return
	}

	n.app.RecordCustomEvent(eventType, attributes)
}

// RecordSlowRequest records a slow request event.
// Implements middleware.TelemetryClient interface.
func (n *NewRelicClient) RecordSlowRequest(ctx interface{}, path string, durationMs int64, traceID, requestID string) {
	if !n.enabled {
		return
	}

	attributes := map[string]interface{}{
		"event_type":  "SlowRequest",
		"service":     n.serviceName,
		"path":        path,
		"duration_ms": durationMs,
		"trace_id":    traceID,
		"request_id":  requestID,
	}

	n.RecordCustomEvent("SlowRequest", attributes)

	// Convert ctx to context.Context if possible
	var ctxContext context.Context
	if c, ok := ctx.(context.Context); ok {
		ctxContext = c
		n.RecordTransaction(ctxContext, path, durationMs, 200, traceID, requestID)
	}

	n.logger.Warn("Slow request detected",
		logging.NewField("path", path),
		logging.NewField("duration_ms", durationMs),
		logging.NewField("trace_id", traceID),
		logging.NewField("request_id", requestID),
	)
}

// RecordError records an error event.
// Implements middleware.TelemetryClient interface.
func (n *NewRelicClient) RecordError(ctx interface{}, path, errorMsg string, statusCode int, traceID, requestID string) {
	if !n.enabled {
		return
	}

	attributes := map[string]interface{}{
		"event_type":  "ServiceError",
		"service":     n.serviceName,
		"path":        path,
		"error":       errorMsg,
		"status_code": statusCode,
		"trace_id":    traceID,
		"request_id":  requestID,
	}

	n.RecordCustomEvent("ServiceError", attributes)

	// Convert ctx to context.Context if possible
	var ctxContext context.Context
	if c, ok := ctx.(context.Context); ok {
		ctxContext = c
		n.RecordTransaction(ctxContext, path, 0, statusCode, traceID, requestID)
	}

	n.logger.Error("Service error detected",
		logging.NewField("path", path),
		logging.NewField("error", errorMsg),
		logging.NewField("status_code", statusCode),
		logging.NewField("trace_id", traceID),
		logging.NewField("request_id", requestID),
	)
}

// Shutdown gracefully shuts down the New Relic client.
func (n *NewRelicClient) Shutdown(timeoutMs int) {
	if n.enabled && n.app != nil {
		n.app.Shutdown(time.Duration(timeoutMs) * time.Millisecond)
	}
}
