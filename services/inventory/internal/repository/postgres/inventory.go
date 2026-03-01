package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
)

// InventoryRepository implements both StockRepository and ReservationRepository using PostgreSQL.
type InventoryRepository struct {
	pool database.DBTX
}

// NewInventoryRepository creates a new PostgreSQL-backed inventory repository.
func NewInventoryRepository(pool database.DBTX) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

// ---------------------------------------------------------------------------
// StockRepository implementation
// ---------------------------------------------------------------------------

// GetByProductVariant retrieves stock for a specific product variant.
func (r *InventoryRepository) GetByProductVariant(ctx context.Context, productID, variantID string) (*domain.Stock, error) {
	query := `
		SELECT id, product_id, variant_id, warehouse_id, quantity, reserved, low_stock_threshold, updated_at
		FROM stock
		WHERE product_id = $1 AND variant_id = $2`

	var s domain.Stock
	err := r.pool.QueryRow(ctx, query, productID, variantID).Scan(
		&s.ID,
		&s.ProductID,
		&s.VariantID,
		&s.WarehouseID,
		&s.Quantity,
		&s.Reserved,
		&s.LowStockThreshold,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("get stock by product variant: %w", err)
	}

	return &s, nil
}

// CreateStock inserts a new stock record or updates it if it already exists (idempotent).
// It returns the resulting stock row after the upsert.
func (r *InventoryRepository) CreateStock(ctx context.Context, stock *domain.Stock) (*domain.Stock, error) {
	query := `
		INSERT INTO stock (id, product_id, variant_id, warehouse_id, quantity, reserved, low_stock_threshold, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (product_id, variant_id, warehouse_id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			low_stock_threshold = EXCLUDED.low_stock_threshold,
			updated_at = EXCLUDED.updated_at
		RETURNING id, product_id, variant_id, warehouse_id, quantity, reserved, low_stock_threshold, updated_at`

	var result domain.Stock
	err := r.pool.QueryRow(ctx, query,
		stock.ID,
		stock.ProductID,
		stock.VariantID,
		stock.WarehouseID,
		stock.Quantity,
		stock.Reserved,
		stock.LowStockThreshold,
		stock.UpdatedAt,
	).Scan(
		&result.ID,
		&result.ProductID,
		&result.VariantID,
		&result.WarehouseID,
		&result.Quantity,
		&result.Reserved,
		&result.LowStockThreshold,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create stock: %w", err)
	}

	return &result, nil
}

// Upsert creates or updates stock for a product variant.
func (r *InventoryRepository) Upsert(ctx context.Context, stock *domain.Stock) error {
	query := `
		INSERT INTO stock (id, product_id, variant_id, warehouse_id, quantity, reserved, low_stock_threshold, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (product_id, variant_id, warehouse_id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			reserved = EXCLUDED.reserved,
			low_stock_threshold = EXCLUDED.low_stock_threshold,
			updated_at = EXCLUDED.updated_at`

	_, err := r.pool.Exec(ctx, query,
		stock.ID,
		stock.ProductID,
		stock.VariantID,
		stock.WarehouseID,
		stock.Quantity,
		stock.Reserved,
		stock.LowStockThreshold,
		stock.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert stock: %w", err)
	}

	return nil
}

