package utils

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

// Retry executes a function with exponential backoff retry logic.
func Retry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts-1 {
			delay := calculateDelay(config, attempt)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}
	
	return fmt.Errorf("max attempts (%d) reached, last error: %w", config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the given attempt using exponential backoff.
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt))
	
	maxDelay := float64(config.MaxDelay)
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return time.Duration(delay)
}

// RetryWithResult executes a function that returns a result with exponential backoff retry logic.
func RetryWithResult[T any](ctx context.Context, config RetryConfig, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}
		
		result, err := fn()
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		if attempt < config.MaxAttempts-1 {
			delay := calculateDelay(config, attempt)
			
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}
	
	return zero, fmt.Errorf("max attempts (%d) reached, last error: %w", config.MaxAttempts, lastErr)
}

