package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
	"github.com/utafrali/EcommerceGo/services/notification/internal/event"
	"github.com/utafrali/EcommerceGo/services/notification/internal/repository"
	"github.com/utafrali/EcommerceGo/services/notification/internal/sender"
)

// NotificationService implements the business logic for notification operations.
type NotificationService struct {
	repo     repository.NotificationRepository
	senders  map[string]sender.Sender
	producer *event.Producer
	logger   *slog.Logger
}

// NewNotificationService creates a new notification service.
func NewNotificationService(
	repo repository.NotificationRepository,
	senders map[string]sender.Sender,
	producer *event.Producer,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		repo:     repo,
		senders:  senders,
		producer: producer,
		logger:   logger,
	}
}

// SendNotificationInput holds the parameters for sending a notification.
type SendNotificationInput struct {
	UserID   string
	Type     string
	Channel  string
	Subject  string
	Body     string
	Priority string
	Metadata map[string]any
}

// SendNotification creates and sends a notification via the appropriate sender.
func (s *NotificationService) SendNotification(ctx context.Context, input *SendNotificationInput) (*domain.Notification, error) {
	if input.UserID == "" {
		return nil, apperrors.InvalidInput("user_id is required")
	}
	if input.Type == "" {
		return nil, apperrors.InvalidInput("type is required")
	}
	if !domain.IsValidType(input.Type) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid notification type %q", input.Type))
	}
	if input.Channel == "" {
		return nil, apperrors.InvalidInput("channel is required")
	}
	if !domain.IsValidChannel(input.Channel) {
		return nil, apperrors.InvalidInput("invalid channel: must be one of email, sms, push, in_app")
	}
	if input.Body == "" {
		return nil, apperrors.InvalidInput("body is required")
	}

	priority := input.Priority
	if priority == "" {
		priority = domain.NotificationPriorityNormal
	}
	if !domain.IsValidPriority(priority) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid priority %q", priority))
	}

	now := time.Now().UTC()
	notification := &domain.Notification{
		ID:         uuid.New().String(),
		UserID:     input.UserID,
		Type:       input.Type,
		Channel:    input.Channel,
		Subject:    input.Subject,
		Body:       input.Body,
		Status:     domain.NotificationStatusPending,
		Priority:   priority,
		Metadata:   input.Metadata,
		RetryCount: 0,
		MaxRetries: domain.DefaultMaxRetries,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if notification.Metadata == nil {
		notification.Metadata = make(map[string]any)
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}

	// Attempt to send the notification.
	s.send(ctx, notification)

	return notification, nil
}

// GetNotification retrieves a notification by its ID.
func (s *NotificationService) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	return notification, nil
}

// ListNotificationsByUser returns a paginated list of notifications for a user.
func (s *NotificationService) ListNotificationsByUser(ctx context.Context, userID string, page, perPage int) ([]domain.Notification, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	notifications, total, err := s.repo.ListByUserID(ctx, userID, offset, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications by user: %w", err)
	}

	return notifications, total, nil
}

// MarkAsRead marks a notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, id string) (*domain.Notification, error) {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification for mark as read: %w", err)
	}

	now := time.Now().UTC()
	notification.Status = domain.NotificationStatusRead
	notification.ReadAt = &now

	if err := s.repo.Update(ctx, notification); err != nil {
		return nil, fmt.Errorf("update notification: %w", err)
	}

	s.logger.InfoContext(ctx, "notification marked as read",
		slog.String("notification_id", notification.ID),
	)

	return notification, nil
}

// RetryNotification retries a failed notification.
func (s *NotificationService) RetryNotification(ctx context.Context, id string) (*domain.Notification, error) {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get notification for retry: %w", err)
	}

	if notification.Status != domain.NotificationStatusFailed {
		return nil, apperrors.InvalidInput("only failed notifications can be retried")
	}

	if notification.RetryCount >= notification.MaxRetries {
		return nil, apperrors.InvalidInput("notification has exceeded maximum retry attempts")
	}

	// Reset status to pending and attempt to send again.
	notification.Status = domain.NotificationStatusPending
	notification.RetryCount++

	if err := s.repo.Update(ctx, notification); err != nil {
		return nil, fmt.Errorf("update notification for retry: %w", err)
	}

	s.send(ctx, notification)

	return notification, nil
}

// ListPendingNotifications returns pending notifications for background retry processing.
func (s *NotificationService) ListPendingNotifications(ctx context.Context, limit int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 50
	}

	notifications, err := s.repo.ListPending(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending notifications: %w", err)
	}

	return notifications, nil
}

// send attempts to send a notification via the appropriate sender and updates its status.
func (s *NotificationService) send(ctx context.Context, notification *domain.Notification) {
	snd, ok := s.senders[notification.Channel]
	if !ok {
		s.logger.ErrorContext(ctx, "no sender registered for channel",
			slog.String("channel", notification.Channel),
			slog.String("notification_id", notification.ID),
		)
		notification.Status = domain.NotificationStatusFailed
		if err := s.repo.Update(ctx, notification); err != nil {
			s.logger.ErrorContext(ctx, "failed to update notification status",
				slog.String("notification_id", notification.ID),
				slog.String("error", err.Error()),
			)
		}

		if err := s.producer.PublishNotificationFailed(ctx, notification); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish notification.failed event",
				slog.String("notification_id", notification.ID),
				slog.String("error", err.Error()),
			)
		}
		return
	}

	if err := snd.Send(ctx, notification); err != nil {
		s.logger.ErrorContext(ctx, "sender failed to send notification",
			slog.String("notification_id", notification.ID),
			slog.String("channel", notification.Channel),
			slog.String("error", err.Error()),
		)
		notification.Status = domain.NotificationStatusFailed
		notification.RetryCount++
		if updateErr := s.repo.Update(ctx, notification); updateErr != nil {
			s.logger.ErrorContext(ctx, "failed to update notification status",
				slog.String("notification_id", notification.ID),
				slog.String("error", updateErr.Error()),
			)
		}

		if pubErr := s.producer.PublishNotificationFailed(ctx, notification); pubErr != nil {
			s.logger.ErrorContext(ctx, "failed to publish notification.failed event",
				slog.String("notification_id", notification.ID),
				slog.String("error", pubErr.Error()),
			)
		}
		return
	}

	now := time.Now().UTC()
	notification.Status = domain.NotificationStatusSent
	notification.SentAt = &now

	if err := s.repo.Update(ctx, notification); err != nil {
		s.logger.ErrorContext(ctx, "failed to update notification status after send",
			slog.String("notification_id", notification.ID),
			slog.String("error", err.Error()),
		)
	}

	if err := s.producer.PublishNotificationSent(ctx, notification); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish notification.sent event",
			slog.String("notification_id", notification.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "notification sent",
		slog.String("notification_id", notification.ID),
		slog.String("channel", notification.Channel),
		slog.String("user_id", notification.UserID),
	)
}
