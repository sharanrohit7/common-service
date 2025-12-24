package jwt

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

const (
	// Context keys for storing JWT claims
	ContextKeyUserID    = "user_id"
	ContextKeyRoleID    = "role_id"
	ContextKeyCompanyID = "company_id"
	ContextKeyClaims    = "jwt_claims"
)

// JWTMiddleware creates a middleware that validates JWT tokens
func JWTMiddleware(jwtService *JWTService, logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Security: Validate header format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn("Invalid authorization header format",
				logging.NewField("header_length", len(authHeader)),
				logging.NewField("ip", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format. Expected: Bearer <token>"})
			c.Abort()
			return
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		// Security: Validate token is not empty
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is required"})
			c.Abort()
			return
		}

		// Security: Check for potential token manipulation
		if containsMultipleTokens(tokenString) {
			logger.Warn("Multiple tokens detected in request",
				logging.NewField("ip", c.ClientIP()),
				logging.NewField("user_agent", c.GetHeader("User-Agent")),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			// Log security events
			logger.Warn("Token validation failed",
				logging.NewField("error", err),
				logging.NewField("ip", c.ClientIP()),
				logging.NewField("path", c.Request.URL.Path),
				logging.NewField("method", c.Request.Method),
			)

			// Return appropriate error based on validation failure
			statusCode := http.StatusUnauthorized
			errorMsg := "Token validation failed"

			switch err {
			case ErrExpiredToken:
				errorMsg = "Token has expired"
			case ErrInvalidToken, ErrInvalidSignature:
				errorMsg = "Invalid token"
			case ErrTokenPoisoned:
				errorMsg = "Token validation failed"
				statusCode = http.StatusForbidden // More severe error
			case ErrTokenTooLarge:
				errorMsg = "Token size exceeds maximum allowed"
				statusCode = http.StatusRequestEntityTooLarge
			}

			c.JSON(statusCode, gin.H{"error": errorMsg})
			c.Abort()
			return
		}

		// Security: Additional validation - check if user_id is present
		if claims.UserID == "" {
			logger.Error("Token missing user_id claim",
				logging.NewField("ip", c.ClientIP()),
				logging.NewField("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Store claims in context for use in handlers
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRoleID, claims.RoleID)
		c.Set(ContextKeyCompanyID, claims.CompanyID)
		c.Set(ContextKeyClaims, claims)

		// Continue to next handler
		c.Next()
	}
}

// OptionalJWTMiddleware creates a middleware that validates JWT tokens if present,
// but doesn't require them. Useful for endpoints that work with or without authentication.
func OptionalJWTMiddleware(jwtService *JWTService, logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		// Token provided, validate it
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			// Log but don't fail - optional middleware
			logger.Debug("Optional token validation failed",
				logging.NewField("error", err),
				logging.NewField("ip", c.ClientIP()),
			)
			c.Next()
			return
		}

		// Store claims if validation succeeded
		if claims.UserID != "" {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyRoleID, claims.RoleID)
			c.Set(ContextKeyCompanyID, claims.CompanyID)
			c.Set(ContextKeyClaims, claims)
		}

		c.Next()
	}
}

// RoleBasedMiddleware creates a middleware that checks if the user has one of the required roles
func RoleBasedMiddleware(requiredRoles []string, logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context (set by JWTMiddleware)
		roleID, exists := c.Get(ContextKeyRoleID)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Role information not found in token"})
			c.Abort()
			return
		}

		roleIDStr, ok := roleID.(string)
		if !ok || roleIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role in token"})
			c.Abort()
			return
		}

		// Check if user's role is in the required roles list
		hasRequiredRole := false
		for _, requiredRole := range requiredRoles {
			if roleIDStr == requiredRole {
				hasRequiredRole = true
				break
			}
		}

		if !hasRequiredRole {
			logger.Warn("Access denied: insufficient role",
				logging.NewField("user_role", roleIDStr),
				logging.NewField("required_roles", requiredRoles),
				logging.NewField("ip", c.ClientIP()),
				logging.NewField("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CompanyBasedMiddleware creates a middleware that checks if the user belongs to a specific company
func CompanyBasedMiddleware(requiredCompanyID string, logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID, exists := c.Get(ContextKeyCompanyID)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "Company information not found in token"})
			c.Abort()
			return
		}

		companyIDStr, ok := companyID.(string)
		if !ok || companyIDStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid company in token"})
			c.Abort()
			return
		}

		if companyIDStr != requiredCompanyID {
			logger.Warn("Access denied: company mismatch",
				logging.NewField("user_company", companyIDStr),
				logging.NewField("required_company", requiredCompanyID),
				logging.NewField("ip", c.ClientIP()),
				logging.NewField("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: company mismatch"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// GetRoleID extracts role ID from context
func GetRoleID(c *gin.Context) (string, bool) {
	roleID, exists := c.Get(ContextKeyRoleID)
	if !exists {
		return "", false
	}
	roleIDStr, ok := roleID.(string)
	return roleIDStr, ok
}

// GetCompanyID extracts company ID from context
func GetCompanyID(c *gin.Context) (string, bool) {
	companyID, exists := c.Get(ContextKeyCompanyID)
	if !exists {
		return "", false
	}
	companyIDStr, ok := companyID.(string)
	return companyIDStr, ok
}

// GetClaims extracts full JWT claims from context
func GetClaims(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*Claims)
	return jwtClaims, ok
}

// containsMultipleTokens checks if the token string contains multiple tokens
// This is a security check to prevent token manipulation attacks
func containsMultipleTokens(tokenString string) bool {
	// Count occurrences of "Bearer" (should be 0 after we strip it)
	// Count dots (JWT tokens have exactly 2 dots)
	dotCount := strings.Count(tokenString, ".")
	if dotCount > 2 {
		return true
	}

	// Check for suspicious patterns like multiple base64-like segments
	parts := strings.Split(tokenString, ".")
	if len(parts) > 3 {
		return true
	}

	return false
}
