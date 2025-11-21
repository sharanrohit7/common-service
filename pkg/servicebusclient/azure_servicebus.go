package servicebusclient

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// AzureServiceBusClient implements ServiceBusClient using Azure Service Bus.
type AzureServiceBusClient struct {
	client *azservicebus.Client
	logger logging.Logger
}

// NewAzureServiceBusClient creates a new Azure Service Bus client.
// namespace: Azure Service Bus namespace (e.g., "mynamespace.servicebus.windows.net")
// keyName: Shared access key name (optional if using managed identity)
// keyValue: Shared access key value (optional if using managed identity)
// useManagedIdentity: if true, uses managed identity instead of shared access key
func NewAzureServiceBusClient(namespace, keyName, keyValue string, useManagedIdentity bool, logger logging.Logger) (*AzureServiceBusClient, error) {
	var client *azservicebus.Client
	var err error
	
	namespaceURL := fmt.Sprintf("https://%s.servicebus.windows.net/", namespace)
	
	if useManagedIdentity || keyName == "" || keyValue == "" {
		// Use managed identity
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure credential: %w", err)
		}
		client, err = azservicebus.NewClient(namespaceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Service Bus client: %w", err)
		}
	} else {
		// Use connection string (constructed from key)
		// Note: In production, you might want to use a full connection string
		// For now, we'll use shared key credential
		connStr := fmt.Sprintf("Endpoint=sb://%s.servicebus.windows.net/;SharedAccessKeyName=%s;SharedAccessKey=%s",
			namespace, keyName, keyValue)
		client, err = azservicebus.NewClientFromConnectionString(connStr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Service Bus client: %w", err)
		}
	}
	
	return &AzureServiceBusClient{
		client: client,
		logger: logger,
	}, nil
}

