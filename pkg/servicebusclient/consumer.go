package servicebusclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// MessageHandler processes a single message.
type MessageHandler func(ctx context.Context, msg Message) error

// ConsumerConfig configures a Service Bus consumer.
type ConsumerConfig struct {
	QueueOrSubscription string
	MaxConcurrent       int
	MaxMessages         int
	ReceiveTimeout      time.Duration
	Logger              logging.Logger
}

// Consumer handles receiving and processing messages from Service Bus.
type Consumer struct {
	client      *azservicebus.Client
	config      ConsumerConfig
	receiver    *azservicebus.Receiver
	handler     MessageHandler
	wg          sync.WaitGroup
	stopChan    chan struct{}
	logger      logging.Logger
}

// NewConsumer creates a new Service Bus consumer.
func NewConsumer(client *azservicebus.Client, config ConsumerConfig, handler MessageHandler) (*Consumer, error) {
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 1
	}
	if config.MaxMessages <= 0 {
		config.MaxMessages = 10
	}
	if config.ReceiveTimeout == 0 {
		config.ReceiveTimeout = 5 * time.Second
	}
	
	receiver, err := client.NewReceiverForQueue(config.QueueOrSubscription, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create receiver: %w", err)
	}
	
	return &Consumer{
		client:   client,
		config:   config,
		receiver: receiver,
		handler:  handler,
		stopChan: make(chan struct{}),
		logger:   config.Logger,
	}, nil
}

// Start starts the consumer with configurable concurrency.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Service Bus consumer",
		logging.NewField("queue", c.config.QueueOrSubscription),
		logging.NewField("concurrency", c.config.MaxConcurrent),
	)
	
	// Start worker goroutines
	for i := 0; i < c.config.MaxConcurrent; i++ {
		c.wg.Add(1)
		go c.worker(ctx, i)
	}
	
	return nil
}

// worker is a single worker goroutine that receives and processes messages.
func (c *Consumer) worker(ctx context.Context, workerID int) {
	defer c.wg.Done()
	
	logger := c.logger.With(logging.NewField("worker", workerID))
	logger.Info("Worker started")
	
	for {
		select {
		case <-c.stopChan:
			logger.Info("Worker stopping")
			return
		case <-ctx.Done():
			logger.Info("Worker stopping (context cancelled)")
			return
		default:
			// Receive messages
			receiveCtx, cancel := context.WithTimeout(ctx, c.config.ReceiveTimeout)
			messages, err := c.receiver.ReceiveMessages(receiveCtx, c.config.MaxMessages, nil)
			cancel()
			
			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					// Timeout is expected, continue
					continue
				}
				logger.Error("Failed to receive messages", logging.NewField("error", err))
				time.Sleep(1 * time.Second) // Back off on error
				continue
			}
			
			// Process each message
			for _, sbMsg := range messages {
				msg := convertAzureMessage(sbMsg)
				
				handlerCtx := context.WithValue(ctx, "message", msg)
				err := c.handler(handlerCtx, msg)
				
				if err != nil {
					logger.Error("Message handler failed",
						logging.NewField("messageID", msg.ID),
						logging.NewField("error", err),
					)
					// Abandon the message so it can be retried
					if abandonErr := c.receiver.AbandonMessage(handlerCtx, sbMsg, nil); abandonErr != nil {
						logger.Error("Failed to abandon message", logging.NewField("error", abandonErr))
					}
				} else {
					// Complete the message
					if completeErr := c.receiver.CompleteMessage(handlerCtx, sbMsg, nil); completeErr != nil {
						logger.Error("Failed to complete message", logging.NewField("error", completeErr))
					} else {
						logger.Debug("Message processed successfully", logging.NewField("messageID", msg.ID))
					}
				}
			}
		}
	}
}

// Stop gracefully stops the consumer.
func (c *Consumer) Stop(ctx context.Context) error {
	c.logger.Info("Stopping Service Bus consumer")
	
	close(c.stopChan)
	
	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		c.logger.Info("All workers stopped")
	case <-ctx.Done():
		c.logger.Warn("Timeout waiting for workers to stop")
	case <-time.After(30 * time.Second):
		c.logger.Warn("Timeout waiting for workers to stop")
	}
	
	if c.receiver != nil {
		return c.receiver.Close(ctx)
	}
	
	return nil
}

// convertAzureMessage converts an Azure Service Bus message to our Message type.
func convertAzureMessage(sbMsg *azservicebus.ReceivedMessage) Message {
	msg := Message{
		Body:       sbMsg.Body,
		LockToken:  string(sbMsg.LockToken[:]),
		Properties: make(map[string]interface{}),
	}
	
	// MessageID is a string in the SDK
	msg.ID = sbMsg.MessageID
	
	if sbMsg.ContentType != nil {
		msg.ContentType = *sbMsg.ContentType
	}
	
	if sbMsg.ApplicationProperties != nil {
		for k, v := range sbMsg.ApplicationProperties {
			msg.Properties[k] = v
		}
	}
	
	if sbMsg.EnqueuedTime != nil {
		msg.EnqueuedAt = *sbMsg.EnqueuedTime
	}
	
	return msg
}

