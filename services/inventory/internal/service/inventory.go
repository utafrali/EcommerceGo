package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/event"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/repository"
)

// InventoryService implements the business logic for inventory operations.
type InventoryService struct {
	stockRepo       repository.StockRepository
	reservationRepo repository.ReservationRepository
	pool            *pgxpool.Pool
	producer        *event.Producer
	logger          *slog.Logger
	reservationTTL  int // seconds
}

// NewInventoryService creates a new inventory service.
func NewInventoryService(
	stockRepo repository.StockRepository,
	reservationRepo repository.ReservationRepository,
	pool *pgxpool.Pool,
	producer *event.Producer,
	logger *slog.Logger,
	reservationTTL int,
) *InventoryService {
	return &InventoryService{
		stockRepo:       stockRepo,
		reservationRepo: reservationRepo,
		pool:            pool,
		producer:        producer,
		logger:          logger,
		reservationTTL:  reservationTTL,
	}
}

// InitializeStock creates a new stock record or updates it if one already exists for the
// given product/variant/warehouse combination. This is the entry point for seeding initial
// inventory via the HTTP API.
func (s *InventoryService) InitializeStock(ctx context.Context, stock *domain.Stock) (*domain.Stock, error) {
	if stock.ProductID == "" {
		return nil, apperrors.InvalidInput("product_id is required")
	}
	if stock.VariantID == "" {
		return nil, apperrors.InvalidInput("variant_id is required")
	}
	if stock.Quantity < 0 {
		return nil, apperrors.InvalidInput("quantity must be non-negative")
	}

	// Default warehouse when not specified.
	if stock.WarehouseID == "" {
		stock.WarehouseID = domain.DefaultWarehouseID
	}

	// Assign a new ID and timestamp.
	stock.ID = uuid.New().String()
	stock.Reserved = 0
	stock.UpdatedAt = time.Now().UTC()

	result, err := s.stockRepo.CreateStock(ctx, stock)
	if err != nil {
		return nil, fmt.Errorf("initialize stock: %w", err)
	}

	// Publish inventory.updated event.
	if err := s.producer.PublishInventoryUpdated(ctx, result); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish inventory.updated event after initialization",
			slog.String("product_id", result.ProductID),
			slog.String("variant_id", result.VariantID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "stock initialized",
		slog.String("product_id", result.ProductID),
		slog.String("variant_id", result.VariantID),
		slog.String("warehouse_id", result.WarehouseID),
		slog.Int("quantity", result.Quantity),
	)

	return result, nil
}

// GetStock retrieves the stock level for a specific product variant.
func (s *InventoryService) GetStock(ctx context.Context, productID, variantID string) (*domain.Stock, error) {
	stock, err := s.stockRepo.GetByProductVariant(ctx, productID, variantID)
	if err != nil {
		return nil, fmt.Errorf("get stock: %w", err)
	}
	return stock, nil
}

// AdjustStock modifies the stock quantity by delta and publishes events.
func (s *InventoryService) AdjustStock(ctx context.Context, productID, variantID string, delta int, reason string) (*domain.Stock, error) {
	if productID == "" {
		return nil, apperrors.InvalidInput("product_id is required")
	}
	if variantID == "" {
		return nil, apperrors.InvalidInput("variant_id is required")
	}
	if !domain.IsValidMovementReason(reason) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("invalid movement reason %q", reason))
	}

	if err := s.stockRepo.AdjustQuantity(ctx, productID, variantID, delta, reason, nil); err != nil {
		return nil, fmt.Errorf("adjust stock: %w", err)
	}

	// Fetch the updated stock to return and check thresholds.
	stock, err := s.stockRepo.GetByProductVariant(ctx, productID, variantID)
	if err != nil {
		return nil, fmt.Errorf("get stock after adjustment: %w", err)
	}

	// Publish inventory.updated event.
	if err := s.producer.PublishInventoryUpdated(ctx, stock); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish inventory.updated event",
			slog.String("product_id", productID),
			slog.String("variant_id", variantID),
			slog.String("error", err.Error()),
		)
	}

	// If stock falls below threshold, publish inventory.low_stock event.
	if stock.Available() <= stock.LowStockThreshold {
		if err := s.producer.PublishInventoryLowStock(ctx, stock); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish inventory.low_stock event",
				slog.String("product_id", productID),
				slog.String("variant_id", variantID),
				slog.String("error", err.Error()),
			)
		}
	}

	s.logger.InfoContext(ctx, "stock adjusted",
		slog.String("product_id", productID),
		slog.String("variant_id", variantID),
		slog.Int("delta", delta),
		slog.String("reason", reason),
		slog.Int("new_quantity", stock.Quantity),
		slog.Int("available", stock.Available()),
	)

	return stock, nil
}

