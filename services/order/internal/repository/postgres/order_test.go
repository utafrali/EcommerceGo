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
	"github.com/utafrali/EcommerceGo/services/order/internal/domain"
	"github.com/utafrali/EcommerceGo/services/order/internal/repository"
)

// --- Test Helpers ---

func newTestRepo(t *testing.T) (*OrderRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	repo := NewOrderRepository(mock)
	return repo, mock
}

func sampleAddress() *domain.Address {
	return &domain.Address{
		FullName:    "John Doe",
		AddressLine: "123 Main St",
		City:        "Istanbul",
		State:       "Istanbul",
		PostalCode:  "34000",
		Country:     "TR",
		Phone:       "+905551234567",
	}
}

func sampleOrder() *domain.Order {
	now := time.Now().UTC().Truncate(time.Microsecond)
	addr := sampleAddress()
	return &domain.Order{
		ID:              "order-001",
		UserID:          "user-001",
		Status:          domain.OrderStatusPending,
		SubtotalAmount:  10000,
		DiscountAmount:  500,
		ShippingAmount:  1000,
		TotalAmount:     10500,
		Currency:        "TRY",
		ShippingAddress: addr,
		BillingAddress:  addr,
		Notes:           "Leave at door",
		CanceledReason:  "",
		CreatedAt:       now,
		UpdatedAt:       now,
		Items: []domain.OrderItem{
			{
				ID:        "item-001",
				OrderID:   "order-001",
				ProductID: "prod-001",
				VariantID: "var-001",
				Name:      "Widget",
				SKU:       "WDG-001",
				Price:     5000,
				Quantity:  1,
				Subtotal:  5000,
			},
			{
				ID:        "item-002",
				OrderID:   "order-001",
				ProductID: "prod-002",
				VariantID: "var-002",
				Name:      "Gadget",
				SKU:       "GDG-001",
				Price:     2500,
				Quantity:  2,
				Subtotal:  5000,
			},
		},
	}
}

// --- Create Tests ---

