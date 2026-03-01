package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func setupRepo(t *testing.T) (*InventoryRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	repo := NewInventoryRepository(mock)
	return repo, mock
}

var stockColumns = []string{
	"id", "product_id", "variant_id", "warehouse_id",
	"quantity", "reserved", "low_stock_threshold", "updated_at",
}

var reservationColumns = []string{
	"id", "product_id", "variant_id", "quantity",
	"checkout_id", "status", "expires_at", "created_at",
}

func sampleStock() domain.Stock {
	return domain.Stock{
		ID:                "stock-1",
		ProductID:         "prod-1",
		VariantID:         "var-1",
		WarehouseID:       "wh-1",
		Quantity:          100,
		Reserved:          10,
		LowStockThreshold: 5,
		UpdatedAt:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func sampleReservation() domain.StockReservation {
	return domain.StockReservation{
		ID:         "res-1",
		ProductID:  "prod-1",
		VariantID:  "var-1",
		Quantity:   3,
		CheckoutID: "checkout-1",
		Status:     "active",
		ExpiresAt:  time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		CreatedAt:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// ---------------------------------------------------------------------------
// GetByProductVariant
// ---------------------------------------------------------------------------

func TestInventoryRepository_GetByProductVariant_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	s := sampleStock()
	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs(s.ProductID, s.VariantID).
		WillReturnRows(
			pgxmock.NewRows(stockColumns).
				AddRow(s.ID, s.ProductID, s.VariantID, s.WarehouseID,
					s.Quantity, s.Reserved, s.LowStockThreshold, s.UpdatedAt),
		)

	result, err := repo.GetByProductVariant(context.Background(), s.ProductID, s.VariantID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, result.ID)
	assert.Equal(t, s.ProductID, result.ProductID)
	assert.Equal(t, s.VariantID, result.VariantID)
	assert.Equal(t, s.Quantity, result.Quantity)
	assert.Equal(t, s.Reserved, result.Reserved)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_GetByProductVariant_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs("prod-x", "var-x").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByProductVariant(context.Background(), "prod-x", "var-x")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// CreateStock
// ---------------------------------------------------------------------------

func TestInventoryRepository_CreateStock_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	s := sampleStock()
	mock.ExpectQuery("INSERT INTO stock").
		WithArgs(s.ID, s.ProductID, s.VariantID, s.WarehouseID,
			s.Quantity, s.Reserved, s.LowStockThreshold, s.UpdatedAt).
		WillReturnRows(
			pgxmock.NewRows(stockColumns).
				AddRow(s.ID, s.ProductID, s.VariantID, s.WarehouseID,
					s.Quantity, s.Reserved, s.LowStockThreshold, s.UpdatedAt),
		)

	result, err := repo.CreateStock(context.Background(), &s)
	require.NoError(t, err)
	assert.Equal(t, s.ID, result.ID)
	assert.Equal(t, s.ProductID, result.ProductID)
	assert.Equal(t, s.Quantity, result.Quantity)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_CreateStock_Error(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	s := sampleStock()
	mock.ExpectQuery("INSERT INTO stock").
		WithArgs(s.ID, s.ProductID, s.VariantID, s.WarehouseID,
			s.Quantity, s.Reserved, s.LowStockThreshold, s.UpdatedAt).
		WillReturnError(errors.New("db write error"))

	result, err := repo.CreateStock(context.Background(), &s)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create stock")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Upsert
// ---------------------------------------------------------------------------

func TestInventoryRepository_Upsert_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	s := sampleStock()
	mock.ExpectExec("INSERT INTO stock").
		WithArgs(s.ID, s.ProductID, s.VariantID, s.WarehouseID,
			s.Quantity, s.Reserved, s.LowStockThreshold, s.UpdatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Upsert(context.Background(), &s)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// AdjustQuantity
// ---------------------------------------------------------------------------

func TestInventoryRepository_AdjustQuantity_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	refID := "order-123"

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO stock").
		WithArgs(10, "prod-1", "var-1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("stock-id"))
	mock.ExpectExec("INSERT INTO stock_movements").
		WithArgs("prod-1", "var-1", 10, "order", &refID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	err := repo.AdjustQuantity(context.Background(), "prod-1", "var-1", 10, "order", &refID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_AdjustQuantity_WithNilRefID(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO stock").
		WithArgs(5, "prod-2", "var-2").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("stock-id-2"))
	mock.ExpectExec("INSERT INTO stock_movements").
		WithArgs("prod-2", "var-2", 5, "adjustment", (*string)(nil)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	err := repo.AdjustQuantity(context.Background(), "prod-2", "var-2", 5, "adjustment", nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_AdjustQuantity_BeginError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err := repo.AdjustQuantity(context.Background(), "prod-1", "var-1", 10, "order", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_AdjustQuantity_UpsertError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO stock").
		WithArgs(10, "prod-1", "var-1").
		WillReturnError(errors.New("upsert failed"))
	mock.ExpectRollback()

	err := repo.AdjustQuantity(context.Background(), "prod-1", "var-1", 10, "order", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "adjust stock quantity")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_AdjustQuantity_MovementError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO stock").
		WithArgs(10, "prod-1", "var-1").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("stock-id"))
	mock.ExpectExec("INSERT INTO stock_movements").
		WithArgs("prod-1", "var-1", 10, "order", (*string)(nil)).
		WillReturnError(errors.New("movement insert failed"))
	mock.ExpectRollback()

	err := repo.AdjustQuantity(context.Background(), "prod-1", "var-1", 10, "order", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert stock movement")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListLowStock
// ---------------------------------------------------------------------------

func TestInventoryRepository_ListLowStock_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	s := sampleStock()
	cols := append(stockColumns, "total_count")
	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs(10, 0). // perPage=10, offset=0 (page 1)
		WillReturnRows(
			pgxmock.NewRows(cols).
				AddRow(s.ID, s.ProductID, s.VariantID, s.WarehouseID,
					s.Quantity, s.Reserved, s.LowStockThreshold, s.UpdatedAt, 1),
		)

	stocks, total, err := repo.ListLowStock(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Len(t, stocks, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, s.ID, stocks[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_ListLowStock_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	cols := append(stockColumns, "total_count")
	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs(10, 0).
		WillReturnRows(pgxmock.NewRows(cols)) // no rows

	stocks, total, err := repo.ListLowStock(context.Background(), 1, 10)
	require.NoError(t, err)
	assert.Equal(t, []domain.Stock{}, stocks) // empty slice, not nil
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_ListLowStock_Defaults(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	cols := append(stockColumns, "total_count")
	// page<=0 → page=1, perPage<=0 → perPage=20, so offset=0, limit=20
	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs(20, 0).
		WillReturnRows(pgxmock.NewRows(cols))

	stocks, total, err := repo.ListLowStock(context.Background(), 0, 0)
	require.NoError(t, err)
	assert.Equal(t, []domain.Stock{}, stocks)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// BulkCheck
// ---------------------------------------------------------------------------

func TestInventoryRepository_BulkCheck_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	items := []domain.StockCheckItem{
		{ProductID: "p1", VariantID: "v1", Quantity: 2},
		{ProductID: "p2", VariantID: "v2", Quantity: 5},
	}

	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs("p1", "v1", "p2", "v2").
		WillReturnRows(
			pgxmock.NewRows([]string{"product_id", "variant_id", "quantity", "reserved"}).
				AddRow("p1", "v1", 10, 3).
				AddRow("p2", "v2", 8, 1),
		)

	results, err := repo.BulkCheck(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// p1: available = 10-3 = 7, requested 2, InStock true
	assert.Equal(t, "p1", results[0].ProductID)
	assert.Equal(t, 7, results[0].Available)
	assert.Equal(t, 2, results[0].Requested)
	assert.True(t, results[0].InStock)

	// p2: available = 8-1 = 7, requested 5, InStock true
	assert.Equal(t, "p2", results[1].ProductID)
	assert.Equal(t, 7, results[1].Available)
	assert.Equal(t, 5, results[1].Requested)
	assert.True(t, results[1].InStock)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_BulkCheck_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	results, err := repo.BulkCheck(context.Background(), []domain.StockCheckItem{})
	require.NoError(t, err)
	assert.Equal(t, []domain.StockCheckResult{}, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_BulkCheck_MissingItem(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	items := []domain.StockCheckItem{
		{ProductID: "p1", VariantID: "v1", Quantity: 2},
		{ProductID: "p-missing", VariantID: "v-missing", Quantity: 1},
	}

	// Only p1 is returned from DB; p-missing is not found
	mock.ExpectQuery("SELECT .+ FROM stock WHERE").
		WithArgs("p1", "v1", "p-missing", "v-missing").
		WillReturnRows(
			pgxmock.NewRows([]string{"product_id", "variant_id", "quantity", "reserved"}).
				AddRow("p1", "v1", 10, 3),
		)

	results, err := repo.BulkCheck(context.Background(), items)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// p1 exists
	assert.Equal(t, 7, results[0].Available)
	assert.True(t, results[0].InStock)

	// p-missing: Available=0, InStock=false
	assert.Equal(t, "p-missing", results[1].ProductID)
	assert.Equal(t, 0, results[1].Available)
	assert.Equal(t, 1, results[1].Requested)
	assert.False(t, results[1].InStock)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Create (Reservation)
// ---------------------------------------------------------------------------

func TestInventoryRepository_CreateReservation_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r := sampleReservation()
	mock.ExpectExec("INSERT INTO stock_reservations").
		WithArgs(r.ID, r.ProductID, r.VariantID, r.Quantity,
			r.CheckoutID, r.Status, r.ExpiresAt, r.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), &r)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByID (Reservation)
// ---------------------------------------------------------------------------

func TestInventoryRepository_GetReservationByID_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r := sampleReservation()
	mock.ExpectQuery("SELECT .+ FROM stock_reservations WHERE id").
		WithArgs(r.ID).
		WillReturnRows(
			pgxmock.NewRows(reservationColumns).
				AddRow(r.ID, r.ProductID, r.VariantID, r.Quantity,
					r.CheckoutID, r.Status, r.ExpiresAt, r.CreatedAt),
		)

	result, err := repo.GetByID(context.Background(), r.ID)
	require.NoError(t, err)
	assert.Equal(t, r.ID, result.ID)
	assert.Equal(t, r.ProductID, result.ProductID)
	assert.Equal(t, r.CheckoutID, result.CheckoutID)
	assert.Equal(t, r.Status, result.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_GetReservationByID_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM stock_reservations WHERE id").
		WithArgs("missing-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "missing-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByCheckoutID
// ---------------------------------------------------------------------------

func TestInventoryRepository_GetByCheckoutID_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r1 := sampleReservation()
	r2 := domain.StockReservation{
		ID:         "res-2",
		ProductID:  "prod-2",
		VariantID:  "var-2",
		Quantity:   5,
		CheckoutID: "checkout-1",
		Status:     "active",
		ExpiresAt:  time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		CreatedAt:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	mock.ExpectQuery("SELECT .+ FROM stock_reservations WHERE checkout_id").
		WithArgs("checkout-1").
		WillReturnRows(
			pgxmock.NewRows(reservationColumns).
				AddRow(r1.ID, r1.ProductID, r1.VariantID, r1.Quantity,
					r1.CheckoutID, r1.Status, r1.ExpiresAt, r1.CreatedAt).
				AddRow(r2.ID, r2.ProductID, r2.VariantID, r2.Quantity,
					r2.CheckoutID, r2.Status, r2.ExpiresAt, r2.CreatedAt),
		)

	results, err := repo.GetByCheckoutID(context.Background(), "checkout-1")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "res-1", results[0].ID)
	assert.Equal(t, "res-2", results[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_GetByCheckoutID_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM stock_reservations WHERE checkout_id").
		WithArgs("no-such-checkout").
		WillReturnRows(pgxmock.NewRows(reservationColumns))

	results, err := repo.GetByCheckoutID(context.Background(), "no-such-checkout")
	require.NoError(t, err)
	assert.Equal(t, []domain.StockReservation{}, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestInventoryRepository_UpdateStatus_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("UPDATE stock_reservations").
		WithArgs("confirmed", "res-1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateStatus(context.Background(), "res-1", "confirmed")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_UpdateStatus_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("UPDATE stock_reservations").
		WithArgs("confirmed", "missing-id").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.UpdateStatus(context.Background(), "missing-id", "confirmed")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetExpired
// ---------------------------------------------------------------------------

func TestInventoryRepository_GetExpired_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	r := sampleReservation()
	r.Status = "active"
	r.ExpiresAt = time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC) // expired

	mock.ExpectQuery("SELECT .+ FROM stock_reservations WHERE status").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(
			pgxmock.NewRows(reservationColumns).
				AddRow(r.ID, r.ProductID, r.VariantID, r.Quantity,
					r.CheckoutID, r.Status, r.ExpiresAt, r.CreatedAt),
		)

	results, err := repo.GetExpired(context.Background())
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, r.ID, results[0].ID)
	assert.Equal(t, "active", results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInventoryRepository_GetExpired_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM stock_reservations WHERE status").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows(reservationColumns))

	results, err := repo.GetExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []domain.StockReservation{}, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}
