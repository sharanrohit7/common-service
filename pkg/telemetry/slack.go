package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/yourorg/go-service-kit/pkg/logging"
	"github.com/yourorg/go-service-kit/pkg/utils"
)

// SlackClient handles Slack webhook notifications with rate limiting.
type SlackClient struct {
	webhookURL  string
	serviceName string
	logger      logging.Logger
	enabled     bool
	client      *http.Client
	mu          sync.Mutex
	lastSent    time.Time
	minInterval time.Duration
}

// SlackConfig holds Slack configuration.
type SlackConfig struct {
	WebhookURL  string
	ServiceName string // Name of the service (e.g., "api_gateway", "hrms_core")
	Channel     string
	Enabled     bool
}

// SlackMessage represents a Slack webhook message.
type SlackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment.
type SlackAttachment struct {
	Color     string       `json:"color,omitempty"`
	Title     string       `json:"title,omitempty"`
	Text      string       `json:"text,omitempty"`
	Fields    []SlackField `json:"fields,omitempty"`
	Timestamp int64        `json:"ts,omitempty"`
}

// SlackField represents a field in a Slack attachment.
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// NewSlackClient creates a new Slack client.
func NewSlackClient(cfg SlackConfig, logger logging.Logger) *SlackClient {
	if !cfg.Enabled || cfg.WebhookURL == "" {
		logger.Info("Slack notifications disabled or webhook URL not provided")
		return &SlackClient{
			enabled: false,
			logger:  logger,
		}
	}

	return &SlackClient{
		webhookURL:  cfg.WebhookURL,
		serviceName: cfg.ServiceName,
		logger:      logger,
		enabled:     true,
		client:      &http.Client{Timeout: 10 * time.Second},
		minInterval: 1 * time.Second, // Minimum interval between messages
	}
}

// SendMessage sends a message to Slack with rate limiting.
func (s *SlackClient) SendMessage(ctx context.Context, msg SlackMessage) error {
	if !s.enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Rate limiting: ensure minimum interval between messages
	now := time.Now()
	if !s.lastSent.IsZero() && now.Sub(s.lastSent) < s.minInterval {
		time.Sleep(s.minInterval - now.Sub(s.lastSent))
	}

	// Set channel if not provided
	if msg.Channel == "" {
		msg.Channel = "#alerts"
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API returned status %d", resp.StatusCode)
	}

	s.lastSent = time.Now()
	return nil
}

// SendSlowRequestAlert sends a slow request alert to Slack.
// Implements middleware.SlackClient interface.
func (s *SlackClient) SendSlowRequestAlert(ctx interface{}, path string, durationMs int64, traceID, requestID string) error {
	if !s.enabled {
		return nil
	}

	color := "warning"
	title := fmt.Sprintf("⚠️ Slow Request Detected - %s", s.serviceName)
	text := fmt.Sprintf("A slow request was detected in %s", s.serviceName)

	fields := []SlackField{
		{Title: "Path", Value: path, Short: true},
		{Title: "Service", Value: s.serviceName, Short: true},
		{Title: "Duration", Value: fmt.Sprintf("%d ms", durationMs), Short: true},
		{Title: "Trace ID", Value: traceID, Short: true},
		{Title: "Request ID", Value: requestID, Short: true},
	}

	attachment := SlackAttachment{
		Color:     color,
		Title:     title,
		Text:      text,
		Fields:    fields,
		Timestamp: time.Now().Unix(),
	}

	msg := SlackMessage{
		Text:        title,
		Attachments: []SlackAttachment{attachment},
	}

	// Convert ctx to context.Context if possible
	var ctxContext context.Context
	if c, ok := ctx.(context.Context); ok {
		ctxContext = c
	} else {
		ctxContext = context.Background()
	}

	return s.SendMessage(ctxContext, msg)
}

// SendErrorAlert sends an error alert to Slack.
// Implements middleware.SlackClient interface.
func (s *SlackClient) SendErrorAlert(ctx interface{}, path, errorMsg string, statusCode int, traceID, requestID string) error {
	if !s.enabled {
		return nil
	}

	color := "danger"
	title := fmt.Sprintf("🚨 Error - %s", s.serviceName)
	text := fmt.Sprintf("An error occurred in %s", s.serviceName)

	fields := []SlackField{
		{Title: "Path", Value: path, Short: true},
		{Title: "Service", Value: s.serviceName, Short: true},
		{Title: "Error", Value: errorMsg, Short: false},
		{Title: "Status Code", Value: fmt.Sprintf("%d", statusCode), Short: true},
		{Title: "Trace ID", Value: traceID, Short: true},
		{Title: "Request ID", Value: requestID, Short: true},
	}

	attachment := SlackAttachment{
		Color:     color,
		Title:     title,
		Text:      text,
		Fields:    fields,
		Timestamp: time.Now().Unix(),
	}

	msg := SlackMessage{
		Text:        title,
		Attachments: []SlackAttachment{attachment},
	}

	// Convert ctx to context.Context if possible
	var ctxContext context.Context
	if c, ok := ctx.(context.Context); ok {
		ctxContext = c
	} else {
		ctxContext = context.Background()
	}

	return s.SendMessage(ctxContext, msg)
}

// RetrySendMessage sends a message with retry logic.
func (s *SlackClient) RetrySendMessage(ctx context.Context, msg SlackMessage, maxAttempts int) error {
	if !s.enabled {
		return nil
	}

	config := utils.RetryConfig{
		MaxAttempts:  maxAttempts,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2.0,
	}

	return utils.Retry(ctx, config, func() error {
		return s.SendMessage(ctx, msg)
	})
}
