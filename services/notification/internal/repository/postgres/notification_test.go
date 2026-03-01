package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
)

// helper to build a sample notification for tests.
func sampleNotification() *domain.Notification {
	now := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	sentAt := time.Date(2025, 7, 1, 12, 1, 0, 0, time.UTC)
	readAt := time.Date(2025, 7, 1, 12, 5, 0, 0, time.UTC)

	return &domain.Notification{
		ID:         "notif-001",
		UserID:     "usr-001",
		Type:       domain.NotificationTypeEmail,
		Channel:    domain.ChannelEmail,
		Subject:    "Order Shipped",
		Body:       "Your order #123 has been shipped.",
		Status:     domain.NotificationStatusPending,
		Priority:   domain.NotificationPriorityNormal,
		Metadata:   map[string]any{"order_id": "ord-123", "tracking": "TRK456"},
		SentAt:     &sentAt,
		ReadAt:     &readAt,
		RetryCount: 0,
		MaxRetries: domain.DefaultMaxRetries,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

var notificationColumns = []string{
	"id", "user_id", "type", "channel", "subject", "body",
	"status", "priority", "metadata", "sent_at", "read_at",
	"retry_count", "max_retries", "created_at", "updated_at",
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestNotificationRepository_Create_Success(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)
	n := sampleNotification()

	metadataJSON, err := json.Marshal(n.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO notifications").
		WithArgs(
			n.ID, n.UserID, n.Type, n.Channel, n.Subject, n.Body,
			n.Status, n.Priority, metadataJSON,
			n.SentAt, n.ReadAt, n.RetryCount, n.MaxRetries,
			n.CreatedAt, n.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), n)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_Create_ExecError(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)
	n := sampleNotification()

	metadataJSON, err := json.Marshal(n.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO notifications").
		WithArgs(
			n.ID, n.UserID, n.Type, n.Channel, n.Subject, n.Body,
			n.Status, n.Priority, metadataJSON,
			n.SentAt, n.ReadAt, n.RetryCount, n.MaxRetries,
			n.CreatedAt, n.UpdatedAt,
		).
		WillReturnError(errors.New("connection refused"))

	err = repo.Create(context.Background(), n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert notification")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestNotificationRepository_GetByID_Success(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)
	n := sampleNotification()

	metadataJSON, err := json.Marshal(n.Metadata)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(n.ID).
		WillReturnRows(
			pgxmock.NewRows(notificationColumns).
				AddRow(
					n.ID, n.UserID, n.Type, n.Channel, n.Subject, n.Body,
					n.Status, n.Priority, metadataJSON,
					n.SentAt, n.ReadAt, n.RetryCount, n.MaxRetries,
					n.CreatedAt, n.UpdatedAt,
				),
		)

	result, err := repo.GetByID(context.Background(), n.ID)
	require.NoError(t, err)
	assert.Equal(t, n.ID, result.ID)
	assert.Equal(t, n.UserID, result.UserID)
	assert.Equal(t, n.Type, result.Type)
	assert.Equal(t, n.Channel, result.Channel)
	assert.Equal(t, n.Subject, result.Subject)
	assert.Equal(t, n.Body, result.Body)
	assert.Equal(t, n.Status, result.Status)
	assert.Equal(t, n.Priority, result.Priority)
	assert.Equal(t, "ord-123", result.Metadata["order_id"])
	assert.Equal(t, "TRK456", result.Metadata["tracking"])
	assert.NotNil(t, result.SentAt)
	assert.NotNil(t, result.ReadAt)
	assert.Equal(t, n.RetryCount, result.RetryCount)
	assert.Equal(t, n.MaxRetries, result.MaxRetries)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_GetByID_NotFound(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs("nonexistent-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_GetByID_ScanError(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs("notif-bad").
		WillReturnError(errors.New("unexpected column type"))

	result, err := repo.GetByID(context.Background(), "notif-bad")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan notification")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestNotificationRepository_Update_Success(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)
	n := sampleNotification()
	n.Status = domain.NotificationStatusSent

	metadataJSON, err := json.Marshal(n.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE notifications").
		WithArgs(
			n.UserID, n.Type, n.Channel, n.Subject, n.Body,
			n.Status, n.Priority, metadataJSON,
			n.SentAt, n.ReadAt, n.RetryCount, n.MaxRetries,
			pgxmock.AnyArg(), // UpdatedAt is set at call time
			n.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), n)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_Update_NotFound(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)
	n := sampleNotification()
	n.ID = "nonexistent-notif-id"

	metadataJSON, err := json.Marshal(n.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE notifications").
		WithArgs(
			n.UserID, n.Type, n.Channel, n.Subject, n.Body,
			n.Status, n.Priority, metadataJSON,
			n.SentAt, n.ReadAt, n.RetryCount, n.MaxRetries,
			pgxmock.AnyArg(), // UpdatedAt
			n.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = repo.Update(context.Background(), n)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_Update_ExecError(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)
	n := sampleNotification()

	metadataJSON, err := json.Marshal(n.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE notifications").
		WithArgs(
			n.UserID, n.Type, n.Channel, n.Subject, n.Body,
			n.Status, n.Priority, metadataJSON,
			n.SentAt, n.ReadAt, n.RetryCount, n.MaxRetries,
			pgxmock.AnyArg(), // UpdatedAt
			n.ID,
		).
		WillReturnError(errors.New("connection lost"))

	err = repo.Update(context.Background(), n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update notification")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── ListByUserID ────────────────────────────────────────────────────────────

func TestNotificationRepository_ListByUserID_Success(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	now := time.Now().UTC()
	sentAt := time.Date(2025, 7, 1, 12, 1, 0, 0, time.UTC)

	meta1 := map[string]any{"order_id": "ord-001"}
	meta1JSON, err := json.Marshal(meta1)
	require.NoError(t, err)

	meta2 := map[string]any{"order_id": "ord-002"}
	meta2JSON, err := json.Marshal(meta2)
	require.NoError(t, err)

	listColumns := append(notificationColumns, "total_count")

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs("usr-001", 10, 0).
		WillReturnRows(
			pgxmock.NewRows(listColumns).
				AddRow(
					"notif-001", "usr-001", "email", "email", "Subject 1", "Body 1",
					"pending", "normal", meta1JSON,
					&sentAt, nil, 0, 3,
					now, now,
					2, // total_count
				).
				AddRow(
					"notif-002", "usr-001", "sms", "sms", "Subject 2", "Body 2",
					"sent", "high", meta2JSON,
					nil, nil, 1, 3,
					now, now,
					2, // total_count
				),
		)

	notifications, total, err := repo.ListByUserID(context.Background(), "usr-001", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, notifications, 2)

	assert.Equal(t, "notif-001", notifications[0].ID)
	assert.Equal(t, "usr-001", notifications[0].UserID)
	assert.Equal(t, "email", notifications[0].Type)
	assert.Equal(t, "Subject 1", notifications[0].Subject)
	assert.Equal(t, "ord-001", notifications[0].Metadata["order_id"])
	assert.NotNil(t, notifications[0].SentAt)
	assert.Nil(t, notifications[0].ReadAt)

	assert.Equal(t, "notif-002", notifications[1].ID)
	assert.Equal(t, "sms", notifications[1].Type)
	assert.Equal(t, "high", notifications[1].Priority)
	assert.Equal(t, "ord-002", notifications[1].Metadata["order_id"])
	assert.Nil(t, notifications[1].SentAt)
	assert.Equal(t, 1, notifications[1].RetryCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_ListByUserID_Empty(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	listColumns := append(notificationColumns, "total_count")

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs("usr-999", 20, 0).
		WillReturnRows(pgxmock.NewRows(listColumns))

	notifications, total, err := repo.ListByUserID(context.Background(), "usr-999", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NotNil(t, notifications)
	assert.Empty(t, notifications)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_ListByUserID_QueryError(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs("usr-001", 10, 0).
		WillReturnError(errors.New("database timeout"))

	notifications, total, err := repo.ListByUserID(context.Background(), "usr-001", 0, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list notifications by user")
	assert.Nil(t, notifications)
	assert.Equal(t, 0, total)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── ListPending ─────────────────────────────────────────────────────────────

func TestNotificationRepository_ListPending_Success(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	now := time.Now().UTC()

	meta1 := map[string]any{"order_id": "ord-100"}
	meta1JSON, err := json.Marshal(meta1)
	require.NoError(t, err)

	meta2 := map[string]any{"order_id": "ord-200"}
	meta2JSON, err := json.Marshal(meta2)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(domain.NotificationStatusPending, 50).
		WillReturnRows(
			pgxmock.NewRows(notificationColumns).
				AddRow(
					"notif-p1", "usr-001", "email", "email", "Pending 1", "Body pending 1",
					"pending", "normal", meta1JSON,
					nil, nil, 0, 3,
					now, now,
				).
				AddRow(
					"notif-p2", "usr-002", "sms", "sms", "Pending 2", "Body pending 2",
					"pending", "high", meta2JSON,
					nil, nil, 1, 3,
					now, now,
				),
		)

	notifications, err := repo.ListPending(context.Background(), 50)
	require.NoError(t, err)
	assert.Len(t, notifications, 2)

	assert.Equal(t, "notif-p1", notifications[0].ID)
	assert.Equal(t, domain.NotificationStatusPending, notifications[0].Status)
	assert.Equal(t, "ord-100", notifications[0].Metadata["order_id"])

	assert.Equal(t, "notif-p2", notifications[1].ID)
	assert.Equal(t, domain.NotificationStatusPending, notifications[1].Status)
	assert.Equal(t, "ord-200", notifications[1].Metadata["order_id"])
	assert.Equal(t, 1, notifications[1].RetryCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_ListPending_Empty(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(domain.NotificationStatusPending, 10).
		WillReturnRows(pgxmock.NewRows(notificationColumns))

	notifications, err := repo.ListPending(context.Background(), 10)
	require.NoError(t, err)
	assert.NotNil(t, notifications)
	assert.Empty(t, notifications)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── ListFailed ──────────────────────────────────────────────────────────────

func TestNotificationRepository_ListFailed_Success(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	now := time.Now().UTC()

	meta1 := map[string]any{"error": "smtp timeout"}
	meta1JSON, err := json.Marshal(meta1)
	require.NoError(t, err)

	meta2 := map[string]any{"error": "invalid recipient"}
	meta2JSON, err := json.Marshal(meta2)
	require.NoError(t, err)

	sentAt := time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(domain.NotificationStatusFailed, 25).
		WillReturnRows(
			pgxmock.NewRows(notificationColumns).
				AddRow(
					"notif-f1", "usr-010", "email", "email", "Failed 1", "Body failed 1",
					"failed", "urgent", meta1JSON,
					&sentAt, nil, 3, 3,
					now, now,
				).
				AddRow(
					"notif-f2", "usr-020", "push", "push", "Failed 2", "Body failed 2",
					"failed", "normal", meta2JSON,
					nil, nil, 2, 3,
					now, now,
				),
		)

	notifications, err := repo.ListFailed(context.Background(), 25)
	require.NoError(t, err)
	assert.Len(t, notifications, 2)

	assert.Equal(t, "notif-f1", notifications[0].ID)
	assert.Equal(t, domain.NotificationStatusFailed, notifications[0].Status)
	assert.Equal(t, "smtp timeout", notifications[0].Metadata["error"])
	assert.NotNil(t, notifications[0].SentAt)
	assert.Equal(t, 3, notifications[0].RetryCount)

	assert.Equal(t, "notif-f2", notifications[1].ID)
	assert.Equal(t, domain.NotificationStatusFailed, notifications[1].Status)
	assert.Equal(t, "invalid recipient", notifications[1].Metadata["error"])
	assert.Nil(t, notifications[1].SentAt)
	assert.Equal(t, 2, notifications[1].RetryCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestNotificationRepository_ListFailed_Empty(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewNotificationRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM notifications").
		WithArgs(domain.NotificationStatusFailed, 10).
		WillReturnRows(pgxmock.NewRows(notificationColumns))

	notifications, err := repo.ListFailed(context.Background(), 10)
	require.NoError(t, err)
	assert.NotNil(t, notifications)
	assert.Empty(t, notifications)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
