package event

import (
	"context"
	"fmt"
	"log/slog"

	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
)

// Kafka topic constants for notification domain events.
const (
	TopicNotificationSent   = "ecommerce.notification.sent"
	TopicNotificationFailed = "ecommerce.notification.failed"
)

// Aggregate type constant.
const AggregateTypeNotification = "notification"

// Source identifier for events originating from the notification service.
const SourceNotificationService = "notification-service"

// NotificationSentData is the payload for a notification.sent event.
type NotificationSentData struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Subject string `json:"subject"`
}

// NotificationFailedData is the payload for a notification.failed event.
type NotificationFailedData struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Type       string `json:"type"`
	Channel    string `json:"channel"`
	RetryCount int    `json:"retry_count"`
}

// Producer publishes notification domain events to Kafka.
type Producer struct {
	kafka  *pkgkafka.Producer
	logger *slog.Logger
}

// NewProducer creates a new event producer for the notification service.
func NewProducer(kafka *pkgkafka.Producer, logger *slog.Logger) *Producer {
	return &Producer{
		kafka:  kafka,
		logger: logger,
	}
}

// PublishNotificationSent publishes a notification.sent event.
func (p *Producer) PublishNotificationSent(ctx context.Context, notification *domain.Notification) error {
	if p.kafka == nil {
		return nil
	}

	data := NotificationSentData{
		ID:      notification.ID,
		UserID:  notification.UserID,
		Type:    notification.Type,
		Channel: notification.Channel,
		Subject: notification.Subject,
	}

	event, err := pkgkafka.NewEvent(TopicNotificationSent, notification.ID, AggregateTypeNotification, SourceNotificationService, data)
	if err != nil {
		return fmt.Errorf("create notification.sent event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicNotificationSent, event); err != nil {
		return fmt.Errorf("publish notification.sent event: %w", err)
	}

	p.logger.DebugContext(ctx, "published notification.sent event",
		slog.String("notification_id", notification.ID),
	)

	return nil
}

// PublishNotificationFailed publishes a notification.failed event.
func (p *Producer) PublishNotificationFailed(ctx context.Context, notification *domain.Notification) error {
	if p.kafka == nil {
		return nil
	}

	data := NotificationFailedData{
		ID:         notification.ID,
		UserID:     notification.UserID,
		Type:       notification.Type,
		Channel:    notification.Channel,
		RetryCount: notification.RetryCount,
	}

	event, err := pkgkafka.NewEvent(TopicNotificationFailed, notification.ID, AggregateTypeNotification, SourceNotificationService, data)
	if err != nil {
		return fmt.Errorf("create notification.failed event: %w", err)
	}

	if err := p.kafka.Publish(ctx, TopicNotificationFailed, event); err != nil {
		return fmt.Errorf("publish notification.failed event: %w", err)
	}

	p.logger.DebugContext(ctx, "published notification.failed event",
		slog.String("notification_id", notification.ID),
	)

	return nil
}
