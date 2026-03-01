package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/domain"
	"github.com/utafrali/EcommerceGo/services/inventory/internal/service"
)

// InventoryHandler handles HTTP requests for inventory endpoints.
type InventoryHandler struct {
	service *service.InventoryService
	logger  *slog.Logger
}

// NewInventoryHandler creates a new inventory HTTP handler.
func NewInventoryHandler(svc *service.InventoryService, logger *slog.Logger) *InventoryHandler {
	return &InventoryHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// InitializeStockRequest is the JSON request body for creating/initializing stock.
type InitializeStockRequest struct {
	ProductID         string `json:"product_id" validate:"required,uuid"`
	VariantID         string `json:"variant_id" validate:"required,uuid"`
	WarehouseID       string `json:"warehouse_id" validate:"omitempty,uuid"`
	Quantity          int    `json:"quantity" validate:"gte=0"`
	LowStockThreshold int    `json:"low_stock_threshold" validate:"omitempty,gte=0"`
}

// AdjustStockRequest is the JSON request body for adjusting stock.
type AdjustStockRequest struct {
	Delta  int    `json:"delta" validate:"required"`
	Reason string `json:"reason" validate:"required,oneof=order return adjustment reservation"`
}

// CheckAvailabilityRequest is the JSON request body for checking availability.
type CheckAvailabilityRequest struct {
	Items []StockCheckItemRequest `json:"items" validate:"required,min=1,dive"`
}

// StockCheckItemRequest represents a single item in an availability check request.
type StockCheckItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	VariantID string `json:"variant_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
}

// ReserveStockRequest is the JSON request body for reserving stock.
type ReserveStockRequest struct {
	CheckoutID string                  `json:"checkout_id" validate:"required,uuid"`
	Items      []StockCheckItemRequest `json:"items" validate:"required,min=1,dive"`
	TTLSeconds int                     `json:"ttl_seconds" validate:"omitempty,gte=1"`
}

// ReleaseReservationRequest is the JSON request body for releasing a reservation.
type ReleaseReservationRequest struct {
	ReservationID string `json:"reservation_id" validate:"required,uuid"`
}

// ConfirmReservationRequest is the JSON request body for confirming a reservation.
type ConfirmReservationRequest struct {
	ReservationID string `json:"reservation_id" validate:"required,uuid"`
}

// --- Handlers ---

// InitializeStock handles POST /api/v1/inventory
func (h *InventoryHandler) InitializeStock(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req InitializeStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	lowStockThreshold := req.LowStockThreshold
	if lowStockThreshold == 0 {
		lowStockThreshold = 10 // sensible default
	}

	stock := &domain.Stock{
		ProductID:         req.ProductID,
		VariantID:         req.VariantID,
		WarehouseID:       req.WarehouseID,
		Quantity:          req.Quantity,
		LowStockThreshold: lowStockThreshold,
	}

	result, err := h.service.InitializeStock(r.Context(), stock)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: result})
}

// GetStock handles GET /api/v1/inventory/{productId}/variants/{variantId}
func (h *InventoryHandler) GetStock(w http.ResponseWriter, r *http.Request) {
	productID, ok := httputil.ParseUUID(w, chi.URLParam(r, "productId"))
	if !ok {
		return
	}
	variantID, ok := httputil.ParseUUID(w, chi.URLParam(r, "variantId"))
	if !ok {
		return
	}

	stock, err := h.service.GetStock(r.Context(), productID.String(), variantID.String())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: stock})
}

// AdjustStock handles PUT /api/v1/inventory/{productId}/variants/{variantId}
func (h *InventoryHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	productID, ok := httputil.ParseUUID(w, chi.URLParam(r, "productId"))
	if !ok {
		return
	}
	variantID, ok := httputil.ParseUUID(w, chi.URLParam(r, "variantId"))
	if !ok {
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	stock, err := h.service.AdjustStock(r.Context(), productID.String(), variantID.String(), req.Delta, req.Reason)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: stock})
}

// CheckAvailability handles POST /api/v1/inventory/check
func (h *InventoryHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req CheckAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	items := make([]domain.StockCheckItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = domain.StockCheckItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		}
	}

	results, allAvailable, err := h.service.CheckAvailability(r.Context(), items)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]any{
		"items":         results,
		"all_available": allAvailable,
	}})
}

// ReserveStock handles POST /api/v1/inventory/reserve
func (h *InventoryHandler) ReserveStock(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req ReserveStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	items := make([]domain.StockCheckItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = domain.StockCheckItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		}
	}

	reservationIDs, err := h.service.ReserveStock(r.Context(), req.CheckoutID, items, req.TTLSeconds)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: map[string]any{
		"reservation_ids": reservationIDs,
		"checkout_id":     req.CheckoutID,
	}})
}

// ReleaseReservation handles POST /api/v1/inventory/release
func (h *InventoryHandler) ReleaseReservation(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req ReleaseReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	if err := h.service.ReleaseReservation(r.Context(), req.ReservationID); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{
		"reservation_id": req.ReservationID,
		"status":         "released",
	}})
}

// ConfirmReservation handles POST /api/v1/inventory/confirm
func (h *InventoryHandler) ConfirmReservation(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req ConfirmReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		httputil.WriteValidationError(w, err)
		return
	}

	if err := h.service.ConfirmReservation(r.Context(), req.ReservationID); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{
		"reservation_id": req.ReservationID,
		"status":         "confirmed",
	}})
}

// ListLowStock handles GET /api/v1/inventory/low-stock
func (h *InventoryHandler) ListLowStock(w http.ResponseWriter, r *http.Request) {
	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "page must be a valid positive integer"},
			})
			return
		}
		page = p
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		pp, err := strconv.Atoi(v)
		if err != nil || pp < 1 || pp > 100 {
			httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
				Error: &httputil.ErrorResponse{Code: "INVALID_PARAMETER", Message: "per_page must be a valid integer between 1 and 100"},
			})
			return
		}
		perPage = pp
	}

	stocks, total, err := h.service.ListLowStock(r.Context(), page, perPage)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.NewPaginatedResponse[domain.Stock](stocks, total, page, perPage))
}
