package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigSource defines an interface for loading configuration from various sources.
type ConfigSource interface {
	Get(key string) (string, bool)
	GetWithDefault(key, defaultValue string) string
}

// EnvConfigSource loads configuration from environment variables.
type EnvConfigSource struct{}

// Get retrieves an environment variable.
func (e *EnvConfigSource) Get(key string) (string, bool) {
	val := os.Getenv(key)
	return val, val != ""
}

// GetWithDefault retrieves an environment variable or returns a default value.
func (e *EnvConfigSource) GetWithDefault(key, defaultValue string) string {
	if val, ok := e.Get(key); ok {
		return val
	}
	return defaultValue
}

// FileConfigSource loads configuration from a JSON or YAML file.
type FileConfigSource struct {
	data map[string]interface{}
}

// NewFileConfigSource creates a new file-based config source.
// Supports both JSON and YAML files based on file extension.
func NewFileConfigSource(filePath string) (*FileConfigSource, error) {
	data := make(map[string]interface{})
	
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	if strings.HasSuffix(filePath, ".yaml") || strings.HasSuffix(filePath, ".yml") {
		if err := yaml.Unmarshal(fileData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	} else if strings.HasSuffix(filePath, ".json") {
		if err := json.Unmarshal(fileData, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported config file format, use .json, .yaml, or .yml")
	}
	
	return &FileConfigSource{data: data}, nil
}

// Get retrieves a value from the config file using dot notation (e.g., "blob.container").
func (f *FileConfigSource) Get(key string) (string, bool) {
	keys := strings.Split(key, ".")
	var current interface{} = f.data
	
	for _, k := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[k]; exists {
				current = val
			} else {
				return "", false
			}
		} else {
			return "", false
		}
	}
	
	if str, ok := current.(string); ok {
		return str, true
	}
	return fmt.Sprintf("%v", current), true
}

// GetWithDefault retrieves a value from the config file or returns a default.
func (f *FileConfigSource) GetWithDefault(key, defaultValue string) string {
	if val, ok := f.Get(key); ok {
		return val
	}
	return defaultValue
}

// Config holds application configuration.
type Config struct {
	// Blob Storage configuration
	BlobStorageAccountName string
	BlobStorageAccountKey  string
	BlobContainer          string
	BlobAccessTier         string // Hot, Cool, Archive
	
	// Service Bus configuration
	ServiceBusNamespace    string
	ServiceBusKeyName      string
	ServiceBusKeyValue     string
	ServiceBusQueue        string
	ServiceBusTopic        string
	
	// HTTP Server configuration
	HTTPPort               int
	HTTPReadTimeout        int // seconds
	HTTPWriteTimeout       int // seconds
	HTTPIdleTimeout        int // seconds
	
	// Logging configuration
	LogLevel               string // debug, info, warn, error
	LogFormat              string // json, text
	
	// Application configuration
	AppName                string
	AppVersion             string
	Environment            string // dev, staging, prod
	
	// Retry configuration
	RetryMaxAttempts       int
	RetryInitialDelay      int // milliseconds
	RetryMaxDelay          int // milliseconds
	
	// Service Bus Consumer configuration
	ServiceBusConcurrency  int // number of concurrent message handlers
}

// LoadConfig loads configuration from the provided source.
// Environment variables take precedence over file config.
func LoadConfig(source ConfigSource) (*Config, error) {
	cfg := &Config{}
	
	// Helper to get int from config
	getInt := func(key string, defaultValue int) int {
		str := source.GetWithDefault(key, fmt.Sprintf("%d", defaultValue))
		val, err := strconv.Atoi(str)
		if err != nil {
			return defaultValue
		}
		return val
	}
	
	cfg.BlobStorageAccountName = source.GetWithDefault("BLOB_STORAGE_ACCOUNT_NAME", "")
	cfg.BlobStorageAccountKey = source.GetWithDefault("BLOB_STORAGE_ACCOUNT_KEY", "")
	cfg.BlobContainer = source.GetWithDefault("BLOB_CONTAINER", "default-container")
	cfg.BlobAccessTier = source.GetWithDefault("BLOB_ACCESS_TIER", "Hot")
	
	cfg.ServiceBusNamespace = source.GetWithDefault("SERVICE_BUS_NAMESPACE", "")
	cfg.ServiceBusKeyName = source.GetWithDefault("SERVICE_BUS_KEY_NAME", "")
	cfg.ServiceBusKeyValue = source.GetWithDefault("SERVICE_BUS_KEY_VALUE", "")
	cfg.ServiceBusQueue = source.GetWithDefault("SERVICE_BUS_QUEUE", "default-queue")
	cfg.ServiceBusTopic = source.GetWithDefault("SERVICE_BUS_TOPIC", "")
	cfg.ServiceBusConcurrency = getInt("SERVICE_BUS_CONCURRENCY", 1)
	
	cfg.HTTPPort = getInt("HTTP_PORT", 8080)
	cfg.HTTPReadTimeout = getInt("HTTP_READ_TIMEOUT", 30)
	cfg.HTTPWriteTimeout = getInt("HTTP_WRITE_TIMEOUT", 30)
	cfg.HTTPIdleTimeout = getInt("HTTP_IDLE_TIMEOUT", 120)
	
	cfg.LogLevel = source.GetWithDefault("LOG_LEVEL", "info")
	cfg.LogFormat = source.GetWithDefault("LOG_FORMAT", "json")
	
	cfg.AppName = source.GetWithDefault("APP_NAME", "go-service-kit")
	cfg.AppVersion = source.GetWithDefault("APP_VERSION", "1.0.0")
	cfg.Environment = source.GetWithDefault("ENVIRONMENT", "dev")
	
	cfg.RetryMaxAttempts = getInt("RETRY_MAX_ATTEMPTS", 3)
	cfg.RetryInitialDelay = getInt("RETRY_INITIAL_DELAY", 100)
	cfg.RetryMaxDelay = getInt("RETRY_MAX_DELAY", 5000)
	
	return cfg, nil
}

// LoadConfigFromEnv loads configuration from environment variables.
func LoadConfigFromEnv() (*Config, error) {
	return LoadConfig(&EnvConfigSource{})
}

// LoadConfigFromFile loads configuration from a JSON or YAML file.
// Environment variables will override file values if both are set.
func LoadConfigFromFile(filePath string) (*Config, error) {
	fileSource, err := NewFileConfigSource(filePath)
	if err != nil {
		return nil, err
	}
	
	// Create a composite source that checks env first, then file
	composite := &CompositeConfigSource{
		sources: []ConfigSource{&EnvConfigSource{}, fileSource},
	}
	
	return LoadConfig(composite)
}

// CompositeConfigSource checks multiple config sources in order.
type CompositeConfigSource struct {
	sources []ConfigSource
}

// Get retrieves a value from the first source that has it.
func (c *CompositeConfigSource) Get(key string) (string, bool) {
	for _, source := range c.sources {
		if val, ok := source.Get(key); ok {
			return val, true
		}
	}
	return "", false
}

// GetWithDefault retrieves a value from sources or returns default.
func (c *CompositeConfigSource) GetWithDefault(key, defaultValue string) string {
	for _, source := range c.sources {
		if val, ok := source.Get(key); ok {
			return val
		}
	}
	return defaultValue
}

