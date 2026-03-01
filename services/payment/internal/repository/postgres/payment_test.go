package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/payment/internal/domain"
)

// helper to build a sample payment for tests.
func samplePayment() *domain.Payment {
	return &domain.Payment{
		ID:             "pay-001",
		CheckoutID:     "chk-001",
		OrderID:        "ord-001",
		UserID:         "usr-001",
		Amount:         9999,
		Currency:       "USD",
		Status:         domain.PaymentStatusPending,
		Method:         domain.PaymentMethodCreditCard,
		ProviderName:   "stripe",
		ProviderPayID:  "pi_abc123",
		FailureReason:  "",
		IdempotencyKey: "idem-key-001",
		Metadata:       map[string]any{"source": "web", "retry": float64(0)},
		CreatedAt:      time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

// helper to build a sample refund for tests.
func sampleRefund() *domain.Refund {
	return &domain.Refund{
		ID:            "ref-001",
		PaymentID:     "pay-001",
		Amount:        5000,
		Currency:      "USD",
		Status:        domain.RefundStatusPending,
		Reason:        "customer request",
		ProviderRefID: "re_xyz789",
		CreatedAt:     time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC),
	}
}

var paymentColumns = []string{
	"id", "checkout_id", "order_id", "user_id", "amount", "currency",
	"status", "method", "provider_name", "provider_payment_id",
	"failure_reason", "idempotency_key", "metadata", "created_at", "updated_at",
}

var refundColumns = []string{
	"id", "payment_id", "amount", "currency", "status", "reason",
	"provider_refund_id", "created_at", "updated_at",
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestPaymentRepository_Create(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO payments").
		WithArgs(
			p.ID, p.CheckoutID, p.OrderID, p.UserID,
			p.Amount, p.Currency, p.Status, p.Method,
			p.ProviderName, p.ProviderPayID, p.FailureReason,
			p.IdempotencyKey, metadataJSON, p.CreatedAt, p.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), p)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_Create_ExecError(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("INSERT INTO payments").
		WithArgs(
			p.ID, p.CheckoutID, p.OrderID, p.UserID,
			p.Amount, p.Currency, p.Status, p.Method,
			p.ProviderName, p.ProviderPayID, p.FailureReason,
			p.IdempotencyKey, metadataJSON, p.CreatedAt, p.UpdatedAt,
		).
		WillReturnError(errors.New("connection refused"))

	err = repo.Create(context.Background(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert payment")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestPaymentRepository_GetByID(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT .+ FROM payments").
		WithArgs(p.ID).
		WillReturnRows(
			pgxmock.NewRows(paymentColumns).
				AddRow(
					p.ID, p.CheckoutID, p.OrderID, p.UserID,
					p.Amount, p.Currency, p.Status, p.Method,
					p.ProviderName, p.ProviderPayID, p.FailureReason,
					p.IdempotencyKey, metadataJSON, p.CreatedAt, p.UpdatedAt,
				),
		)

	result, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, p.CheckoutID, result.CheckoutID)
	assert.Equal(t, p.Amount, result.Amount)
	assert.Equal(t, p.Currency, result.Currency)
	assert.Equal(t, p.Status, result.Status)
	assert.Equal(t, p.Method, result.Method)
	assert.Equal(t, p.ProviderName, result.ProviderName)
	assert.Equal(t, "web", result.Metadata["source"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_GetByID_NotFound(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM payments").
		WithArgs("nonexistent-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── GetByCheckoutID ─────────────────────────────────────────────────────────

func TestPaymentRepository_GetByCheckoutID(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT .+ FROM payments").
		WithArgs(p.CheckoutID).
		WillReturnRows(
			pgxmock.NewRows(paymentColumns).
				AddRow(
					p.ID, p.CheckoutID, p.OrderID, p.UserID,
					p.Amount, p.Currency, p.Status, p.Method,
					p.ProviderName, p.ProviderPayID, p.FailureReason,
					p.IdempotencyKey, metadataJSON, p.CreatedAt, p.UpdatedAt,
				),
		)

	result, err := repo.GetByCheckoutID(context.Background(), p.CheckoutID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, p.CheckoutID, result.CheckoutID)
	assert.Equal(t, p.OrderID, result.OrderID)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── GetByIdempotencyKey ─────────────────────────────────────────────────────

func TestPaymentRepository_GetByIdempotencyKey(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectQuery("SELECT .+ FROM payments").
		WithArgs(p.IdempotencyKey).
		WillReturnRows(
			pgxmock.NewRows(paymentColumns).
				AddRow(
					p.ID, p.CheckoutID, p.OrderID, p.UserID,
					p.Amount, p.Currency, p.Status, p.Method,
					p.ProviderName, p.ProviderPayID, p.FailureReason,
					p.IdempotencyKey, metadataJSON, p.CreatedAt, p.UpdatedAt,
				),
		)

	result, err := repo.GetByIdempotencyKey(context.Background(), p.IdempotencyKey)
	require.NoError(t, err)
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, p.IdempotencyKey, result.IdempotencyKey)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_GetByIdempotencyKey_EmptyKey(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)

	result, err := repo.GetByIdempotencyKey(context.Background(), "")
	assert.Nil(t, result)
	assert.Error(t, err)
	// The method returns apperrors.NotFound("payment", "") which wraps ErrNotFound.
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	// No DB query should have been made for empty key.
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestPaymentRepository_Update(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()
	p.Status = domain.PaymentStatusSucceeded

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE payments").
		WithArgs(
			p.CheckoutID, p.OrderID, p.UserID, p.Amount, p.Currency,
			p.Status, p.Method, p.ProviderName, p.ProviderPayID,
			p.FailureReason, metadataJSON,
			pgxmock.AnyArg(), // UpdatedAt is set at call time
			p.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), p)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_Update_NotFound(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	p := samplePayment()
	p.ID = "nonexistent-pay-id"

	metadataJSON, err := json.Marshal(p.Metadata)
	require.NoError(t, err)

	mock.ExpectExec("UPDATE payments").
		WithArgs(
			p.CheckoutID, p.OrderID, p.UserID, p.Amount, p.Currency,
			p.Status, p.Method, p.ProviderName, p.ProviderPayID,
			p.FailureReason, metadataJSON,
			pgxmock.AnyArg(), // UpdatedAt
			p.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = repo.Update(context.Background(), p)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── ListByUserID ────────────────────────────────────────────────────────────

func TestPaymentRepository_ListByUserID(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)

	meta1 := map[string]any{"source": "web"}
	meta1JSON, err := json.Marshal(meta1)
	require.NoError(t, err)

	meta2 := map[string]any{"source": "mobile"}
	meta2JSON, err := json.Marshal(meta2)
	require.NoError(t, err)

	now := time.Now().UTC()
	listColumns := append(paymentColumns, "total_count")

	mock.ExpectQuery("SELECT .+ FROM payments").
		WithArgs("usr-001", 10, 0).
		WillReturnRows(
			pgxmock.NewRows(listColumns).
				AddRow(
					"pay-001", "chk-001", "ord-001", "usr-001",
					int64(9999), "USD", "pending", "credit_card",
					"stripe", "pi_1", "", "idem-1",
					meta1JSON, now, now,
					2, // total_count
				).
				AddRow(
					"pay-002", "chk-002", "ord-002", "usr-001",
					int64(5000), "EUR", "succeeded", "debit_card",
					"adyen", "pi_2", "", "idem-2",
					meta2JSON, now, now,
					2, // total_count
				),
		)

	payments, total, err := repo.ListByUserID(context.Background(), "usr-001", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, payments, 2)

	assert.Equal(t, "pay-001", payments[0].ID)
	assert.Equal(t, int64(9999), payments[0].Amount)
	assert.Equal(t, "web", payments[0].Metadata["source"])

	assert.Equal(t, "pay-002", payments[1].ID)
	assert.Equal(t, int64(5000), payments[1].Amount)
	assert.Equal(t, "mobile", payments[1].Metadata["source"])

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_ListByUserID_Empty(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)

	listColumns := append(paymentColumns, "total_count")

	mock.ExpectQuery("SELECT .+ FROM payments").
		WithArgs("usr-999", 20, 0).
		WillReturnRows(pgxmock.NewRows(listColumns))

	payments, total, err := repo.ListByUserID(context.Background(), "usr-999", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NotNil(t, payments)
	assert.Empty(t, payments)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── CreateRefund ────────────────────────────────────────────────────────────

func TestPaymentRepository_CreateRefund(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	ref := sampleRefund()

	mock.ExpectExec("INSERT INTO refunds").
		WithArgs(
			ref.ID, ref.PaymentID, ref.Amount, ref.Currency,
			ref.Status, ref.Reason, ref.ProviderRefID,
			ref.CreatedAt, ref.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.CreateRefund(context.Background(), ref)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── GetRefundByID ───────────────────────────────────────────────────────────

func TestPaymentRepository_GetRefundByID(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	ref := sampleRefund()

	mock.ExpectQuery("SELECT .+ FROM refunds").
		WithArgs(ref.ID).
		WillReturnRows(
			pgxmock.NewRows(refundColumns).
				AddRow(
					ref.ID, ref.PaymentID, ref.Amount, ref.Currency,
					ref.Status, ref.Reason, ref.ProviderRefID,
					ref.CreatedAt, ref.UpdatedAt,
				),
		)

	result, err := repo.GetRefundByID(context.Background(), ref.ID)
	require.NoError(t, err)
	assert.Equal(t, ref.ID, result.ID)
	assert.Equal(t, ref.PaymentID, result.PaymentID)
	assert.Equal(t, ref.Amount, result.Amount)
	assert.Equal(t, ref.Currency, result.Currency)
	assert.Equal(t, ref.Status, result.Status)
	assert.Equal(t, ref.Reason, result.Reason)
	assert.Equal(t, ref.ProviderRefID, result.ProviderRefID)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_GetRefundByID_NotFound(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM refunds").
		WithArgs("nonexistent-ref").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetRefundByID(context.Background(), "nonexistent-ref")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── ListRefundsByPaymentID ──────────────────────────────────────────────────

func TestPaymentRepository_ListRefundsByPaymentID(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	now := time.Now().UTC()

	mock.ExpectQuery("SELECT .+ FROM refunds").
		WithArgs("pay-001").
		WillReturnRows(
			pgxmock.NewRows(refundColumns).
				AddRow("ref-001", "pay-001", int64(3000), "USD", "succeeded", "partial refund", "re_1", now, now).
				AddRow("ref-002", "pay-001", int64(2000), "USD", "pending", "remaining", "re_2", now, now),
		)

	refunds, err := repo.ListRefundsByPaymentID(context.Background(), "pay-001")
	require.NoError(t, err)
	assert.Len(t, refunds, 2)

	assert.Equal(t, "ref-001", refunds[0].ID)
	assert.Equal(t, int64(3000), refunds[0].Amount)
	assert.Equal(t, "succeeded", refunds[0].Status)

	assert.Equal(t, "ref-002", refunds[1].ID)
	assert.Equal(t, int64(2000), refunds[1].Amount)
	assert.Equal(t, "pending", refunds[1].Status)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_ListRefundsByPaymentID_Empty(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM refunds").
		WithArgs("pay-no-refunds").
		WillReturnRows(pgxmock.NewRows(refundColumns))

	refunds, err := repo.ListRefundsByPaymentID(context.Background(), "pay-no-refunds")
	require.NoError(t, err)
	assert.NotNil(t, refunds)
	assert.Empty(t, refunds)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

// ─── UpdateRefund ────────────────────────────────────────────────────────────

func TestPaymentRepository_UpdateRefund(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	ref := sampleRefund()
	ref.Status = domain.RefundStatusSucceeded

	mock.ExpectExec("UPDATE refunds").
		WithArgs(
			ref.Amount, ref.Currency, ref.Status, ref.Reason,
			ref.ProviderRefID,
			pgxmock.AnyArg(), // UpdatedAt set at call time
			ref.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateRefund(context.Background(), ref)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPaymentRepository_UpdateRefund_NotFound(t *testing.T) {
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewPaymentRepository(mock)
	ref := sampleRefund()
	ref.ID = "nonexistent-ref-id"

	mock.ExpectExec("UPDATE refunds").
		WithArgs(
			ref.Amount, ref.Currency, ref.Status, ref.Reason,
			ref.ProviderRefID,
			pgxmock.AnyArg(), // UpdatedAt
			ref.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = repo.UpdateRefund(context.Background(), ref)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