// AdjustQuantity atomically adjusts the stock quantity by delta and records a movement.
// If the stock record does not exist, it is created with the delta as the initial quantity
// (using INSERT ... ON CONFLICT ... UPDATE, i.e. UPSERT logic).
func (r *InventoryRepository) AdjustQuantity(ctx context.Context, productID, variantID string, delta int, reason string, refID *string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Upsert: insert if missing, otherwise adjust existing quantity.
	upsertQuery := `
		INSERT INTO stock (product_id, variant_id, warehouse_id, quantity, reserved, low_stock_threshold, updated_at)
		VALUES ($2, $3, '00000000-0000-0000-0000-000000000001', GREATEST($1, 0), 0, 10, NOW())
		ON CONFLICT (product_id, variant_id, warehouse_id) DO UPDATE SET
			quantity = stock.quantity + $1,
			updated_at = NOW()
		RETURNING id`

	var stockID string
	err = tx.QueryRow(ctx, upsertQuery, delta, productID, variantID).Scan(&stockID)
	if err != nil {
		return fmt.Errorf("adjust stock quantity: %w", err)
	}

	// Record the stock movement.
	movementQuery := `
		INSERT INTO stock_movements (product_id, variant_id, quantity_change, reason, reference_id)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.Exec(ctx, movementQuery, productID, variantID, delta, reason, refID)
	if err != nil {
		return fmt.Errorf("insert stock movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ListLowStock returns stock items where available quantity is below the threshold.
func (r *InventoryRepository) ListLowStock(ctx context.Context, page, perPage int) ([]domain.Stock, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	query := `
		SELECT id, product_id, variant_id, warehouse_id, quantity, reserved, low_stock_threshold, updated_at,
			   count(*) OVER() AS total_count
		FROM stock
		WHERE (quantity - reserved) <= low_stock_threshold
		ORDER BY (quantity - reserved) ASC, updated_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list low stock: %w", err)
	}
	defer rows.Close()

	var (
		stocks     []domain.Stock
		totalCount int
	)

	for rows.Next() {
		var s domain.Stock
		if err := rows.Scan(
			&s.ID,
			&s.ProductID,
			&s.VariantID,
			&s.WarehouseID,
			&s.Quantity,
			&s.Reserved,
			&s.LowStockThreshold,
			&s.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan low stock row: %w", err)
		}
		stocks = append(stocks, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate low stock rows: %w", err)
	}

	if stocks == nil {
		stocks = []domain.Stock{}
	}

	return stocks, totalCount, nil
}