// CheckAvailability checks whether the requested quantities are available for multiple items.
func (s *InventoryService) CheckAvailability(ctx context.Context, items []domain.StockCheckItem) ([]domain.StockCheckResult, bool, error) {
	if len(items) == 0 {
		return nil, false, apperrors.InvalidInput("items list cannot be empty")
	}

	results, err := s.stockRepo.BulkCheck(ctx, items)
	if err != nil {
		return nil, false, fmt.Errorf("check availability: %w", err)
	}

	allAvailable := true
	for _, r := range results {
		if !r.InStock {
			allAvailable = false
			break
		}
	}

	return results, allAvailable, nil
}

// ReserveStock atomically reserves stock for a checkout session using row-level locking.
func (s *InventoryService) ReserveStock(ctx context.Context, checkoutID string, items []domain.StockCheckItem, ttlSeconds int) ([]string, error) {
	if checkoutID == "" {
		return nil, apperrors.InvalidInput("checkout_id is required")
	}
	if len(items) == 0 {
		return nil, apperrors.InvalidInput("items list cannot be empty")
	}
	if ttlSeconds <= 0 {
		ttlSeconds = s.reservationTTL
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin reservation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(ttlSeconds) * time.Second)
	reservationIDs := make([]string, 0, len(items))

	for _, item := range items {
		// Lock the stock row with SELECT FOR UPDATE to prevent overselling.
		var stockQuantity, stockReserved int
		lockQuery := `
			SELECT quantity, reserved
			FROM stock
			WHERE product_id = $1 AND variant_id = $2
			FOR UPDATE`

		err := tx.QueryRow(ctx, lockQuery, item.ProductID, item.VariantID).Scan(&stockQuantity, &stockReserved)
		if err != nil {
			return nil, apperrors.InvalidInput(fmt.Sprintf("stock not found for product %s variant %s", item.ProductID, item.VariantID))
		}

		available := stockQuantity - stockReserved
		if available < item.Quantity {
			return nil, apperrors.InvalidInput(fmt.Sprintf(
				"insufficient stock for product %s variant %s: requested %d, available %d",
				item.ProductID, item.VariantID, item.Quantity, available,
			))
		}

		// Increment the reserved count.
		updateQuery := `
			UPDATE stock
			SET reserved = reserved + $1, updated_at = NOW()
			WHERE product_id = $2 AND variant_id = $3`

		_, err = tx.Exec(ctx, updateQuery, item.Quantity, item.ProductID, item.VariantID)
		if err != nil {
			return nil, fmt.Errorf("update reserved count: %w", err)
		}

		// Create the reservation record.
		reservationID := uuid.New().String()
		insertQuery := `
			INSERT INTO stock_reservations (id, product_id, variant_id, quantity, checkout_id, status, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

		_, err = tx.Exec(ctx, insertQuery,
			reservationID,
			item.ProductID,
			item.VariantID,
			item.Quantity,
			checkoutID,
			domain.ReservationStatusActive,
			expiresAt,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("create reservation record: %w", err)
		}

		reservationIDs = append(reservationIDs, reservationID)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit reservation transaction: %w", err)
	}

	// Publish events for each reservation (outside of transaction).
	for i, item := range items {
		reservation := &domain.StockReservation{
			ID:         reservationIDs[i],
			ProductID:  item.ProductID,
			VariantID:  item.VariantID,
			Quantity:   item.Quantity,
			CheckoutID: checkoutID,
		}
		if err := s.producer.PublishInventoryReserved(ctx, reservation); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish inventory.reserved event",
				slog.String("reservation_id", reservationIDs[i]),
				slog.String("error", err.Error()),
			)
		}
	}

	s.logger.InfoContext(ctx, "stock reserved",
		slog.String("checkout_id", checkoutID),
		slog.Int("item_count", len(items)),
		slog.Any("reservation_ids", reservationIDs),
	)

	return reservationIDs, nil
}

// ReleaseReservation releases a reservation, restoring the reserved count.
// It uses SELECT FOR UPDATE inside the transaction to prevent double-release
// from concurrent calls (e.g., overlapping CleanExpiredReservations runs).
func (s *InventoryService) ReleaseReservation(ctx context.Context, reservationID string) error {
	reservation, err := s.reservationRepo.GetByID(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("get reservation for release: %w", err)
	}

	if reservation.Status != domain.ReservationStatusActive {
		return apperrors.InvalidInput(fmt.Sprintf("reservation %s is already %s", reservationID, reservation.Status))
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin release transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Re-check reservation status under row lock to prevent double-release.
	var lockedStatus string
	lockQuery := `
		SELECT status
		FROM stock_reservations
		WHERE id = $1
		FOR UPDATE`

	if err := tx.QueryRow(ctx, lockQuery, reservationID).Scan(&lockedStatus); err != nil {
		return fmt.Errorf("lock reservation for release: %w", err)
	}
	if lockedStatus != domain.ReservationStatusActive {
		// Another concurrent call already released/confirmed this reservation.
		return apperrors.InvalidInput(fmt.Sprintf("reservation %s is already %s", reservationID, lockedStatus))
	}

	// Restore the reserved count.
	updateQuery := `
		UPDATE stock
		SET reserved = GREATEST(reserved - $1, 0), updated_at = NOW()
		WHERE product_id = $2 AND variant_id = $3`

	_, err = tx.Exec(ctx, updateQuery, reservation.Quantity, reservation.ProductID, reservation.VariantID)
	if err != nil {
		return fmt.Errorf("restore reserved count: %w", err)
	}

	// Update reservation status.
	statusQuery := `
		UPDATE stock_reservations
		SET status = $1
		WHERE id = $2`

	_, err = tx.Exec(ctx, statusQuery, domain.ReservationStatusReleased, reservationID)
	if err != nil {
		return fmt.Errorf("update reservation status to released: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit release transaction: %w", err)
	}

	// Publish event.
	if err := s.producer.PublishInventoryReleased(ctx, reservation); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish inventory.released event",
			slog.String("reservation_id", reservationID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "reservation released",
		slog.String("reservation_id", reservationID),
		slog.String("product_id", reservation.ProductID),
		slog.String("variant_id", reservation.VariantID),
	)

	return nil
}

// ConfirmReservation confirms a reservation, deducting from quantity and clearing reserved.
func (s *InventoryService) ConfirmReservation(ctx context.Context, reservationID string) error {
	reservation, err := s.reservationRepo.GetByID(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("get reservation for confirm: %w", err)
	}

	if reservation.Status == domain.ReservationStatusConfirmed {
		return apperrors.InvalidInput(fmt.Sprintf("reservation %s is already confirmed", reservationID))
	}

	if reservation.Status != domain.ReservationStatusActive {
		return apperrors.InvalidInput(fmt.Sprintf("reservation %s cannot be confirmed, status is %s", reservationID, reservation.Status))
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin confirm transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Deduct from quantity and clear reserved.
	updateQuery := `
		UPDATE stock
		SET quantity = quantity - $1, reserved = reserved - $1, updated_at = NOW()
		WHERE product_id = $2 AND variant_id = $3`

	_, err = tx.Exec(ctx, updateQuery, reservation.Quantity, reservation.ProductID, reservation.VariantID)
	if err != nil {
		return fmt.Errorf("deduct stock for confirmation: %w", err)
	}

	// Record the stock movement.
	refID := reservation.ID
	movementQuery := `
		INSERT INTO stock_movements (product_id, variant_id, quantity_change, reason, reference_id)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.Exec(ctx, movementQuery,
		reservation.ProductID,
		reservation.VariantID,
		-reservation.Quantity,
		domain.MovementReasonOrder,
		refID,
	)
	if err != nil {
		return fmt.Errorf("insert stock movement for confirmation: %w", err)
	}

	// Update reservation status.
	statusQuery := `
		UPDATE stock_reservations
		SET status = $1
		WHERE id = $2`

	_, err = tx.Exec(ctx, statusQuery, domain.ReservationStatusConfirmed, reservationID)
	if err != nil {
		return fmt.Errorf("update reservation status to confirmed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit confirm transaction: %w", err)
	}

	s.logger.InfoContext(ctx, "reservation confirmed",
		slog.String("reservation_id", reservationID),
		slog.String("product_id", reservation.ProductID),
		slog.String("variant_id", reservation.VariantID),
		slog.Int("quantity", reservation.Quantity),
	)

	return nil
}

// ConfirmReservationByCheckoutID confirms all active reservations for a checkout session.
func (s *InventoryService) ConfirmReservationByCheckoutID(ctx context.Context, checkoutID string) error {
	reservations, err := s.reservationRepo.GetByCheckoutID(ctx, checkoutID)
	if err != nil {
		return fmt.Errorf("get reservations by checkout: %w", err)
	}

	for i := range reservations {
		if reservations[i].Status == domain.ReservationStatusActive {
			if err := s.ConfirmReservation(ctx, reservations[i].ID); err != nil {
				return fmt.Errorf("confirm reservation %s: %w", reservations[i].ID, err)
			}
		}
	}

	return nil
}

// ReleaseReservationByCheckoutID releases all active reservations for a checkout session.
func (s *InventoryService) ReleaseReservationByCheckoutID(ctx context.Context, checkoutID string) error {
	reservations, err := s.reservationRepo.GetByCheckoutID(ctx, checkoutID)
	if err != nil {
		return fmt.Errorf("get reservations by checkout: %w", err)
	}

	for i := range reservations {
		if reservations[i].Status == domain.ReservationStatusActive {
			if err := s.ReleaseReservation(ctx, reservations[i].ID); err != nil {
				return fmt.Errorf("release reservation %s: %w", reservations[i].ID, err)
			}
		}
	}

	return nil
}

// ListLowStock returns stock items where available quantity is below the threshold.
func (s *InventoryService) ListLowStock(ctx context.Context, page, perPage int) ([]domain.Stock, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	stocks, total, err := s.stockRepo.ListLowStock(ctx, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list low stock: %w", err)
	}

	return stocks, total, nil
}

// CleanExpiredReservations finds and releases all expired active reservations.
func (s *InventoryService) CleanExpiredReservations(ctx context.Context) (int, error) {
	expired, err := s.reservationRepo.GetExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("get expired reservations: %w", err)
	}

	released := 0
	for i := range expired {
		if err := s.ReleaseReservation(ctx, expired[i].ID); err != nil {
			s.logger.ErrorContext(ctx, "failed to release expired reservation",
				slog.String("reservation_id", expired[i].ID),
				slog.String("error", err.Error()),
			)
			continue
		}

		// Mark as expired instead of released for tracking.
		if err := s.reservationRepo.UpdateStatus(ctx, expired[i].ID, domain.ReservationStatusExpired); err != nil {
			s.logger.ErrorContext(ctx, "failed to mark reservation as expired",
				slog.String("reservation_id", expired[i].ID),
				slog.String("error", err.Error()),
			)
		}

		released++
	}

	if released > 0 {
		s.logger.InfoContext(ctx, "cleaned expired reservations",
			slog.Int("released_count", released),
			slog.Int("total_expired", len(expired)),
		)
	}

	return released, nil
}
