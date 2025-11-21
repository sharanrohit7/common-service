package utils

import (
	"github.com/google/uuid"
)

// GenerateUUID generates a new UUID v4.
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateRequestID generates a request ID (UUID v4).
func GenerateRequestID() string {
	return GenerateUUID()
}

// IsValidUUID checks if a string is a valid UUID.
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

