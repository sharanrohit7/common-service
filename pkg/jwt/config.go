package jwt

import (
	"time"

	"github.com/yourorg/go-service-kit/pkg/logging"
)

// Config holds JWT configuration
type Config struct {
	SecretKey             string
	AccessTokenExpiryMins int
	RefreshTokenExpiryHrs int
}

// NewJWTServiceFromConfig creates a new JWT service from configuration
func NewJWTServiceFromConfig(cfg Config, logger logging.Logger) (*JWTService, error) {
	// Validate secret key is provided
	if cfg.SecretKey == "" {
		return nil, ErrInvalidToken // Reuse error for missing secret
	}

	// Set default expiry times if not configured
	accessExpiry := time.Duration(cfg.AccessTokenExpiryMins) * time.Minute
	if accessExpiry == 0 {
		accessExpiry = 15 * time.Minute // Default 15 minutes
	}

	refreshExpiry := time.Duration(cfg.RefreshTokenExpiryHrs) * time.Hour
	if refreshExpiry == 0 {
		refreshExpiry = 168 * time.Hour // Default 7 days
	}

	return NewJWTService(cfg.SecretKey, accessExpiry, refreshExpiry, logger)
}
