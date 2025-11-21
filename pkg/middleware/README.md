# Middleware Package

This package provides reusable middleware components for Go microservices built with Gin. All middleware is designed to be DI-friendly and configurable.

## Available Middleware

### 1. TracingMiddleware

Generates or extracts trace ID and attaches it to context.

```go
router.Use(middleware.TracingMiddleware(logger, "service-name"))
```

**Features:**
- Extracts `X-Trace-ID` from request header if present
- Generates new UUID v4 if header is missing
- Attaches trace ID to context and Gin context
- Adds trace ID to response header

### 2. ServiceRequestIDMiddleware

Generates a service-specific request ID for internal operations.

```go
router.Use(middleware.ServiceRequestIDMiddleware("X-Service-Request-ID"))
```

**Features:**
- Generates UUID v4 request ID
- Attaches to context and Gin context
- Adds to response header with custom header name

### 3. ContextLoggerMiddleware

Attaches a contextual logger to request context with pre-populated fields.

```go
router.Use(middleware.ContextLoggerMiddleware(logger, "service-name"))
```

**Features:**
- Creates logger with `service`, `trace_id`, and `request_id` fields
- Attaches to request context
- Handlers can retrieve with `logging.FromContext(ctx)`

### 4. ErrorHandlerMiddleware

Centralized error handling that converts `AppError` to HTTP responses.

```go
router.Use(middleware.ErrorHandlerMiddleware(logger))
```

**Features:**
- Catches errors set via `middleware.SetError(c, err)`
- Converts `AppError` to standardized JSON response
- Sets `X-Service-Handled` header when appropriate
- Logs errors with contextual information

**Usage in handlers:**
```go
func (h *Handler) GetResource(c *gin.Context) {
    // Business logic
    if err != nil {
        middleware.SetError(c, errors.NewNotFoundError("resource not found"))
        return
    }
    // Success response
    c.JSON(http.StatusOK, resource)
}
```

### 5. SlowRequestMiddleware

Detects slow requests and triggers alerts (New Relic, Slack).

```go
router.Use(middleware.SlowRequestMiddleware(
    slowThresholdMs,  // int64 - threshold in milliseconds
    telemetryClient,   // Implements TelemetryClient interface
    slackClient,       // Implements SlackClient interface
    logger,
))
```

**Features:**
- Measures request duration
- Triggers alerts if duration > threshold
- Respects `X-Service-Handled` header to avoid duplicate alerts
- Records slow requests and errors to telemetry

**TelemetryClient Interface:**
```go
type TelemetryClient interface {
    RecordSlowRequest(ctx interface{}, path string, durationMs int64, traceID, requestID string)
    RecordError(ctx interface{}, path, errorMsg string, statusCode int, traceID, requestID string)
}
```

**SlackClient Interface:**
```go
type SlackClient interface {
    SendSlowRequestAlert(ctx interface{}, path string, durationMs int64, traceID, requestID string) error
    SendErrorAlert(ctx interface{}, path, errorMsg string, statusCode int, traceID, requestID string) error
}
```

## Middleware Chain Order

The order of middleware matters. Recommended order:

1. **TracingMiddleware** - Must be first to generate/extract trace ID
2. **ServiceRequestIDMiddleware** - Generates request ID (needs trace ID)
3. **ContextLoggerMiddleware** - Creates contextual logger (needs trace ID and request ID)
4. **ErrorHandlerMiddleware** - Handles errors (needs contextual logger)
5. **SlowRequestMiddleware** - Detects slow requests (must be last to measure full duration)

## Example: Complete Middleware Setup

```go
func main() {
    logger, _ := logging.NewLogger("info", "json")
    
    // Initialize telemetry clients
    newRelicClient := telemetry.NewNewRelicClient(...)
    slackClient := telemetry.NewSlackClient(...)
    
    server, _ := httpservice.NewServer(serverConfig, handlers...)
    router := server.Router()
    
    // Wire middleware chain
    router.Use(middleware.TracingMiddleware(logger, "my-service"))
    router.Use(middleware.ServiceRequestIDMiddleware("X-Service-Request-ID"))
    router.Use(middleware.ContextLoggerMiddleware(logger, "my-service"))
    router.Use(middleware.ErrorHandlerMiddleware(logger))
    router.Use(middleware.SlowRequestMiddleware(
        cfg.SlowMs,
        newRelicClient,
        slackClient,
        logger,
    ))
    
    server.Start()
}
```

## Writing Thin Handlers

With middleware handling cross-cutting concerns, handlers should focus only on business logic:

```go
type MyHandler struct {
    // Only business dependencies, no logger/telemetry
    repository Repository
}

func (h *MyHandler) GetResource(c *gin.Context) {
    // Get contextual logger from context
    logger := logging.FromContext(c.Request.Context())
    
    // Business logic
    id := c.Param("id")
    resource, err := h.repository.Get(id)
    if err != nil {
        // Use middleware.SetError for error handling
        middleware.SetError(c, errors.NewNotFoundError("resource not found"))
        return
    }
    
    // Log business events (optional, middleware handles request logging)
    logger.Info("Resource retrieved", logging.NewField("resource_id", id))
    
    // Return success
    c.JSON(http.StatusOK, resource)
}
```

**Benefits:**
- Handlers are testable without logger/telemetry mocks
- Business logic is separated from infrastructure concerns
- Consistent error handling across all handlers
- Automatic logging and telemetry for all requests

