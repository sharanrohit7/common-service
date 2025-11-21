package servicebusclient

import (
	"context"
	"time"
)

// ServiceBusClient defines the interface for Service Bus operations.
type ServiceBusClient interface {
	// Send sends a message to a queue or topic.
	Send(ctx context.Context, queueOrTopicName string, body []byte, opts ...SendOption) (messageID string, err error)
	
	// SendBatch sends multiple messages in a batch.
	SendBatch(ctx context.Context, queueOrTopicName string, messages [][]byte, opts ...SendOption) error
	
	// Receive receives messages from a queue or subscription.
	Receive(ctx context.Context, queueOrSubscription string, maxMessages int) ([]Message, error)
	
	// Complete marks a message as completed.
	Complete(ctx context.Context, lockToken string) error
	
	// Abandon releases the lock on a message so it can be received again.
	Abandon(ctx context.Context, lockToken string) error
}

// Message represents a Service Bus message.
type Message struct {
	ID          string
	Body        []byte
	LockToken   string
	ContentType string
	Properties  map[string]interface{}
	EnqueuedAt  time.Time
}

// SendOption represents optional parameters for send operations.
type SendOption func(*SendOptions)

// SendOptions contains options for send operations.
type SendOptions struct {
	ContentType string
	Properties  map[string]interface{}
	MessageID   string
}

// WithContentType sets the content type for a message.
func WithContentType(contentType string) SendOption {
	return func(opts *SendOptions) {
		opts.ContentType = contentType
	}
}

// WithProperties sets custom properties for a message.
func WithProperties(properties map[string]interface{}) SendOption {
	return func(opts *SendOptions) {
		opts.Properties = properties
	}
}

// WithMessageID sets a custom message ID.
func WithMessageID(messageID string) SendOption {
	return func(opts *SendOptions) {
		opts.MessageID = messageID
	}
}