func TestOrderRepository_Create_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	o := sampleOrder()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(
			o.ID, o.UserID, o.Status,
			o.SubtotalAmount, o.DiscountAmount, o.ShippingAmount, o.TotalAmount,
			o.Currency,
			pgxmock.AnyArg(), // shipping JSON
			pgxmock.AnyArg(), // billing JSON
			o.Notes, o.CanceledReason,
			o.CreatedAt, o.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	for _, item := range o.Items {
		mock.ExpectExec("INSERT INTO order_items").
			WithArgs(
				item.ID, item.OrderID, item.ProductID, item.VariantID,
				item.Name, item.SKU, item.Price, item.Quantity,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	mock.ExpectCommit()

	err := repo.Create(context.Background(), o)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_Create_NoItems(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	o := sampleOrder()
	o.Items = nil

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO orders").
		WithArgs(
			o.ID, o.UserID, o.Status,
			o.SubtotalAmount, o.DiscountAmount, o.ShippingAmount, o.TotalAmount,
			o.Currency,
			pgxmock.AnyArg(), pgxmock.AnyArg(),
			o.Notes, o.CanceledReason,
			o.CreatedAt, o.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// No item inserts expected.
	mock.ExpectCommit()

	err := repo.Create(context.Background(), o)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_Create_BeginError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectBegin().WillReturnError(errors.New("connection refused"))

	err := repo.Create(context.Background(), sampleOrder())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin transaction")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_Create_OrderInsertError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	o := sampleOrder()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO orders").
		WithArgs(
			o.ID, o.UserID, o.Status,
			o.SubtotalAmount, o.DiscountAmount, o.ShippingAmount, o.TotalAmount,
			o.Currency,
			pgxmock.AnyArg(), pgxmock.AnyArg(),
			o.Notes, o.CanceledReason,
			o.CreatedAt, o.UpdatedAt,
		).
		WillReturnError(errors.New("duplicate key"))
	mock.ExpectRollback()

	err := repo.Create(context.Background(), o)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert order")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_Create_ItemInsertError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	o := sampleOrder()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO orders").
		WithArgs(
			o.ID, o.UserID, o.Status,
			o.SubtotalAmount, o.DiscountAmount, o.ShippingAmount, o.TotalAmount,
			o.Currency,
			pgxmock.AnyArg(), pgxmock.AnyArg(),
			o.Notes, o.CanceledReason,
			o.CreatedAt, o.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// First item succeeds.
	item0 := o.Items[0]
	mock.ExpectExec("INSERT INTO order_items").
		WithArgs(
			item0.ID, item0.OrderID, item0.ProductID, item0.VariantID,
			item0.Name, item0.SKU, item0.Price, item0.Quantity,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Second item fails.
	item1 := o.Items[1]
	mock.ExpectExec("INSERT INTO order_items").
		WithArgs(
			item1.ID, item1.OrderID, item1.ProductID, item1.VariantID,
			item1.Name, item1.SKU, item1.Price, item1.Quantity,
		).
		WillReturnError(errors.New("constraint violation"))
	mock.ExpectRollback()

	err := repo.Create(context.Background(), o)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert order item")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetByID Tests ---

func TestOrderRepository_GetByID_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)
	addr := sampleAddress()

	shippingJSON, err := json.Marshal(addr)
	require.NoError(t, err)
	billingJSON, err := json.Marshal(addr)
	require.NoError(t, err)

	itemsJSON, err := json.Marshal([]map[string]any{
		{
			"id":         "item-001",
			"product_id": "prod-001",
			"variant_id": "var-001",
			"name":       "Widget",
			"sku":        "WDG-001",
			"price":      5000,
			"quantity":   1,
			"subtotal":   5000,
		},
		{
			"id":         "item-002",
			"product_id": "prod-002",
			"variant_id": "var-002",
			"name":       "Gadget",
			"sku":        "GDG-001",
			"price":      2500,
			"quantity":   2,
			"subtotal":   5000,
		},
	})
	require.NoError(t, err)

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"items",
	}).AddRow(
		"order-001", "user-001", "pending",
		int64(10000), int64(500), int64(1000), int64(10500),
		"TRY", shippingJSON, billingJSON,
		"Leave at door", "", now, now,
		itemsJSON,
	)

	mock.ExpectQuery("SELECT").
		WithArgs("order-001").
		WillReturnRows(rows)

	order, err := repo.GetByID(context.Background(), "order-001")
	require.NoError(t, err)
	require.NotNil(t, order)

	assert.Equal(t, "order-001", order.ID)
	assert.Equal(t, "user-001", order.UserID)
	assert.Equal(t, "pending", order.Status)
	assert.Equal(t, int64(10000), order.SubtotalAmount)
	assert.Equal(t, int64(10500), order.TotalAmount)
	assert.Equal(t, "TRY", order.Currency)
	assert.Equal(t, "Leave at door", order.Notes)

	require.NotNil(t, order.ShippingAddress)
	assert.Equal(t, "John Doe", order.ShippingAddress.FullName)
	assert.Equal(t, "Istanbul", order.ShippingAddress.City)

	require.NotNil(t, order.BillingAddress)
	assert.Equal(t, "John Doe", order.BillingAddress.FullName)

	require.Len(t, order.Items, 2)
	assert.Equal(t, "item-001", order.Items[0].ID)
	assert.Equal(t, "Widget", order.Items[0].Name)
	assert.Equal(t, int64(5000), order.Items[0].Price)
	assert.Equal(t, "item-002", order.Items[1].ID)
	assert.Equal(t, "Gadget", order.Items[1].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_GetByID_NoItems(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"items",
	}).AddRow(
		"order-002", "user-002", "confirmed",
		int64(5000), int64(0), int64(500), int64(5500),
		"USD", nil, nil,
		"", "", now, now,
		[]byte("[]"),
	)

	mock.ExpectQuery("SELECT").
		WithArgs("order-002").
		WillReturnRows(rows)

	order, err := repo.GetByID(context.Background(), "order-002")
	require.NoError(t, err)
	require.NotNil(t, order)

	assert.Equal(t, "order-002", order.ID)
	assert.Nil(t, order.ShippingAddress)
	assert.Nil(t, order.BillingAddress)
	assert.Empty(t, order.Items)
	assert.NotNil(t, order.Items) // should be [] not nil

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectQuery("SELECT").
		WithArgs("nonexistent-id").
		WillReturnError(pgx.ErrNoRows)

	order, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Nil(t, order)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_GetByID_ScanError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectQuery("SELECT").
		WithArgs("order-err").
		WillReturnError(errors.New("connection reset"))

	order, err := repo.GetByID(context.Background(), "order-err")
	assert.Nil(t, order)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan order")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- List Tests ---

func TestOrderRepository_List_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)

	addr := sampleAddress()
	shippingJSON, err := json.Marshal(addr)
	require.NoError(t, err)

	// Main query returns 2 orders with count(*) OVER() = 2.
	orderRows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"total_count",
	}).
		AddRow(
			"order-001", "user-001", "pending",
			int64(10000), int64(0), int64(1000), int64(11000),
			"TRY", shippingJSON, nil,
			"", "", now, now, 2,
		).
		AddRow(
			"order-002", "user-001", "confirmed",
			int64(5000), int64(500), int64(500), int64(5000),
			"TRY", nil, nil,
			"Fast delivery", "", now, now, 2,
		)

	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(10, 0).
		WillReturnRows(orderRows)

	// Batch items query.
	itemRows := pgxmock.NewRows([]string{
		"id", "order_id", "product_id", "variant_id", "name", "sku", "price", "quantity", "subtotal",
	}).
		AddRow("item-001", "order-001", "prod-001", "var-001", "Widget", "WDG-001", int64(5000), 2, int64(10000)).
		AddRow("item-002", "order-002", "prod-002", "var-002", "Gadget", "GDG-001", int64(2500), 2, int64(5000))

	mock.ExpectQuery("SELECT .+ FROM order_items").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(itemRows)

	filter := repository.OrderFilter{Page: 1, PerPage: 10}
	orders, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	require.Len(t, orders, 2)

	assert.Equal(t, "order-001", orders[0].ID)
	require.NotNil(t, orders[0].ShippingAddress)
	assert.Equal(t, "John Doe", orders[0].ShippingAddress.FullName)
	require.Len(t, orders[0].Items, 1)
	assert.Equal(t, "item-001", orders[0].Items[0].ID)

	assert.Equal(t, "order-002", orders[1].ID)
	assert.Nil(t, orders[1].ShippingAddress)
	require.Len(t, orders[1].Items, 1)
	assert.Equal(t, "item-002", orders[1].Items[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_List_WithUserFilter(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)
	userID := "user-filtered"

	orderRows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"total_count",
	}).AddRow(
		"order-100", userID, "pending",
		int64(3000), int64(0), int64(0), int64(3000),
		"TRY", nil, nil,
		"", "", now, now, 1,
	)

	// With user_id filter: args are user_id, limit, offset.
	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(userID, 20, 0).
		WillReturnRows(orderRows)

	itemRows := pgxmock.NewRows([]string{
		"id", "order_id", "product_id", "variant_id", "name", "sku", "price", "quantity", "subtotal",
	}).AddRow("item-100", "order-100", "prod-100", "", "Item", "SKU-100", int64(3000), 1, int64(3000))

	mock.ExpectQuery("SELECT .+ FROM order_items").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(itemRows)

	filter := repository.OrderFilter{UserID: &userID, Page: 1, PerPage: 20}
	orders, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 1, total)
	require.Len(t, orders, 1)
	assert.Equal(t, "order-100", orders[0].ID)
	assert.Equal(t, userID, orders[0].UserID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_List_WithStatusFilter(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)
	status := "shipped"

	orderRows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"total_count",
	}).AddRow(
		"order-200", "user-010", status,
		int64(7500), int64(0), int64(1000), int64(8500),
		"USD", nil, nil,
		"", "", now, now, 1,
	)

	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(status, 10, 0).
		WillReturnRows(orderRows)

	itemRows := pgxmock.NewRows([]string{
		"id", "order_id", "product_id", "variant_id", "name", "sku", "price", "quantity", "subtotal",
	})

	mock.ExpectQuery("SELECT .+ FROM order_items").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(itemRows)

	filter := repository.OrderFilter{Status: &status, Page: 1, PerPage: 10}
	orders, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 1, total)
	require.Len(t, orders, 1)
	assert.Equal(t, "shipped", orders[0].Status)
	// No items matched, but should have empty slice.
	assert.Empty(t, orders[0].Items)
	assert.NotNil(t, orders[0].Items)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_List_Empty(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	orderRows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"total_count",
	})

	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(20, 0).
		WillReturnRows(orderRows)

	// No batch items query expected because orders slice is empty.

	filter := repository.OrderFilter{Page: 1, PerPage: 20}
	orders, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 0, total)
	assert.Empty(t, orders)
	assert.NotNil(t, orders) // should be [] not nil

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_List_DefaultPerPage(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)

	orderRows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"total_count",
	}).AddRow(
		"order-300", "user-020", "pending",
		int64(1000), int64(0), int64(0), int64(1000),
		"TRY", nil, nil,
		"", "", now, now, 1,
	)

	// PerPage=0 should default to 20; args: limit=20, offset=0.
	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(20, 0).
		WillReturnRows(orderRows)

	itemRows := pgxmock.NewRows([]string{
		"id", "order_id", "product_id", "variant_id", "name", "sku", "price", "quantity", "subtotal",
	})

	mock.ExpectQuery("SELECT .+ FROM order_items").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(itemRows)

	filter := repository.OrderFilter{Page: 0, PerPage: 0}
	orders, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 1, total)
	require.Len(t, orders, 1)
	assert.Equal(t, "order-300", orders[0].ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_List_QueryError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(20, 0).
		WillReturnError(errors.New("database timeout"))

	filter := repository.OrderFilter{Page: 1, PerPage: 20}
	orders, total, err := repo.List(context.Background(), filter)
	assert.Nil(t, orders)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list orders")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_List_ItemsQueryError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	now := time.Now().UTC().Truncate(time.Microsecond)

	orderRows := pgxmock.NewRows([]string{
		"id", "user_id", "status", "subtotal_amount", "discount_amount",
		"shipping_amount", "total_amount", "currency", "shipping_address",
		"billing_address", "notes", "canceled_reason", "created_at", "updated_at",
		"total_count",
	}).AddRow(
		"order-400", "user-030", "pending",
		int64(2000), int64(0), int64(0), int64(2000),
		"TRY", nil, nil,
		"", "", now, now, 1,
	)

	mock.ExpectQuery("SELECT .+ FROM orders").
		WithArgs(20, 0).
		WillReturnRows(orderRows)

	mock.ExpectQuery("SELECT .+ FROM order_items").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("batch query failed"))

	filter := repository.OrderFilter{Page: 1, PerPage: 20}
	orders, total, err := repo.List(context.Background(), filter)
	assert.Nil(t, orders)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch load order items")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- UpdateStatus Tests ---

func TestOrderRepository_UpdateStatus_Success(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectExec("UPDATE orders").
		WithArgs("confirmed", "", pgxmock.AnyArg(), "order-001").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateStatus(context.Background(), "order-001", "confirmed", "")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_UpdateStatus_WithReason(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectExec("UPDATE orders").
		WithArgs("canceled", "out of stock", pgxmock.AnyArg(), "order-002").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateStatus(context.Background(), "order-002", "canceled", "out of stock")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_UpdateStatus_NotFound(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectExec("UPDATE orders").
		WithArgs("shipped", "", pgxmock.AnyArg(), "nonexistent").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.UpdateStatus(context.Background(), "nonexistent", "shipped", "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderRepository_UpdateStatus_ExecError(t *testing.T) {
	repo, mock := newTestRepo(t)
	defer mock.ExpectationsWereMet()

	mock.ExpectExec("UPDATE orders").
		WithArgs("processing", "", pgxmock.AnyArg(), "order-003").
		WillReturnError(errors.New("write conflict"))

	err := repo.UpdateStatus(context.Background(), "order-003", "processing", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update order status")

	assert.NoError(t, mock.ExpectationsWereMet())
}
