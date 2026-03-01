package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
)

// NotificationRepository implements repository.NotificationRepository using PostgreSQL.
type NotificationRepository struct {
	pool database.DBTX
}

// NewNotificationRepository creates a new PostgreSQL-backed notification repository.
func NewNotificationRepository(pool database.DBTX) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

// Create inserts a new notification into the database.
func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	metadataJSON, err := json.Marshal(n.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO notifications (id, user_id, type, channel, subject, body, status, priority, metadata, sent_at, read_at, retry_count, max_retries, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err = r.pool.Exec(ctx, query,
		n.ID,
		n.UserID,
		n.Type,
		n.Channel,
		n.Subject,
		n.Body,
		n.Status,
		n.Priority,
		metadataJSON,
		n.SentAt,
		n.ReadAt,
		n.RetryCount,
		n.MaxRetries,
		n.CreatedAt,
		n.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	return nil
}

// GetByID retrieves a notification by its ID.
func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, subject, body, status, priority, metadata, sent_at, read_at, retry_count, max_retries, created_at, updated_at
		FROM notifications
		WHERE id = $1`

	return r.scanNotification(ctx, query, id)
}

// Update modifies an existing notification in the database.
func (r *NotificationRepository) Update(ctx context.Context, n *domain.Notification) error {
	metadataJSON, err := json.Marshal(n.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	n.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE notifications
		SET user_id = $1, type = $2, channel = $3, subject = $4, body = $5,
		    status = $6, priority = $7, metadata = $8, sent_at = $9, read_at = $10,
		    retry_count = $11, max_retries = $12, updated_at = $13
		WHERE id = $14`

	ct, err := r.pool.Exec(ctx, query,
		n.UserID,
		n.Type,
		n.Channel,
		n.Subject,
		n.Body,
		n.Status,
		n.Priority,
		metadataJSON,
		n.SentAt,
		n.ReadAt,
		n.RetryCount,
		n.MaxRetries,
		n.UpdatedAt,
		n.ID,
	)
	if err != nil {
		return fmt.Errorf("update notification: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("notification", n.ID)
	}

	return nil
}

// ListByUserID returns notifications for a specific user with pagination.
func (r *NotificationRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Notification, int, error) {
	query := `
		SELECT id, user_id, type, channel, subject, body, status, priority, metadata, sent_at, read_at, retry_count, max_retries, created_at, updated_at,
		       count(*) OVER() AS total_count
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications by user: %w", err)
	}
	defer rows.Close()

	var totalCount int
	notifications := make([]domain.Notification, 0)

	for rows.Next() {
		var (
			n            domain.Notification
			metadataJSON []byte
		)

		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Type,
			&n.Channel,
			&n.Subject,
			&n.Body,
			&n.Status,
			&n.Priority,
			&metadataJSON,
			&n.SentAt,
			&n.ReadAt,
			&n.RetryCount,
			&n.MaxRetries,
			&n.CreatedAt,
			&n.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan notification row: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
				return nil, 0, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate notification rows: %w", err)
	}

	return notifications, totalCount, nil
}

// ListPending returns pending notifications up to the given limit.
func (r *NotificationRepository) ListPending(ctx context.Context, limit int) ([]domain.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, subject, body, status, priority, metadata, sent_at, read_at, retry_count, max_retries, created_at, updated_at
		FROM notifications
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	return r.scanNotifications(ctx, query, domain.NotificationStatusPending, limit)
}

// ListFailed returns failed notifications up to the given limit.
func (r *NotificationRepository) ListFailed(ctx context.Context, limit int) ([]domain.Notification, error) {
	query := `
		SELECT id, user_id, type, channel, subject, body, status, priority, metadata, sent_at, read_at, retry_count, max_retries, created_at, updated_at
		FROM notifications
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	return r.scanNotifications(ctx, query, domain.NotificationStatusFailed, limit)
}

// scanNotification is a helper that executes a query expected to return a single notification row.
func (r *NotificationRepository) scanNotification(ctx context.Context, query string, args ...any) (*domain.Notification, error) {
	var (
		n            domain.Notification
		metadataJSON []byte
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&n.ID,
		&n.UserID,
		&n.Type,
		&n.Channel,
		&n.Subject,
		&n.Body,
		&n.Status,
		&n.Priority,
		&metadataJSON,
		&n.SentAt,
		&n.ReadAt,
		&n.RetryCount,
		&n.MaxRetries,
		&n.CreatedAt,
		&n.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan notification: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return &n, nil
}

// scanNotifications is a helper that executes a query expected to return multiple notification rows.
func (r *NotificationRepository) scanNotifications(ctx context.Context, query string, args ...any) ([]domain.Notification, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]domain.Notification, 0)

	for rows.Next() {
		var (
			n            domain.Notification
			metadataJSON []byte
		)

		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Type,
			&n.Channel,
			&n.Subject,
			&n.Body,
			&n.Status,
			&n.Priority,
			&metadataJSON,
			&n.SentAt,
			&n.ReadAt,
			&n.RetryCount,
			&n.MaxRetries,
			&n.CreatedAt,
			&n.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification row: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &n.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification rows: %w", err)
	}

	return notifications, nil
}
