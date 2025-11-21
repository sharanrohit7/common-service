package httpservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// Server wraps a Gin server with configuration and middleware.
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	logger     logging.Logger
	port       int
}

// ServerConfig configures the HTTP server.
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Logger       logging.Logger
	// Security Configuration
	RateLimitRPS   float64
	RateLimitBurst int
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxBodySize    int64 // Maximum request body size in bytes (default: 10MB)
}

// NewServer creates a new HTTP server with the provided configuration and handlers.
func NewServer(cfg ServerConfig, handlers ...Handler) (*Server, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Apply default middleware
	router.Use(RecoveryMiddleware(cfg.Logger))

	// Request size limit (if configured)
	if cfg.MaxBodySize > 0 {
		router.Use(RequestSizeLimitMiddleware(cfg.MaxBodySize, cfg.Logger))
	}

	router.Use(BodyLoggingMiddleware(cfg.Logger)) // Comprehensive request/response logging
	router.Use(RequestIDMiddleware())
	router.Use(SecurityHeadersMiddleware())
	router.Use(XSSProtectionMiddleware())                       // XSS protection
	router.Use(EnhancedSQLInjectionCheckMiddleware(cfg.Logger)) // Enhanced SQL injection detection

	// HTTP Method Whitelist (defense-in-depth)
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	router.Use(HTTPMethodWhitelistMiddleware(allowedMethods, cfg.Logger))

	// Configure CORS
	corsCfg := CORSConfig{
		AllowedOrigins: cfg.AllowedOrigins,
		AllowedMethods: cfg.AllowedMethods,
		AllowedHeaders: cfg.AllowedHeaders,
	}
	// Set defaults if not provided
	if len(corsCfg.AllowedOrigins) == 0 {
		corsCfg.AllowedOrigins = []string{"*"}
	}
	router.Use(CORSMiddleware(corsCfg))

	// Configure Rate Limiting if enabled
	if cfg.RateLimitRPS > 0 {
		router.Use(RateLimitMiddleware(RateLimitConfig{
			RPS:   cfg.RateLimitRPS,
			Burst: cfg.RateLimitBurst,
		}))
	}

	// Register handlers
	for _, handler := range handlers {
		handler.Register(router)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		router:     router,
		httpServer: httpServer,
		logger:     cfg.Logger,
		port:       cfg.Port,
	}, nil
}

// Handler defines an interface for registering HTTP handlers.
type Handler interface {
	Register(router *gin.Engine)
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", logging.NewField("port", s.port))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// StartTLS starts the HTTP server with TLS.
func (s *Server) StartTLS(certFile, keyFile string) error {
	s.logger.Info("Starting HTTP server with TLS", logging.NewField("port", s.port))

	if err := s.httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the underlying Gin router for advanced configuration.
func (s *Server) Router() *gin.Engine {
	return s.router
}