// BulkCheck checks availability for multiple items at once using a single batch query.
func (r *InventoryRepository) BulkCheck(ctx context.Context, items []domain.StockCheckItem) ([]domain.StockCheckResult, error) {
	if len(items) == 0 {
		return []domain.StockCheckResult{}, nil
	}

	// Build a single query using VALUES list for all product_id/variant_id pairs.
	// Query: SELECT product_id, variant_id, quantity, reserved FROM stock
	//        WHERE (product_id, variant_id) IN (VALUES ($1,$2), ($3,$4), ...)
	args := make([]interface{}, 0, len(items)*2)
	valueClauses := make([]string, 0, len(items))
	for i, item := range items {
		p1 := strconv.Itoa(i*2 + 1)
		p2 := strconv.Itoa(i*2 + 2)
		valueClauses = append(valueClauses, "($"+p1+",$"+p2+")")
		args = append(args, item.ProductID, item.VariantID)
	}

	query := `
		SELECT product_id, variant_id, quantity, reserved
		FROM stock
		WHERE (product_id, variant_id) IN (VALUES ` + strings.Join(valueClauses, ", ") + `)`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("bulk check stock: %w", err)
	}
	defer rows.Close()

	// Collect DB results into a map keyed by "product_id|variant_id".
	type stockRow struct {
		quantity int
		reserved int
	}
	stockMap := make(map[string]stockRow, len(items))

	for rows.Next() {
		var productID, variantID string
		var quantity, reserved int
		if err := rows.Scan(&productID, &variantID, &quantity, &reserved); err != nil {
			return nil, fmt.Errorf("scan bulk check row: %w", err)
		}
		stockMap[productID+"|"+variantID] = stockRow{quantity: quantity, reserved: reserved}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bulk check rows: %w", err)
	}

	// Build results in the same order as the input items.
	results := make([]domain.StockCheckResult, 0, len(items))
	for _, item := range items {
		key := item.ProductID + "|" + item.VariantID
		if row, ok := stockMap[key]; ok {
			available := row.quantity - row.reserved
			results = append(results, domain.StockCheckResult{
				ProductID: item.ProductID,
				VariantID: item.VariantID,
				Requested: item.Quantity,
				Available: available,
				InStock:   available >= item.Quantity,
			})
		} else {
			results = append(results, domain.StockCheckResult{
				ProductID: item.ProductID,
				VariantID: item.VariantID,
				Requested: item.Quantity,
				Available: 0,
				InStock:   false,
			})
		}
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// ReservationRepository implementation
// ---------------------------------------------------------------------------

// Create inserts a new stock reservation.
func (r *InventoryRepository) Create(ctx context.Context, reservation *domain.StockReservation) error {
	query := `
		INSERT INTO stock_reservations (id, product_id, variant_id, quantity, checkout_id, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		reservation.ID,
		reservation.ProductID,
		reservation.VariantID,
		reservation.Quantity,
		reservation.CheckoutID,
		reservation.Status,
		reservation.ExpiresAt,
		reservation.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create reservation: %w", err)
	}

	return nil
}

// GetByID retrieves a reservation by its unique identifier.
func (r *InventoryRepository) GetByID(ctx context.Context, id string) (*domain.StockReservation, error) {
	query := `
		SELECT id, product_id, variant_id, quantity, checkout_id, status, expires_at, created_at
		FROM stock_reservations
		WHERE id = $1`

	var res domain.StockReservation
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&res.ID,
		&res.ProductID,
		&res.VariantID,
		&res.Quantity,
		&res.CheckoutID,
		&res.Status,
		&res.ExpiresAt,
		&res.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("get reservation by id: %w", err)
	}

	return &res, nil
}

// GetByCheckoutID retrieves all reservations for a checkout session.
func (r *InventoryRepository) GetByCheckoutID(ctx context.Context, checkoutID string) ([]domain.StockReservation, error) {
	query := `
		SELECT id, product_id, variant_id, quantity, checkout_id, status, expires_at, created_at
		FROM stock_reservations
		WHERE checkout_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, checkoutID)
	if err != nil {
		return nil, fmt.Errorf("get reservations by checkout id: %w", err)
	}
	defer rows.Close()

	var reservations []domain.StockReservation
	for rows.Next() {
		var res domain.StockReservation
		if err := rows.Scan(
			&res.ID,
			&res.ProductID,
			&res.VariantID,
			&res.Quantity,
			&res.CheckoutID,
			&res.Status,
			&res.ExpiresAt,
			&res.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reservation row: %w", err)
		}
		reservations = append(reservations, res)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reservation rows: %w", err)
	}

	if reservations == nil {
		reservations = []domain.StockReservation{}
	}

	return reservations, nil
}

// UpdateStatus updates the status of a reservation.
func (r *InventoryRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `
		UPDATE stock_reservations
		SET status = $1
		WHERE id = $2`

	ct, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update reservation status: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("reservation", id)
	}

	return nil
}

// GetExpired returns all active reservations that have passed their expiration time.
func (r *InventoryRepository) GetExpired(ctx context.Context) ([]domain.StockReservation, error) {
	query := `
		SELECT id, product_id, variant_id, quantity, checkout_id, status, expires_at, created_at
		FROM stock_reservations
		WHERE status = 'active' AND expires_at < $1
		ORDER BY expires_at ASC`

	rows, err := r.pool.Query(ctx, query, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("get expired reservations: %w", err)
	}
	defer rows.Close()

	var reservations []domain.StockReservation
	for rows.Next() {
		var res domain.StockReservation
		if err := rows.Scan(
			&res.ID,
			&res.ProductID,
			&res.VariantID,
			&res.Quantity,
			&res.CheckoutID,
			&res.Status,
			&res.ExpiresAt,
			&res.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan expired reservation row: %w", err)
		}
		reservations = append(reservations, res)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired reservation rows: %w", err)
	}

	if reservations == nil {
		reservations = []domain.StockReservation{}
	}

	return reservations, nil
}

// ---------------------------------------------------------------------------
// Transactional helpers (used by service layer)
// ---------------------------------------------------------------------------

// Pool returns the underlying connection pool for transactional operations in the service layer.
func (r *InventoryRepository) Pool() database.DBTX {
	return r.pool
}
