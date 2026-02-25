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
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
)

// CheckoutRepository implements repository.CheckoutRepository using PostgreSQL.
type CheckoutRepository struct {
	pool *pgxpool.Pool
}

// NewCheckoutRepository creates a new PostgreSQL-backed checkout repository.
func NewCheckoutRepository(pool *pgxpool.Pool) *CheckoutRepository {
	return &CheckoutRepository{pool: pool}
}

// Create inserts a new checkout session into the database.
func (r *CheckoutRepository) Create(ctx context.Context, session *domain.CheckoutSession) error {
	itemsJSON, err := json.Marshal(session.Items)
	if err != nil {
		return fmt.Errorf("marshal items: %w", err)
	}

	shippingJSON, err := json.Marshal(session.ShippingAddress)
	if err != nil {
		return fmt.Errorf("marshal shipping address: %w", err)
	}

	billingJSON, err := json.Marshal(session.BillingAddress)
	if err != nil {
		return fmt.Errorf("marshal billing address: %w", err)
	}

	query := `
		INSERT INTO checkout_sessions (
			id, user_id, status, items,
			subtotal_amount, discount_amount, shipping_amount, total_amount,
			currency, shipping_address, billing_address,
			payment_method, payment_id, order_id, failure_reason,
			expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18
		)`

	_, err = r.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.Status,
		itemsJSON,
		session.SubtotalAmount,
		session.DiscountAmount,
		session.ShippingAmount,
		session.TotalAmount,
		session.Currency,
		shippingJSON,
		billingJSON,
		nullableString(session.PaymentMethod),
		nullableString(session.PaymentID),
		nullableString(session.OrderID),
		nullableString(session.FailureReason),
		session.ExpiresAt,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert checkout session: %w", err)
	}

	return nil
}

// GetByID retrieves a checkout session by its ID.
func (r *CheckoutRepository) GetByID(ctx context.Context, id string) (*domain.CheckoutSession, error) {
	query := `
		SELECT id, user_id, status, items,
			subtotal_amount, discount_amount, shipping_amount, total_amount,
			currency, shipping_address, billing_address,
			payment_method, payment_id, order_id, failure_reason,
			expires_at, created_at, updated_at
		FROM checkout_sessions
		WHERE id = $1`

	return r.scanSession(ctx, query, id)
}

