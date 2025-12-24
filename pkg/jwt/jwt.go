package jwt

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidClaims    = errors.New("invalid token claims")
	ErrTokenPoisoned    = errors.New("token appears to be poisoned or tampered")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrMissingClaims    = errors.New("required claims are missing")
	ErrTokenTooLarge    = errors.New("token size exceeds maximum allowed")
)

const (
	// MaxTokenSize to prevent DoS attacks (16KB)
	MaxTokenSize = 16 * 1024
	// MinSecretKeyLength for security
	MinSecretKeyLength = 32
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID    string `json:"user_id"`
	RoleID    string `json:"role_id"`
	CompanyID string `json:"company_id"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	logger        logging.Logger
}

// NewJWTService creates a new JWT service instance
func NewJWTService(secretKey string, accessExpiry, refreshExpiry time.Duration, logger logging.Logger) (*JWTService, error) {
	// Validate secret key length
	if len(secretKey) < MinSecretKeyLength {
		return nil, fmt.Errorf("secret key must be at least %d characters long", MinSecretKeyLength)
	}

	return &JWTService{
		secretKey:     []byte(secretKey),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
		logger:        logger,
	}, nil
}

// GenerateAccessToken generates a new access token with user_id, role_id, and company_id
func (j *JWTService) GenerateAccessToken(userID, roleID, companyID string) (string, error) {
	// Validate inputs to prevent injection attacks
	if err := validateInputs(userID, roleID, companyID); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// Create claims
	claims := &Claims{
		UserID:    userID,
		RoleID:    roleID,
		CompanyID: companyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
			Subject:   userID,
			ID:        generateTokenID(), // JTI claim for token revocation tracking
		},
	}

	// Create token with HS256 algorithm (HMAC-SHA256)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		j.logger.Error("Failed to sign token", logging.NewField("error", err))
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	// Validate token size to prevent DoS
	if len(tokenString) > MaxTokenSize {
		return "", ErrTokenTooLarge
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token
func (j *JWTService) GenerateRefreshToken(userID string) (string, error) {
	if err := validateInputs(userID, "", ""); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
			Subject:   userID,
			ID:        generateTokenID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		j.logger.Error("Failed to sign refresh token", logging.NewField("error", err))
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	if len(tokenString) > MaxTokenSize {
		return "", ErrTokenTooLarge
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	// Security check: Validate token size before parsing
	if len(tokenString) > MaxTokenSize {
		return nil, ErrTokenTooLarge
	}

	// Security check: Basic format validation
	if strings.TrimSpace(tokenString) == "" {
		return nil, ErrInvalidToken
	}

	// Security check: Prevent potential injection by checking for suspicious patterns
	if containsSuspiciousPatterns(tokenString) {
		j.logger.Warn("Suspicious token pattern detected", logging.NewField("token_length", len(tokenString)))
		return nil, ErrTokenPoisoned
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		// Check for specific JWT errors
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrInvalidToken
		}
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		j.logger.Warn("Token validation failed", logging.NewField("error", err))
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	// Additional security validations
	if err := j.validateClaims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// validateClaims performs additional security checks on claims
func (j *JWTService) validateClaims(claims *Claims) error {
	// Check required claims
	if claims.UserID == "" {
		return fmt.Errorf("%w: user_id is required", ErrMissingClaims)
	}

	// Validate claim values to prevent injection
	if err := validateInputs(claims.UserID, claims.RoleID, claims.CompanyID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidClaims, err)
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return ErrExpiredToken
	}

	// Check not before time
	if claims.NotBefore != nil && claims.NotBefore.Time.After(time.Now()) {
		return ErrInvalidToken
	}

	// Validate issuer
	if claims.Issuer != "" && claims.Issuer != "auth-service" {
		j.logger.Warn("Token from unexpected issuer", logging.NewField("issuer", claims.Issuer))
		// Don't reject, but log for monitoring
	}

	return nil
}

// ExtractClaims extracts claims from a token string without full validation
// Use this only when you need to read claims from an expired token
func (j *JWTService) ExtractClaims(tokenString string) (*Claims, error) {
	if len(tokenString) > MaxTokenSize {
		return nil, ErrTokenTooLarge
	}

	// Parse without validation (use with caution)
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshToken generates a new access token from a refresh token
func (j *JWTService) RefreshToken(refreshTokenString string) (string, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token with the same user info
	// Note: Refresh tokens typically don't contain role_id and company_id
	// You may need to fetch these from the database
	return j.GenerateAccessToken(claims.UserID, claims.RoleID, claims.CompanyID)
}

// validateInputs validates input strings to prevent injection attacks
func validateInputs(userID, roleID, companyID string) error {
	// Check for empty user_id (required)
	if strings.TrimSpace(userID) == "" {
		return errors.New("user_id cannot be empty")
	}

	// Check for suspicious patterns that might indicate injection attempts
	suspiciousPatterns := []string{
		"<script", "javascript:", "onerror=", "onload=",
		"../", "..\\", "union select", "drop table",
		"exec(", "eval(", "base64",
	}

	inputs := []string{userID, roleID, companyID}
	for _, input := range inputs {
		if input == "" {
			continue // Optional fields can be empty
		}
		lowerInput := strings.ToLower(input)
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(lowerInput, pattern) {
				return fmt.Errorf("suspicious pattern detected in input: %s", pattern)
			}
		}
		// Check for extremely long inputs (potential DoS)
		if len(input) > 255 {
			return errors.New("input value exceeds maximum length")
		}
	}

	return nil
}

// containsSuspiciousPatterns checks for patterns that might indicate a poisoned token
func containsSuspiciousPatterns(tokenString string) bool {
	// Check for nested JSON structures that might cause parsing issues
	if strings.Count(tokenString, "{") > 10 || strings.Count(tokenString, "[") > 10 {
		return true
	}

	// Check for extremely long segments
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false // Invalid format, will be caught by parser
	}

	// Check payload size (middle part)
	if len(parts[1]) > 8192 { // 8KB payload limit
		return true
	}

	return false
}

// generateTokenID generates a unique token ID (JTI claim)
func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
