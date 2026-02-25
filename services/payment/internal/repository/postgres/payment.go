package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
)

// PaymentRepository implements repository.PaymentRepository using PostgreSQL.
type PaymentRepository struct {
	pool *pgxpool.Pool
}

// NewPaymentRepository creates a new PostgreSQL-backed payment repository.
func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

// Create inserts a new payment into the database.
func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO payments (id, checkout_id, order_id, user_id, amount, currency, status, method, provider_name, provider_payment_id, failure_reason, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err = r.pool.Exec(ctx, query,
		p.ID,
		p.CheckoutID,
		p.OrderID,
		p.UserID,
		p.Amount,
		p.Currency,
		p.Status,
		p.Method,
		p.ProviderName,
		p.ProviderPayID,
		p.FailureReason,
		metadataJSON,
		p.CreatedAt,
		p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	return nil
}

// GetByID retrieves a payment by its ID.
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, checkout_id, order_id, user_id, amount, currency, status, method, provider_name, provider_payment_id, failure_reason, metadata, created_at, updated_at
		FROM payments
		WHERE id = $1`

	return r.scanPayment(ctx, query, id)
}

// GetByCheckoutID retrieves a payment by its checkout ID.
func (r *PaymentRepository) GetByCheckoutID(ctx context.Context, checkoutID string) (*domain.Payment, error) {
	query := `
		SELECT id, checkout_id, order_id, user_id, amount, currency, status, method, provider_name, provider_payment_id, failure_reason, metadata, created_at, updated_at
		FROM payments
		WHERE checkout_id = $1`

	return r.scanPayment(ctx, query, checkoutID)
}

// Update modifies an existing payment in the database.
func (r *PaymentRepository) Update(ctx context.Context, p *domain.Payment) error {
	metadataJSON, err := json.Marshal(p.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	p.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE payments
		SET checkout_id = $1, order_id = $2, user_id = $3, amount = $4, currency = $5,
		    status = $6, method = $7, provider_name = $8, provider_payment_id = $9,
		    failure_reason = $10, metadata = $11, updated_at = $12
		WHERE id = $13`

	ct, err := r.pool.Exec(ctx, query,
		p.CheckoutID,
		p.OrderID,
		p.UserID,
		p.Amount,
		p.Currency,
		p.Status,
		p.Method,
		p.ProviderName,
		p.ProviderPayID,
		p.FailureReason,
		metadataJSON,
		p.UpdatedAt,
		p.ID,
	)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("payment", p.ID)
	}

	return nil
}

// ListByUserID returns payments for a given user with pagination.
func (r *PaymentRepository) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Payment, int, error) {
	query := `
		SELECT id, checkout_id, order_id, user_id, amount, currency, status, method, provider_name, provider_payment_id, failure_reason, metadata, created_at, updated_at,
		       count(*) OVER() AS total_count
		FROM payments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list payments by user: %w", err)
	}
	defer rows.Close()

	var (
		payments   []domain.Payment
		totalCount int
	)

	for rows.Next() {
		var (
			p            domain.Payment
			metadataJSON []byte
		)

		if err := rows.Scan(
			&p.ID,
			&p.CheckoutID,
			&p.OrderID,
			&p.UserID,
			&p.Amount,
			&p.Currency,
			&p.Status,
			&p.Method,
			&p.ProviderName,
			&p.ProviderPayID,
			&p.FailureReason,
			&metadataJSON,
			&p.CreatedAt,
			&p.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan payment row: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
				return nil, 0, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate payment rows: %w", err)
	}

	if payments == nil {
		payments = []domain.Payment{}
	}

	return payments, totalCount, nil
}

// CreateRefund inserts a new refund into the database.
func (r *PaymentRepository) CreateRefund(ctx context.Context, ref *domain.Refund) error {
	query := `
		INSERT INTO refunds (id, payment_id, amount, currency, status, reason, provider_refund_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		ref.ID,
		ref.PaymentID,
		ref.Amount,
		ref.Currency,
		ref.Status,
		ref.Reason,
		ref.ProviderRefID,
		ref.CreatedAt,
		ref.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert refund: %w", err)
	}

	return nil
}

// GetRefundByID retrieves a refund by its ID.
func (r *PaymentRepository) GetRefundByID(ctx context.Context, id string) (*domain.Refund, error) {
	query := `
		SELECT id, payment_id, amount, currency, status, reason, provider_refund_id, created_at, updated_at
		FROM refunds
		WHERE id = $1`

	return r.scanRefund(ctx, query, id)
}

// ListRefundsByPaymentID returns all refunds for a given payment.
func (r *PaymentRepository) ListRefundsByPaymentID(ctx context.Context, paymentID string) ([]domain.Refund, error) {
	query := `
		SELECT id, payment_id, amount, currency, status, reason, provider_refund_id, created_at, updated_at
		FROM refunds
		WHERE payment_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, paymentID)
	if err != nil {
		return nil, fmt.Errorf("list refunds by payment: %w", err)
	}
	defer rows.Close()

	var refunds []domain.Refund
	for rows.Next() {
		var ref domain.Refund
		if err := rows.Scan(
			&ref.ID,
			&ref.PaymentID,
			&ref.Amount,
			&ref.Currency,
			&ref.Status,
			&ref.Reason,
			&ref.ProviderRefID,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan refund row: %w", err)
		}
		refunds = append(refunds, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate refund rows: %w", err)
	}

	if refunds == nil {
		refunds = []domain.Refund{}
	}

	return refunds, nil
}

// UpdateRefund modifies an existing refund in the database.
func (r *PaymentRepository) UpdateRefund(ctx context.Context, ref *domain.Refund) error {
	ref.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE refunds
		SET amount = $1, currency = $2, status = $3, reason = $4, provider_refund_id = $5, updated_at = $6
		WHERE id = $7`

	ct, err := r.pool.Exec(ctx, query,
		ref.Amount,
		ref.Currency,
		ref.Status,
		ref.Reason,
		ref.ProviderRefID,
		ref.UpdatedAt,
		ref.ID,
	)
	if err != nil {
		return fmt.Errorf("update refund: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("refund", ref.ID)
	}

	return nil
}

// scanPayment executes a query expected to return a single payment row.
func (r *PaymentRepository) scanPayment(ctx context.Context, query string, args ...any) (*domain.Payment, error) {
	var (
		p            domain.Payment
		metadataJSON []byte
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&p.ID,
		&p.CheckoutID,
		&p.OrderID,
		&p.UserID,
		&p.Amount,
		&p.Currency,
		&p.Status,
		&p.Method,
		&p.ProviderName,
		&p.ProviderPayID,
		&p.FailureReason,
		&metadataJSON,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan payment: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &p.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return &p, nil
}

// scanRefund executes a query expected to return a single refund row.
func (r *PaymentRepository) scanRefund(ctx context.Context, query string, args ...any) (*domain.Refund, error) {
	var ref domain.Refund

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&ref.ID,
		&ref.PaymentID,
		&ref.Amount,
		&ref.Currency,
		&ref.Status,
		&ref.Reason,
		&ref.ProviderRefID,
		&ref.CreatedAt,
		&ref.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan refund: %w", err)
	}

	return &ref, nil
}