// Update modifies an existing checkout session in the database.
func (r *CheckoutRepository) Update(ctx context.Context, session *domain.CheckoutSession) error {
	itemsJSON, err := json.Marshal(session.Items)
	if err != nil {
		return fmt.Errorf("marshal items: %w", err)
	}

	shippingJSON, err := json.Marshal(session.ShippingAddress)
	if err != nil {
		return fmt.Errorf("marshal shipping address: %w", err)
	}

	billingJSON, err := json.Marshal(session.BillingAddress)
	if err != nil {
		return fmt.Errorf("marshal billing address: %w", err)
	}

	session.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE checkout_sessions
		SET status = $1, items = $2,
			subtotal_amount = $3, discount_amount = $4, shipping_amount = $5, total_amount = $6,
			currency = $7, shipping_address = $8, billing_address = $9,
			payment_method = $10, payment_id = $11, order_id = $12, failure_reason = $13,
			expires_at = $14, updated_at = $15
		WHERE id = $16`

	ct, err := r.pool.Exec(ctx, query,
		session.Status,
		itemsJSON,
		session.SubtotalAmount,
		session.DiscountAmount,
		session.ShippingAmount,
		session.TotalAmount,
		session.Currency,
		shippingJSON,
		billingJSON,
		nullableString(session.PaymentMethod),
		nullableString(session.PaymentID),
		nullableString(session.OrderID),
		nullableString(session.FailureReason),
		session.ExpiresAt,
		session.UpdatedAt,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("update checkout session: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("checkout_session", session.ID)
	}

	return nil
}

// GetActiveByUserID retrieves the active checkout session for a user.
func (r *CheckoutRepository) GetActiveByUserID(ctx context.Context, userID string) (*domain.CheckoutSession, error) {
	query := `
		SELECT id, user_id, status, items,
			subtotal_amount, discount_amount, shipping_amount, total_amount,
			currency, shipping_address, billing_address,
			payment_method, payment_id, order_id, failure_reason,
			expires_at, created_at, updated_at
		FROM checkout_sessions
		WHERE user_id = $1 AND status NOT IN ('completed', 'failed', 'expired')
		ORDER BY created_at DESC
		LIMIT 1`

	return r.scanSession(ctx, query, userID)
}

// ListExpired returns checkout sessions that have expired before the given time.
func (r *CheckoutRepository) ListExpired(ctx context.Context, before time.Time) ([]domain.CheckoutSession, error) {
	query := `
		SELECT id, user_id, status, items,
			subtotal_amount, discount_amount, shipping_amount, total_amount,
			currency, shipping_address, billing_address,
			payment_method, payment_id, order_id, failure_reason,
			expires_at, created_at, updated_at
		FROM checkout_sessions
		WHERE expires_at < $1 AND status NOT IN ('completed', 'failed', 'expired')
		ORDER BY expires_at ASC`

	rows, err := r.pool.Query(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("list expired sessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.CheckoutSession
	for rows.Next() {
		session, err := r.scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expired session row: %w", err)
		}
		sessions = append(sessions, *session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired session rows: %w", err)
	}

	if sessions == nil {
		sessions = []domain.CheckoutSession{}
	}

	return sessions, nil
}

// scanSession executes a query expected to return a single checkout session row.
func (r *CheckoutRepository) scanSession(ctx context.Context, query string, args ...any) (*domain.CheckoutSession, error) {
	var (
		session      domain.CheckoutSession
		itemsJSON    []byte
		shippingJSON []byte
		billingJSON  []byte
		paymentMethod *string
		paymentID     *string
		orderID       *string
		failureReason *string
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&session.ID,
		&session.UserID,
		&session.Status,
		&itemsJSON,
		&session.SubtotalAmount,
		&session.DiscountAmount,
		&session.ShippingAmount,
		&session.TotalAmount,
		&session.Currency,
		&shippingJSON,
		&billingJSON,
		&paymentMethod,
		&paymentID,
		&orderID,
		&failureReason,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan checkout session: %w", err)
	}

	if err := r.unmarshalFields(&session, itemsJSON, shippingJSON, billingJSON); err != nil {
		return nil, err
	}

	if paymentMethod != nil {
		session.PaymentMethod = *paymentMethod
	}
	if paymentID != nil {
		session.PaymentID = *paymentID
	}
	if orderID != nil {
		session.OrderID = *orderID
	}
	if failureReason != nil {
		session.FailureReason = *failureReason
	}

	return &session, nil
}

// scanRow scans a single row from a rows result set.
func (r *CheckoutRepository) scanRow(rows pgx.Rows) (*domain.CheckoutSession, error) {
	var (
		session       domain.CheckoutSession
		itemsJSON     []byte
		shippingJSON  []byte
		billingJSON   []byte
		paymentMethod *string
		paymentID     *string
		orderID       *string
		failureReason *string
	)

	if err := rows.Scan(
		&session.ID,
		&session.UserID,
		&session.Status,
		&itemsJSON,
		&session.SubtotalAmount,
		&session.DiscountAmount,
		&session.ShippingAmount,
		&session.TotalAmount,
		&session.Currency,
		&shippingJSON,
		&billingJSON,
		&paymentMethod,
		&paymentID,
		&orderID,
		&failureReason,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan checkout session row: %w", err)
	}

	if err := r.unmarshalFields(&session, itemsJSON, shippingJSON, billingJSON); err != nil {
		return nil, err
	}

	if paymentMethod != nil {
		session.PaymentMethod = *paymentMethod
	}
	if paymentID != nil {
		session.PaymentID = *paymentID
	}
	if orderID != nil {
		session.OrderID = *orderID
	}
	if failureReason != nil {
		session.FailureReason = *failureReason
	}

	return &session, nil
}

// unmarshalFields deserializes JSON fields on the session.
func (r *CheckoutRepository) unmarshalFields(session *domain.CheckoutSession, itemsJSON, shippingJSON, billingJSON []byte) error {
	if itemsJSON != nil {
		if err := json.Unmarshal(itemsJSON, &session.Items); err != nil {
			return fmt.Errorf("unmarshal items: %w", err)
		}
	}
	if session.Items == nil {
		session.Items = []domain.CheckoutItem{}
	}

	if shippingJSON != nil && string(shippingJSON) != "null" {
		var addr domain.Address
		if err := json.Unmarshal(shippingJSON, &addr); err != nil {
			return fmt.Errorf("unmarshal shipping address: %w", err)
		}
		session.ShippingAddress = &addr
	}

	if billingJSON != nil && string(billingJSON) != "null" {
		var addr domain.Address
		if err := json.Unmarshal(billingJSON, &addr); err != nil {
			return fmt.Errorf("unmarshal billing address: %w", err)
		}
		session.BillingAddress = &addr
	}

	return nil
}

// nullableString returns nil if the string is empty, otherwise a pointer to the string.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
