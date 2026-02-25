package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
	"github.com/utafrali/EcommerceGo/services/notification/internal/event"
	"github.com/utafrali/EcommerceGo/services/notification/internal/sender"
)

// --- Mock Repository ---

type mockNotificationRepository struct {
	mock.Mock
}

func (m *mockNotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *mockNotificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) Update(ctx context.Context, notification *domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *mockNotificationRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Notification, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]domain.Notification), args.Int(1), args.Error(2)
}

func (m *mockNotificationRepository) ListPending(ctx context.Context, limit int) ([]domain.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]domain.Notification), args.Error(1)
}

func (m *mockNotificationRepository) ListFailed(ctx context.Context, limit int) ([]domain.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]domain.Notification), args.Error(1)
}

// --- Mock Sender ---

type mockSender struct {
	mock.Mock
}

func (m *mockSender) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockSender) Send(ctx context.Context, notification *domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockNotificationRepository, senders map[string]sender.Sender) *NotificationService {
	logger := newTestLogger()
	// Create a nil-safe event producer for tests (events are logged but don't fail).
	producer := event.NewProducer(nil, logger)
	return NewNotificationService(repo, senders, producer, logger)
}

func newTestNotification(status string) *domain.Notification {
	now := time.Now().UTC()
	return &domain.Notification{
		ID:         "test-notification-id",
		UserID:     "test-user-id",
		Type:       domain.NotificationTypeEmail,
		Channel:    "email",
		Subject:    "Test Subject",
		Body:       "Test body content",
		Status:     status,
		Priority:   domain.NotificationPriorityNormal,
		Metadata:   map[string]any{"key": "value"},
		RetryCount: 0,
		MaxRetries: 3,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// --- Tests ---

func TestSendNotification_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	mockSnd := new(mockSender)
	senders := map[string]sender.Sender{"email": mockSnd}
	svc := newTestService(repo, senders)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	mockSnd.On("Send", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	input := &SendNotificationInput{
		UserID:   "user-123",
		Type:     domain.NotificationTypeEmail,
		Channel:  "email",
		Subject:  "Test Subject",
		Body:     "Test body",
		Priority: domain.NotificationPriorityHigh,
		Metadata: map[string]any{"order_id": "order-456"},
	}

	notification, err := svc.SendNotification(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, notification.ID)
	assert.Equal(t, "user-123", notification.UserID)
	assert.Equal(t, domain.NotificationTypeEmail, notification.Type)
	assert.Equal(t, "email", notification.Channel)
	assert.Equal(t, "Test Subject", notification.Subject)
	assert.Equal(t, "Test body", notification.Body)
	assert.Equal(t, domain.NotificationPriorityHigh, notification.Priority)
	assert.Equal(t, domain.NotificationStatusSent, notification.Status)
	assert.NotNil(t, notification.SentAt)
	assert.NotZero(t, notification.CreatedAt)
	assert.NotZero(t, notification.UpdatedAt)

	repo.AssertExpectations(t)
	mockSnd.AssertExpectations(t)
}

func TestSendNotification_SenderFailure_IncrementsRetryCount(t *testing.T) {
	repo := new(mockNotificationRepository)
	mockSnd := new(mockSender)
	senders := map[string]sender.Sender{"email": mockSnd}
	svc := newTestService(repo, senders)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	mockSnd.On("Send", ctx, mock.AnythingOfType("*domain.Notification")).Return(errors.New("smtp connection failed"))
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    domain.NotificationTypeEmail,
		Channel: "email",
		Subject: "Test Subject",
		Body:    "Test body",
	}

	notification, err := svc.SendNotification(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusFailed, notification.Status)
	assert.Equal(t, 1, notification.RetryCount)

	repo.AssertExpectations(t)
	mockSnd.AssertExpectations(t)
}

func TestSendNotification_EmptyUserID(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	input := &SendNotificationInput{
		UserID:  "",
		Type:    domain.NotificationTypeEmail,
		Channel: "email",
		Body:    "Test body",
	}

	notification, err := svc.SendNotification(ctx, input)

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSendNotification_EmptyType(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    "",
		Channel: "email",
		Body:    "Test body",
	}

	notification, err := svc.SendNotification(ctx, input)

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSendNotification_InvalidType(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    "invalid_type",
		Channel: "email",
		Body:    "Test body",
	}

	notification, err := svc.SendNotification(ctx, input)

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSendNotification_EmptyChannel(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    domain.NotificationTypeEmail,
		Channel: "",
		Body:    "Test body",
	}

	notification, err := svc.SendNotification(ctx, input)

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSendNotification_EmptyBody(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    domain.NotificationTypeEmail,
		Channel: "email",
		Body:    "",
	}

	notification, err := svc.SendNotification(ctx, input)

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSendNotification_InvalidPriority(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	input := &SendNotificationInput{
		UserID:   "user-123",
		Type:     domain.NotificationTypeEmail,
		Channel:  "email",
		Body:     "Test body",
		Priority: "super_urgent",
	}

	notification, err := svc.SendNotification(ctx, input)

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestSendNotification_DefaultPriority(t *testing.T) {
	repo := new(mockNotificationRepository)
	mockSnd := new(mockSender)
	senders := map[string]sender.Sender{"email": mockSnd}
	svc := newTestService(repo, senders)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	mockSnd.On("Send", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    domain.NotificationTypeEmail,
		Channel: "email",
		Body:    "Test body",
	}

	notification, err := svc.SendNotification(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationPriorityNormal, notification.Priority)

	repo.AssertExpectations(t)
}

func TestSendNotification_NilMetadata(t *testing.T) {
	repo := new(mockNotificationRepository)
	mockSnd := new(mockSender)
	senders := map[string]sender.Sender{"email": mockSnd}
	svc := newTestService(repo, senders)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	mockSnd.On("Send", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	input := &SendNotificationInput{
		UserID:   "user-123",
		Type:     domain.NotificationTypeEmail,
		Channel:  "email",
		Body:     "Test body",
		Metadata: nil,
	}

	notification, err := svc.SendNotification(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, notification.Metadata)
	assert.Empty(t, notification.Metadata)

	repo.AssertExpectations(t)
}

func TestGetNotification_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	expected := newTestNotification(domain.NotificationStatusSent)
	repo.On("GetByID", ctx, "test-notification-id").Return(expected, nil)

	notification, err := svc.GetNotification(ctx, "test-notification-id")

	require.NoError(t, err)
	assert.Equal(t, expected, notification)

	repo.AssertExpectations(t)
}

func TestGetNotification_NotFound(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	notification, err := svc.GetNotification(ctx, "nonexistent")

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestListNotificationsByUser_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	expectedNotifications := []domain.Notification{
		*newTestNotification(domain.NotificationStatusSent),
		*newTestNotification(domain.NotificationStatusRead),
	}

	repo.On("ListByUserID", ctx, "user-123", 0, 20).Return(expectedNotifications, 2, nil)

	notifications, total, err := svc.ListNotificationsByUser(ctx, "user-123", 1, 20)

	require.NoError(t, err)
	assert.Len(t, notifications, 2)
	assert.Equal(t, 2, total)

	repo.AssertExpectations(t)
}

func TestListNotificationsByUser_DefaultPagination(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("ListByUserID", ctx, "user-123", 0, 20).Return([]domain.Notification{}, 0, nil)

	notifications, total, err := svc.ListNotificationsByUser(ctx, "user-123", 0, 0)

	require.NoError(t, err)
	assert.Empty(t, notifications)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestMarkAsRead_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	existing := newTestNotification(domain.NotificationStatusSent)
	repo.On("GetByID", ctx, "test-notification-id").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	notification, err := svc.MarkAsRead(ctx, "test-notification-id")

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusRead, notification.Status)
	assert.NotNil(t, notification.ReadAt)

	repo.AssertExpectations(t)
}

func TestMarkAsRead_NotFound(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	notification, err := svc.MarkAsRead(ctx, "nonexistent")

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestRetryNotification_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	mockSnd := new(mockSender)
	senders := map[string]sender.Sender{"email": mockSnd}
	svc := newTestService(repo, senders)
	ctx := context.Background()

	existing := newTestNotification(domain.NotificationStatusFailed)
	existing.RetryCount = 1

	repo.On("GetByID", ctx, "test-notification-id").Return(existing, nil)
	// First Update: setting status to pending, incrementing retry count.
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	mockSnd.On("Send", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	notification, err := svc.RetryNotification(ctx, "test-notification-id")

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusSent, notification.Status)
	assert.Equal(t, 2, notification.RetryCount)
	assert.NotNil(t, notification.SentAt)

	repo.AssertExpectations(t)
	mockSnd.AssertExpectations(t)
}

func TestRetryNotification_MaxRetriesExceeded(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	existing := newTestNotification(domain.NotificationStatusFailed)
	existing.RetryCount = 3
	existing.MaxRetries = 3

	repo.On("GetByID", ctx, "test-notification-id").Return(existing, nil)

	notification, err := svc.RetryNotification(ctx, "test-notification-id")

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestRetryNotification_NotInFailedState(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	existing := newTestNotification(domain.NotificationStatusSent)

	repo.On("GetByID", ctx, "test-notification-id").Return(existing, nil)

	notification, err := svc.RetryNotification(ctx, "test-notification-id")

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	repo.AssertExpectations(t)
}

func TestRetryNotification_NotFound(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	notification, err := svc.RetryNotification(ctx, "nonexistent")

	assert.Nil(t, notification)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestListPendingNotifications_Success(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	expectedNotifications := []domain.Notification{
		*newTestNotification(domain.NotificationStatusPending),
	}

	repo.On("ListPending", ctx, 50).Return(expectedNotifications, nil)

	notifications, err := svc.ListPendingNotifications(ctx, 50)

	require.NoError(t, err)
	assert.Len(t, notifications, 1)

	repo.AssertExpectations(t)
}

func TestListPendingNotifications_DefaultLimit(t *testing.T) {
	repo := new(mockNotificationRepository)
	svc := newTestService(repo, nil)
	ctx := context.Background()

	repo.On("ListPending", ctx, 50).Return([]domain.Notification{}, nil)

	notifications, err := svc.ListPendingNotifications(ctx, 0)

	require.NoError(t, err)
	assert.Empty(t, notifications)

	repo.AssertExpectations(t)
}

func TestSendNotification_NoSenderForChannel(t *testing.T) {
	repo := new(mockNotificationRepository)
	senders := map[string]sender.Sender{} // no senders registered
	svc := newTestService(repo, senders)
	ctx := context.Background()

	repo.On("Create", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.Notification")).Return(nil)

	input := &SendNotificationInput{
		UserID:  "user-123",
		Type:    domain.NotificationTypeSMS,
		Channel: "sms",
		Body:    "Test SMS",
	}

	notification, err := svc.SendNotification(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusFailed, notification.Status)

	repo.AssertExpectations(t)
}
