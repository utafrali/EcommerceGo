package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/cart/internal/service"
)

// CartHandler handles HTTP requests for cart endpoints.
type CartHandler struct {
	service *service.CartService
	logger  *slog.Logger
}

// NewCartHandler creates a new cart HTTP handler.
func NewCartHandler(svc *service.CartService, logger *slog.Logger) *CartHandler {
	return &CartHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// AddItemRequest is the JSON request body for adding an item to the cart.
type AddItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	VariantID string `json:"variant_id" validate:"required"`
	Name      string `json:"name" validate:"required,min=1,max=500"`
	SKU       string `json:"sku" validate:"required"`
	Price     int64  `json:"price" validate:"required,gte=0"`
	Quantity  int    `json:"quantity" validate:"required,gte=1"`
	ImageURL  string `json:"image_url"`
}

// UpdateQuantityRequest is the JSON request body for updating an item's quantity.
type UpdateQuantityRequest struct {
	Quantity int `json:"quantity" validate:"gte=0"`
}

// --- Handlers ---

// GetCart handles GET /api/v1/cart
// @Summary Get cart contents
// @Description Returns the current user's cart including all items, total amount, and item count
// @Tags cart
// @Produce json
// @Param X-User-ID header string true "Authenticated user UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/cart/ [get]
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
		})
		return
	}

	cart, err := h.service.GetCart(r.Context(), userID)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: cart})
}

// AddItem handles POST /api/v1/cart/items
// @Summary Add item to cart
// @Description Adds a product variant to the cart. If the item already exists, its quantity is incremented.
// @Tags cart
// @Accept json
// @Produce json
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param request body AddItemRequest true "Item to add"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/v1/cart/items [post]
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
		})
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req AddItemRequest
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

	input := service.AddItemInput{
		ProductID: req.ProductID,
		VariantID: req.VariantID,
		Name:      req.Name,
		SKU:       req.SKU,
		Price:     req.Price,
		Quantity:  req.Quantity,
		ImageURL:  req.ImageURL,
	}

	cart, err := h.service.AddItem(r.Context(), userID, input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: cart})
}

// UpdateItemQuantity handles PUT /api/v1/cart/items/{productId}/{variantId}
// @Summary Update cart item quantity
// @Description Updates the quantity of a specific item. Set quantity to 0 to remove the item.
// @Tags cart
// @Accept json
// @Produce json
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param productId path string true "Product UUID"
// @Param variantId path string true "Product variant UUID"
// @Param request body UpdateQuantityRequest true "New quantity"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/v1/cart/items/{productId}/{variantId} [put]
func (h *CartHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
		})
		return
	}

	productID, ok2 := httputil.ParseUUID(w, chi.URLParam(r, "productId"))
	if !ok2 {
		return
	}
	variantID, ok3 := httputil.ParseUUID(w, chi.URLParam(r, "variantId"))
	if !ok3 {
		return
	}

	// Limit request body to 1MB to prevent DoS via large payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req UpdateQuantityRequest
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

	cart, err := h.service.UpdateItemQuantity(r.Context(), userID, productID.String(), variantID.String(), req.Quantity)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: cart})
}

// RemoveItem handles DELETE /api/v1/cart/items/{productId}/{variantId}
// @Summary Remove cart item
// @Description Removes a specific product variant from the cart
// @Tags cart
// @Produce json
// @Param X-User-ID header string true "Authenticated user UUID"
// @Param productId path string true "Product UUID"
// @Param variantId path string true "Product variant UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/v1/cart/items/{productId}/{variantId} [delete]
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
		})
		return
	}

	productID, ok2 := httputil.ParseUUID(w, chi.URLParam(r, "productId"))
	if !ok2 {
		return
	}
	variantID, ok3 := httputil.ParseUUID(w, chi.URLParam(r, "variantId"))
	if !ok3 {
		return
	}

	cart, err := h.service.RemoveItem(r.Context(), userID, productID.String(), variantID.String())
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: cart})
}

// ClearCart handles DELETE /api/v1/cart
// @Summary Clear cart
// @Description Removes all items from the current user's cart
// @Tags cart
// @Produce json
// @Param X-User-ID header string true "Authenticated user UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/cart/ [delete]
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "authentication required"},
		})
		return
	}

	if err := h.service.ClearCart(r.Context(), userID); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{"status": "cleared"}})
}
