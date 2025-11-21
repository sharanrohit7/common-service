package servicebusclient

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockServiceBusClient is an in-memory implementation of ServiceBusClient for testing.
type MockServiceBusClient struct {
	queues map[string][]Message // queueName -> messages
	mu     sync.RWMutex
}

// NewMockServiceBusClient creates a new mock Service Bus client.
func NewMockServiceBusClient() *MockServiceBusClient {
	return &MockServiceBusClient{
		queues: make(map[string][]Message),
	}
}

// Send sends a message to the mock queue.
func (m *MockServiceBusClient) Send(ctx context.Context, queueOrTopicName string, body []byte, opts ...SendOption) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.queues[queueOrTopicName] == nil {
		m.queues[queueOrTopicName] = make([]Message, 0)
	}
	
	sendOptions := &SendOptions{}
	for _, opt := range opts {
		opt(sendOptions)
	}
	
	messageID := sendOptions.MessageID
	if messageID == "" {
		messageID = fmt.Sprintf("mock-msg-%d", time.Now().UnixNano())
	}
	
	msg := Message{
		ID:          messageID,
		Body:        body,
		LockToken:   fmt.Sprintf("lock-%s", messageID),
		ContentType: sendOptions.ContentType,
		Properties:  sendOptions.Properties,
		EnqueuedAt:  time.Now(),
	}
	
	m.queues[queueOrTopicName] = append(m.queues[queueOrTopicName], msg)
	
	return messageID, nil
}

// SendBatch sends multiple messages in a batch.
func (m *MockServiceBusClient) SendBatch(ctx context.Context, queueOrTopicName string, messages [][]byte, opts ...SendOption) error {
	for _, body := range messages {
		_, err := m.Send(ctx, queueOrTopicName, body, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

// Receive receives messages from the mock queue.
func (m *MockServiceBusClient) Receive(ctx context.Context, queueOrSubscription string, maxMessages int) ([]Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	queue, exists := m.queues[queueOrSubscription]
	if !exists || len(queue) == 0 {
		return []Message{}, nil
	}
	
	count := maxMessages
	if count > len(queue) {
		count = len(queue)
	}
	
	messages := queue[:count]
	m.queues[queueOrSubscription] = queue[count:]
	
	return messages, nil
}

// Complete marks a message as completed (no-op in mock).
func (m *MockServiceBusClient) Complete(ctx context.Context, lockToken string) error {
	// In mock, messages are removed on receive, so this is a no-op
	return nil
}

// Abandon releases the lock on a message (adds it back to queue in mock).
func (m *MockServiceBusClient) Abandon(ctx context.Context, lockToken string) error {
	// In a real implementation, you'd need to track which queue the message came from
	// For simplicity, this is a no-op in the mock
	return nil
}