// Send sends a message to a queue or topic.
func (a *AzureServiceBusClient) Send(ctx context.Context, queueOrTopicName string, body []byte, opts ...SendOption) (string, error) {
	logger := a.logger.With(
		logging.NewField("operation", "servicebus.send"),
		logging.NewField("queue", queueOrTopicName),
	)
	
	logger.Info("Sending message to Service Bus")
	
	sendOptions := &SendOptions{}
	for _, opt := range opts {
		opt(sendOptions)
	}
	
	sender, err := a.client.NewSender(queueOrTopicName, nil)
	if err != nil {
		logger.Error("Failed to create sender", logging.NewField("error", err))
		return "", fmt.Errorf("failed to create sender: %w", err)
	}
	defer sender.Close(ctx)
	
	sbMessage := &azservicebus.Message{
		Body: body,
	}
	
	if sendOptions.ContentType != "" {
		sbMessage.ContentType = &sendOptions.ContentType
	}
	
	if sendOptions.MessageID != "" {
		sbMessage.MessageID = &sendOptions.MessageID
	}
	
	if sendOptions.Properties != nil {
		sbMessage.ApplicationProperties = make(map[string]interface{})
		for k, v := range sendOptions.Properties {
			sbMessage.ApplicationProperties[k] = v
		}
	}
	
	err = sender.SendMessage(ctx, sbMessage, nil)
	if err != nil {
		logger.Error("Failed to send message", logging.NewField("error", err))
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	
	messageID := ""
	if sbMessage.MessageID != nil {
		messageID = *sbMessage.MessageID
	} else {
		messageID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}
	
	logger.Info("Message sent successfully", logging.NewField("messageID", messageID))
	return messageID, nil
}

// SendBatch sends multiple messages in a batch.
func (a *AzureServiceBusClient) SendBatch(ctx context.Context, queueOrTopicName string, messages [][]byte, opts ...SendOption) error {
	logger := a.logger.With(
		logging.NewField("operation", "servicebus.sendbatch"),
		logging.NewField("queue", queueOrTopicName),
		logging.NewField("count", len(messages)),
	)
	
	logger.Info("Sending batch of messages")
	
	sender, err := a.client.NewSender(queueOrTopicName, nil)
	if err != nil {
		logger.Error("Failed to create sender", logging.NewField("error", err))
		return fmt.Errorf("failed to create sender: %w", err)
	}
	defer sender.Close(ctx)
	
	sendOptions := &SendOptions{}
	for _, opt := range opts {
		opt(sendOptions)
	}
	
	sbMessages := make([]*azservicebus.Message, 0, len(messages))
	for _, body := range messages {
		sbMessage := &azservicebus.Message{
			Body: body,
		}
		
		if sendOptions.ContentType != "" {
			sbMessage.ContentType = &sendOptions.ContentType
		}
		
		if sendOptions.Properties != nil {
			sbMessage.ApplicationProperties = make(map[string]interface{})
			for k, v := range sendOptions.Properties {
				sbMessage.ApplicationProperties[k] = v
			}
		}
		
		sbMessages = append(sbMessages, sbMessage)
	}
	
	// Send messages one by one (batch send not available in this SDK version)
	for _, msg := range sbMessages {
		if sendErr := sender.SendMessage(ctx, msg, nil); sendErr != nil {
			logger.Error("Failed to send message in batch", logging.NewField("error", sendErr))
			return fmt.Errorf("failed to send message in batch: %w", sendErr)
		}
	}
	
	logger.Info("Batch sent successfully")
	return nil
}

// Receive receives messages from a queue or subscription.
func (a *AzureServiceBusClient) Receive(ctx context.Context, queueOrSubscription string, maxMessages int) ([]Message, error) {
	logger := a.logger.With(
		logging.NewField("operation", "servicebus.receive"),
		logging.NewField("queue", queueOrSubscription),
	)
	
	logger.Info("Receiving messages")
	
	receiver, err := a.client.NewReceiverForQueue(queueOrSubscription, nil)
	if err != nil {
		logger.Error("Failed to create receiver", logging.NewField("error", err))
		return nil, fmt.Errorf("failed to create receiver: %w", err)
	}
	defer receiver.Close(ctx)
	
	var messages []Message
	
	// Receive messages with timeout
	receiveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	receivedMessages, err := receiver.ReceiveMessages(receiveCtx, maxMessages, nil)
	if err != nil {
		if err == context.DeadlineExceeded {
			logger.Debug("No messages received (timeout)")
			return messages, nil
		}
		logger.Error("Failed to receive messages", logging.NewField("error", err))
		return nil, fmt.Errorf("failed to receive messages: %w", err)
	}
	
	for _, sbMsg := range receivedMessages {
		msg := Message{
			Body:        sbMsg.Body,
			LockToken:   string(sbMsg.LockToken[:]),
			ContentType: "",
			Properties:  make(map[string]interface{}),
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
		
		messages = append(messages, msg)
	}
	
	logger.Info("Messages received", logging.NewField("count", len(messages)))
	return messages, nil
}

// Complete marks a message as completed.
func (a *AzureServiceBusClient) Complete(ctx context.Context, lockToken string) error {
	// Note: In Azure Service Bus SDK, completion is done through the receiver
	// This is a simplified interface - in practice, you'd need to store the receiver
	// or pass the received message object
	logger := a.logger.With(
		logging.NewField("operation", "servicebus.complete"),
		logging.NewField("lockToken", lockToken),
	)
	
	logger.Info("Completing message")
	// TODO: Implement completion through stored receiver context
	// This requires refactoring to maintain receiver references
	return fmt.Errorf("Complete must be called through receiver context - see consumer.go for proper usage")
}

// Abandon releases the lock on a message.
func (a *AzureServiceBusClient) Abandon(ctx context.Context, lockToken string) error {
	logger := a.logger.With(
		logging.NewField("operation", "servicebus.abandon"),
		logging.NewField("lockToken", lockToken),
	)
	
	logger.Info("Abandoning message")
	// TODO: Implement abandon through stored receiver context
	return fmt.Errorf("Abandon must be called through receiver context - see consumer.go for proper usage")
}

// Note: For local development, you can use Azure Service Bus Emulator or
// Azure Storage Queue as a simpler alternative:
// 1. Use Azure Storage Queue SDK for simpler queue operations
// 2. Or use a local message broker like RabbitMQ with a compatible interface
// 3. Azure Service Bus Emulator is not officially available, but you can use
//    Azure Storage Queue or a test double for local development

