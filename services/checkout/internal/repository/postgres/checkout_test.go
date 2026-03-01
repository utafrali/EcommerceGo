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
	"github.com/utafrali/EcommerceGo/services/checkout/internal/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestRepo(t *testing.T) (*CheckoutRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	repo := NewCheckoutRepository(mock)
	return repo, mock
}

func sampleSession() *domain.CheckoutSession {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &domain.CheckoutSession{
		ID:     "checkout-001",
		UserID: "user-001",
		Status: domain.StatusInitiated,
		Items: []domain.CheckoutItem{
			{
				ProductID:     "prod-001",
				VariantID:     "var-001",
				Name:          "Widget",
				SKU:           "WDG-001",
				Price:         5000,
				Quantity:      2,
				ReservationID: "res-001",
			},
			{
				ProductID:     "prod-002",
				VariantID:     "var-002",
				Name:          "Gadget",
				SKU:           "GDG-001",
				Price:         2500,
				Quantity:      1,
				ReservationID: "res-002",
			},
		},
		SubtotalAmount: 12500,
		DiscountAmount: 500,
		ShippingAmount: 1000,
		TotalAmount:    13000,
		Currency:       "TRY",
		ShippingAddress: &domain.Address{
			FullName:    "John Doe",
			AddressLine: "123 Main St",
			City:        "Istanbul",
			State:       "Istanbul",
			PostalCode:  "34000",
			Country:     "TR",
			Phone:       "+905551234567",
		},
		BillingAddress: &domain.Address{
			FullName:    "John Doe",
			AddressLine: "456 Side St",
			City:        "Ankara",
			State:       "Ankara",
			PostalCode:  "06000",
			Country:     "TR",
			Phone:       "+905559876543",
		},
		PaymentMethod: "credit_card",
		PaymentID:     "pay-001",
		OrderID:       "ord-001",
		FailureReason: "",
		ExpiresAt:     now.Add(30 * time.Minute),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func sessionColumns() []string {
	return []string{
		"id", "user_id", "status", "items",
		"subtotal_amount", "discount_amount", "shipping_amount", "total_amount",
		"currency", "shipping_address", "billing_address",
		"payment_method", "payment_id", "order_id", "failure_reason",
		"expires_at", "created_at", "updated_at",
	}
}

func sessionRow(t *testing.T, s *domain.CheckoutSession) []any {
	t.Helper()

	itemsJSON, err := json.Marshal(s.Items)
	require.NoError(t, err)

	shippingJSON, err := json.Marshal(s.ShippingAddress)
	require.NoError(t, err)

	billingJSON, err := json.Marshal(s.BillingAddress)
	require.NoError(t, err)

	var paymentMethod, paymentID, orderID, failureReason *string
	if s.PaymentMethod != "" {
		pm := s.PaymentMethod
		paymentMethod = &pm
	}
	if s.PaymentID != "" {
		pid := s.PaymentID
		paymentID = &pid
	}
	if s.OrderID != "" {
		oid := s.OrderID
		orderID = &oid
	}
	if s.FailureReason != "" {
		fr := s.FailureReason
		failureReason = &fr
	}

	return []any{
		s.ID, s.UserID, s.Status, itemsJSON,
		s.SubtotalAmount, s.DiscountAmount, s.ShippingAmount, s.TotalAmount,
		s.Currency, shippingJSON, billingJSON,
		paymentMethod, paymentID, orderID, failureReason,
		s.ExpiresAt, s.CreatedAt, s.UpdatedAt,
	}
}

// strPtr is a convenience helper for creating *string values.
func strPtr(s string) *string {
	return &s
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCheckoutRepository_Create_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()

	itemsJSON, err := json.Marshal(s.Items)
	require.NoError(t, err)
	shippingJSON, err := json.Marshal(s.ShippingAddress)
	require.NoError(t, err)
	billingJSON, err := json.Marshal(s.BillingAddress)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO checkout_sessions").
		WithArgs(
			s.ID, s.UserID, s.Status, itemsJSON,
			s.SubtotalAmount, s.DiscountAmount, s.ShippingAmount, s.TotalAmount,
			s.Currency, shippingJSON, billingJSON,
			strPtr(s.PaymentMethod), strPtr(s.PaymentID), strPtr(s.OrderID), (*string)(nil), // FailureReason is empty -> nil
			s.ExpiresAt, s.CreatedAt, s.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), s)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_Create_ExecError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()

	mock.ExpectExec("INSERT INTO checkout_sessions").
		WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnError(errors.New("duplicate key"))

	err := repo.Create(context.Background(), s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert checkout session")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestCheckoutRepository_GetByID_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()
	row := sessionRow(t, s)

	rows := pgxmock.NewRows(sessionColumns()).AddRow(row...)

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE id").
		WithArgs(s.ID).
		WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), s.ID)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify scalar fields.
	assert.Equal(t, s.ID, result.ID)
	assert.Equal(t, s.UserID, result.UserID)
	assert.Equal(t, s.Status, result.Status)
	assert.Equal(t, s.SubtotalAmount, result.SubtotalAmount)
	assert.Equal(t, s.DiscountAmount, result.DiscountAmount)
	assert.Equal(t, s.ShippingAmount, result.ShippingAmount)
	assert.Equal(t, s.TotalAmount, result.TotalAmount)
	assert.Equal(t, s.Currency, result.Currency)

	// Verify nullable string fields.
	assert.Equal(t, s.PaymentMethod, result.PaymentMethod)
	assert.Equal(t, s.PaymentID, result.PaymentID)
	assert.Equal(t, s.OrderID, result.OrderID)
	assert.Equal(t, "", result.FailureReason) // empty string, was nil in DB

	// Verify JSON-unmarshaled items.
	require.Len(t, result.Items, 2)
	assert.Equal(t, "prod-001", result.Items[0].ProductID)
	assert.Equal(t, "Widget", result.Items[0].Name)
	assert.Equal(t, int64(5000), result.Items[0].Price)
	assert.Equal(t, 2, result.Items[0].Quantity)
	assert.Equal(t, "res-001", result.Items[0].ReservationID)
	assert.Equal(t, "prod-002", result.Items[1].ProductID)
	assert.Equal(t, "Gadget", result.Items[1].Name)
	assert.Equal(t, int64(2500), result.Items[1].Price)
	assert.Equal(t, 1, result.Items[1].Quantity)

	// Verify JSON-unmarshaled addresses.
	require.NotNil(t, result.ShippingAddress)
	assert.Equal(t, "John Doe", result.ShippingAddress.FullName)
	assert.Equal(t, "Istanbul", result.ShippingAddress.City)
	assert.Equal(t, "TR", result.ShippingAddress.Country)

	require.NotNil(t, result.BillingAddress)
	assert.Equal(t, "John Doe", result.BillingAddress.FullName)
	assert.Equal(t, "Ankara", result.BillingAddress.City)

	// Verify timestamps.
	assert.Equal(t, s.ExpiresAt, result.ExpiresAt)
	assert.Equal(t, s.CreatedAt, result.CreatedAt)
	assert.Equal(t, s.UpdatedAt, result.UpdatedAt)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE id").
		WithArgs("nonexistent-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_GetByID_ScanError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE id").
		WithArgs("checkout-err").
		WillReturnError(errors.New("connection reset"))

	result, err := repo.GetByID(context.Background(), "checkout-err")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan checkout session")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_GetByID_NullOptionalFields(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Items as empty array JSON.
	itemsJSON, err := json.Marshal([]domain.CheckoutItem{})
	require.NoError(t, err)

	// Null shipping and billing addresses.
	nullJSON := []byte("null")

	rows := pgxmock.NewRows(sessionColumns()).AddRow(
		"checkout-null", "user-002", domain.StatusInitiated, itemsJSON,
		int64(0), int64(0), int64(0), int64(0),
		"USD", nullJSON, nullJSON,
		(*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil),
		now.Add(30*time.Minute), now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE id").
		WithArgs("checkout-null").
		WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), "checkout-null")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "checkout-null", result.ID)
	assert.Equal(t, "user-002", result.UserID)
	assert.Equal(t, "USD", result.Currency)

	// Optional string fields should be empty strings when DB has NULL.
	assert.Equal(t, "", result.PaymentMethod)
	assert.Equal(t, "", result.PaymentID)
	assert.Equal(t, "", result.OrderID)
	assert.Equal(t, "", result.FailureReason)

	// Addresses should be nil when DB has "null" JSON.
	assert.Nil(t, result.ShippingAddress)
	assert.Nil(t, result.BillingAddress)

	// Items should be empty (not nil) even when DB has [].
	assert.NotNil(t, result.Items)
	assert.Empty(t, result.Items)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCheckoutRepository_Update_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()
	s.Status = domain.StatusPaymentPending

	mock.ExpectExec("UPDATE checkout_sessions").
		WithArgs(
			pgxmock.AnyArg(), // status
			pgxmock.AnyArg(), // items JSON
			pgxmock.AnyArg(), // subtotal_amount
			pgxmock.AnyArg(), // discount_amount
			pgxmock.AnyArg(), // shipping_amount
			pgxmock.AnyArg(), // total_amount
			pgxmock.AnyArg(), // currency
			pgxmock.AnyArg(), // shipping_address JSON
			pgxmock.AnyArg(), // billing_address JSON
			pgxmock.AnyArg(), // payment_method
			pgxmock.AnyArg(), // payment_id
			pgxmock.AnyArg(), // order_id
			pgxmock.AnyArg(), // failure_reason
			pgxmock.AnyArg(), // expires_at
			pgxmock.AnyArg(), // updated_at
			pgxmock.AnyArg(), // id (WHERE clause)
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), s)
	assert.NoError(t, err)

	// Verify UpdatedAt was set to approximately now.
	assert.WithinDuration(t, time.Now().UTC(), s.UpdatedAt, 2*time.Second)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_Update_NotFound(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()
	s.ID = "nonexistent-checkout"

	mock.ExpectExec("UPDATE checkout_sessions").
		WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), s)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_Update_ExecError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()

	mock.ExpectExec("UPDATE checkout_sessions").
		WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).
		WillReturnError(errors.New("write conflict"))

	err := repo.Update(context.Background(), s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update checkout session")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetActiveByUserID
// ---------------------------------------------------------------------------

func TestCheckoutRepository_GetActiveByUserID_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	s := sampleSession()
	s.Status = domain.StatusItemsReserved
	row := sessionRow(t, s)

	rows := pgxmock.NewRows(sessionColumns()).AddRow(row...)

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE user_id").
		WithArgs(s.UserID).
		WillReturnRows(rows)

	result, err := repo.GetActiveByUserID(context.Background(), s.UserID)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, s.ID, result.ID)
	assert.Equal(t, s.UserID, result.UserID)
	assert.Equal(t, domain.StatusItemsReserved, result.Status)
	assert.Equal(t, s.TotalAmount, result.TotalAmount)
	require.Len(t, result.Items, 2)
	require.NotNil(t, result.ShippingAddress)
	assert.Equal(t, "John Doe", result.ShippingAddress.FullName)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_GetActiveByUserID_NotFound(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE user_id").
		WithArgs("user-no-active").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetActiveByUserID(context.Background(), "user-no-active")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListExpired
// ---------------------------------------------------------------------------

func TestCheckoutRepository_ListExpired_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)
	cutoff := now

	// Build two expired sessions.
	s1 := sampleSession()
	s1.ID = "checkout-expired-1"
	s1.UserID = "user-010"
	s1.Status = domain.StatusInitiated
	s1.ExpiresAt = now.Add(-10 * time.Minute)
	s1.CreatedAt = now.Add(-40 * time.Minute)
	s1.UpdatedAt = now.Add(-40 * time.Minute)

	s2 := sampleSession()
	s2.ID = "checkout-expired-2"
	s2.UserID = "user-020"
	s2.Status = domain.StatusPaymentPending
	s2.PaymentMethod = ""
	s2.PaymentID = ""
	s2.OrderID = ""
	s2.ShippingAddress = nil
	s2.BillingAddress = nil
	s2.Items = []domain.CheckoutItem{
		{
			ProductID: "prod-099",
			VariantID: "var-099",
			Name:      "Expired Item",
			SKU:       "EXP-001",
			Price:     3000,
			Quantity:  1,
		},
	}
	s2.SubtotalAmount = 3000
	s2.DiscountAmount = 0
	s2.ShippingAmount = 0
	s2.TotalAmount = 3000
	s2.ExpiresAt = now.Add(-5 * time.Minute)
	s2.CreatedAt = now.Add(-35 * time.Minute)
	s2.UpdatedAt = now.Add(-35 * time.Minute)

	row1 := sessionRow(t, s1)
	row2 := sessionRow(t, s2)

	rows := pgxmock.NewRows(sessionColumns()).
		AddRow(row1...).
		AddRow(row2...)

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE expires_at").
		WithArgs(cutoff).
		WillReturnRows(rows)

	results, err := repo.ListExpired(context.Background(), cutoff)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// First session.
	assert.Equal(t, "checkout-expired-1", results[0].ID)
	assert.Equal(t, "user-010", results[0].UserID)
	assert.Equal(t, domain.StatusInitiated, results[0].Status)
	require.Len(t, results[0].Items, 2)
	require.NotNil(t, results[0].ShippingAddress)
	assert.Equal(t, "credit_card", results[0].PaymentMethod)

	// Second session with null optional fields.
	assert.Equal(t, "checkout-expired-2", results[1].ID)
	assert.Equal(t, "user-020", results[1].UserID)
	assert.Equal(t, domain.StatusPaymentPending, results[1].Status)
	require.Len(t, results[1].Items, 1)
	assert.Equal(t, "Expired Item", results[1].Items[0].Name)
	assert.Nil(t, results[1].ShippingAddress)
	assert.Nil(t, results[1].BillingAddress)
	assert.Equal(t, "", results[1].PaymentMethod)
	assert.Equal(t, "", results[1].PaymentID)
	assert.Equal(t, "", results[1].OrderID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_ListExpired_Empty(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	cutoff := time.Now().UTC()

	rows := pgxmock.NewRows(sessionColumns())

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE expires_at").
		WithArgs(cutoff).
		WillReturnRows(rows)

	results, err := repo.ListExpired(context.Background(), cutoff)
	require.NoError(t, err)
	assert.NotNil(t, results) // should be [] not nil
	assert.Empty(t, results)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckoutRepository_ListExpired_QueryError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.Close()

	cutoff := time.Now().UTC()

	mock.ExpectQuery("SELECT .+ FROM checkout_sessions WHERE expires_at").
		WithArgs(cutoff).
		WillReturnError(errors.New("database timeout"))

	results, err := repo.ListExpired(context.Background(), cutoff)
	assert.Nil(t, results)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list expired sessions")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// nullableString helper
// ---------------------------------------------------------------------------

func TestNullableString(t *testing.T) {
	// Non-empty string returns pointer.
	result := nullableString("hello")
	require.NotNil(t, result)
	assert.Equal(t, "hello", *result)

	// Empty string returns nil.
	result = nullableString("")
	assert.Nil(t, result)
}
