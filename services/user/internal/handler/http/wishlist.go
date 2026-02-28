package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

// WishlistHandler handles HTTP requests for wishlist endpoints.
type WishlistHandler struct {
	repo   domain.WishlistRepository
	logger *slog.Logger
}

// NewWishlistHandler creates a new wishlist HTTP handler.
func NewWishlistHandler(repo domain.WishlistRepository, logger *slog.Logger) *WishlistHandler {
	return &WishlistHandler{repo: repo, logger: logger}
}

// --- Response DTOs ---

// WishlistListResponse is the paginated response for listing wishlist items.
type WishlistListResponse struct {
	Items   []*domain.WishlistItem `json:"items"`
	Total   int                    `json:"total"`
	Page    int                    `json:"page"`
	PerPage int                    `json:"per_page"`
}

// WishlistExistsResponse indicates whether a product is in the wishlist.
type WishlistExistsResponse struct {
	Exists bool `json:"exists"`
}

// --- Handlers ---

// Add handles POST /api/v1/users/wishlist/{productId}
func (h *WishlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	if productID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	if err := h.repo.Add(r.Context(), userID, productID); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{
		Data: map[string]string{"product_id": productID, "status": "added"},
	})
}

// Remove handles DELETE /api/v1/users/wishlist/{productId}
func (h *WishlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	if productID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	if err := h.repo.Remove(r.Context(), userID, productID); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{
		Data: map[string]string{"product_id": productID, "status": "removed"},
	})
}

// List handles GET /api/v1/users/wishlist
func (h *WishlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}

	if v := r.URL.Query().Get("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	items, total, err := h.repo.List(r.Context(), userID, page, perPage)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{
		Data: WishlistListResponse{
			Items:   items,
			Total:   total,
			Page:    page,
			PerPage: perPage,
		},
	})
}

// Exists handles GET /api/v1/users/wishlist/{productId}
func (h *WishlistHandler) Exists(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		httputil.WriteJSON(w, http.StatusUnauthorized, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	if productID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	exists, err := h.repo.Exists(r.Context(), userID, productID)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{
		Data: WishlistExistsResponse{Exists: exists},
	})
}
