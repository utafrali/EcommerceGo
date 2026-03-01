package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
	"github.com/utafrali/EcommerceGo/services/order/internal/repository"
)

// OrderRepository implements repository.OrderRepository using PostgreSQL.
type OrderRepository struct {
	pool database.DBTX
}

// NewOrderRepository creates a new PostgreSQL-backed order repository.
func NewOrderRepository(pool database.DBTX) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create inserts a new order and its items atomically within a transaction.
func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var shippingJSON, billingJSON []byte

	if o.ShippingAddress != nil {
		shippingJSON, err = json.Marshal(o.ShippingAddress)
		if err != nil {
			return fmt.Errorf("marshal shipping address: %w", err)
		}
	}

	if o.BillingAddress != nil {
		billingJSON, err = json.Marshal(o.BillingAddress)
		if err != nil {
			return fmt.Errorf("marshal billing address: %w", err)
		}
	}

	orderQuery := `
		INSERT INTO orders (id, user_id, status, subtotal_amount, discount_amount, shipping_amount, total_amount, currency, shipping_address, billing_address, notes, canceled_reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err = tx.Exec(ctx, orderQuery,
		o.ID,
		o.UserID,
		o.Status,
		o.SubtotalAmount,
		o.DiscountAmount,
		o.ShippingAmount,
		o.TotalAmount,
		o.Currency,
		shippingJSON,
		billingJSON,
		o.Notes,
		o.CanceledReason,
		o.CreatedAt,
		o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	// Insert order items.
	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, variant_id, name, sku, price, quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	for _, item := range o.Items {
		_, err = tx.Exec(ctx, itemQuery,
			item.ID,
			item.OrderID,
			item.ProductID,
			item.VariantID,
			item.Name,
			item.SKU,
			item.Price,
			item.Quantity,
		)
		if err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves an order by its ID, eagerly loading its items.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	// Optimized query: fetch order and items in a single query using LEFT JOIN + JSONB_AGG.
	// This eliminates the N+1 query problem (was: 1 query for order + 1 query for items).
	orderQuery := `
		SELECT
			o.id, o.user_id, o.status, o.subtotal_amount, o.discount_amount,
			o.shipping_amount, o.total_amount, o.currency, o.shipping_address,
			o.billing_address, o.notes, o.canceled_reason, o.created_at, o.updated_at,
			COALESCE(
				JSONB_AGG(
					JSONB_BUILD_OBJECT(
						'id', oi.id,
						'product_id', oi.product_id,
						'variant_id', oi.variant_id,
						'name', oi.name,
						'sku', oi.sku,
						'price', oi.price,
						'quantity', oi.quantity,
						'subtotal', oi.price * oi.quantity
					) ORDER BY oi.created_at
				) FILTER (WHERE oi.id IS NOT NULL),
				'[]'::jsonb
			) AS items
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.id = $1
		GROUP BY o.id, o.user_id, o.status, o.subtotal_amount, o.discount_amount,
			o.shipping_amount, o.total_amount, o.currency, o.shipping_address,
			o.billing_address, o.notes, o.canceled_reason, o.created_at, o.updated_at`

	var (
		o            domain.Order
		shippingJSON []byte
		billingJSON  []byte
		itemsJSON    []byte
	)

	err := r.pool.QueryRow(ctx, orderQuery, id).Scan(
		&o.ID,
		&o.UserID,
		&o.Status,
		&o.SubtotalAmount,
		&o.DiscountAmount,
		&o.ShippingAmount,
		&o.TotalAmount,
		&o.Currency,
		&shippingJSON,
		&billingJSON,
		&o.Notes,
		&o.CanceledReason,
		&o.CreatedAt,
		&o.UpdatedAt,
		&itemsJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan order: %w", err)
	}

	if len(shippingJSON) > 0 && string(shippingJSON) != "null" {
		var addr domain.Address
		if err := json.Unmarshal(shippingJSON, &addr); err != nil {
			return nil, fmt.Errorf("unmarshal shipping address: %w", err)
		}
		o.ShippingAddress = &addr
	}

	if len(billingJSON) > 0 && string(billingJSON) != "null" {
		var addr domain.Address
		if err := json.Unmarshal(billingJSON, &addr); err != nil {
			return nil, fmt.Errorf("unmarshal billing address: %w", err)
		}
		o.BillingAddress = &addr
	}

	// Unmarshal items from JSONB_AGG result.
	if len(itemsJSON) > 0 && string(itemsJSON) != "null" && string(itemsJSON) != "[]" {
		if err := json.Unmarshal(itemsJSON, &o.Items); err != nil {
			return nil, fmt.Errorf("unmarshal order items: %w", err)
		}
	} else {
		o.Items = []domain.OrderItem{}
	}

	return &o, nil
}

// List returns orders matching the given filter with the total count.
func (r *OrderRepository) List(ctx context.Context, filter repository.OrderFilter) ([]domain.Order, int, error) {
	var (
		conditions []string
		args       []any
		argIndex   int = 1
	)

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Use count(*) OVER() for total count in a single query.
	query := fmt.Sprintf(`
		SELECT id, user_id, status, subtotal_amount, discount_amount, shipping_amount, total_amount, currency, shipping_address, billing_address, notes, canceled_reason, created_at, updated_at,
			   count(*) OVER() AS total_count
		FROM orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1,
	)

	limit := filter.PerPage
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var totalCount int
	orders := make([]domain.Order, 0)

	for rows.Next() {
		var (
			o            domain.Order
			shippingJSON []byte
			billingJSON  []byte
		)

		if err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.SubtotalAmount,
			&o.DiscountAmount,
			&o.ShippingAmount,
			&o.TotalAmount,
			&o.Currency,
			&shippingJSON,
			&billingJSON,
			&o.Notes,
			&o.CanceledReason,
			&o.CreatedAt,
			&o.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan order row: %w", err)
		}

		if len(shippingJSON) > 0 && string(shippingJSON) != "null" {
			var addr domain.Address
			if err := json.Unmarshal(shippingJSON, &addr); err != nil {
				return nil, 0, fmt.Errorf("unmarshal shipping address: %w", err)
			}
			o.ShippingAddress = &addr
		}

		if len(billingJSON) > 0 && string(billingJSON) != "null" {
			var addr domain.Address
			if err := json.Unmarshal(billingJSON, &addr); err != nil {
				return nil, 0, fmt.Errorf("unmarshal billing address: %w", err)
			}
			o.BillingAddress = &addr
		}

		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate order rows: %w", err)
	}

	// Batch-load items for all orders in a single query to avoid N+1.
	if len(orders) > 0 {
		orderIDs := make([]string, len(orders))
		for i := range orders {
			orderIDs[i] = orders[i].ID
		}

		itemsQuery := `
			SELECT id, order_id, product_id, variant_id, name, sku, price, quantity, price * quantity AS subtotal
			FROM order_items
			WHERE order_id = ANY($1)
			ORDER BY id`

		itemRows, err := r.pool.Query(ctx, itemsQuery, orderIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("batch load order items: %w", err)
		}
		defer itemRows.Close()

		// Group items by order_id.
		itemsByOrderID := make(map[string][]domain.OrderItem, len(orders))
		for itemRows.Next() {
			var item domain.OrderItem
			if err := itemRows.Scan(
				&item.ID,
				&item.OrderID,
				&item.ProductID,
				&item.VariantID,
				&item.Name,
				&item.SKU,
				&item.Price,
				&item.Quantity,
				&item.Subtotal,
			); err != nil {
				return nil, 0, fmt.Errorf("scan order item: %w", err)
			}
			itemsByOrderID[item.OrderID] = append(itemsByOrderID[item.OrderID], item)
		}
		if err := itemRows.Err(); err != nil {
			return nil, 0, fmt.Errorf("iterate batch order item rows: %w", err)
		}

		// Assign items to their respective orders.
		for i := range orders {
			if items, ok := itemsByOrderID[orders[i].ID]; ok {
				orders[i].Items = items
			} else {
				orders[i].Items = []domain.OrderItem{}
			}
		}
	}

	return orders, totalCount, nil
}

// UpdateStatus changes the status of an order and optionally sets a cancel reason.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status string, reason string) error {
	query := `
		UPDATE orders
		SET status = $1, canceled_reason = $2, updated_at = $3
		WHERE id = $4`

	ct, err := r.pool.Exec(ctx, query, status, reason, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("order", id)
	}

	return nil
}

// loadOrderItems retrieves all items belonging to a given order.
func (r *OrderRepository) loadOrderItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, variant_id, name, sku, price, quantity
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.VariantID,
			&item.Name,
			&item.SKU,
			&item.Price,
			&item.Quantity,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order item rows: %w", err)
	}

	if items == nil {
		items = []domain.OrderItem{}
	}

	return items, nil
}
