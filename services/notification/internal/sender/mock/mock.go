package mock

import (
	"context"
	"log/slog"
	"time"

	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
)

// MockSender is a sender implementation that logs notifications and always succeeds.
// It simulates a 10ms delay to mimic real sending latency.
type MockSender struct {
	channel string
	logger  *slog.Logger
}

// NewMockSender creates a new mock sender for the given channel.
func NewMockSender(channel string, logger *slog.Logger) *MockSender {
	return &MockSender{
		channel: channel,
		logger:  logger,
	}
}

// Name returns the name of this sender.
func (s *MockSender) Name() string {
	return "mock-" + s.channel
}

// Send logs the notification details and simulates a 10ms sending delay.
func (s *MockSender) Send(ctx context.Context, notification *domain.Notification) error {
	// Simulate sending delay.
	time.Sleep(10 * time.Millisecond)

	s.logger.InfoContext(ctx, "mock sender: notification sent",
		slog.String("notification_id", notification.ID),
		slog.String("user_id", notification.UserID),
		slog.String("channel", notification.Channel),
		slog.String("type", notification.Type),
		slog.String("subject", notification.Subject),
		slog.String("priority", notification.Priority),
	)

	return nil
}
