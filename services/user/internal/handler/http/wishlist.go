package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

// WishlistHandler handles HTTP requests for wishlist endpoints.
type WishlistHandler struct {
	repo domain.WishlistRepository
}

// NewWishlistHandler creates a new wishlist HTTP handler.
func NewWishlistHandler(repo domain.WishlistRepository) *WishlistHandler {
	return &WishlistHandler{repo: repo}
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
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	if err := h.repo.Add(r.Context(), userID, productID); err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{
		Data: map[string]string{"product_id": productID, "status": "added"},
	})
}

// Remove handles DELETE /api/v1/users/wishlist/{productId}
func (h *WishlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	if err := h.repo.Remove(r.Context(), userID, productID); err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: map[string]string{"product_id": productID, "status": "removed"},
	})
}

// List handles GET /api/v1/users/wishlist
func (h *WishlistHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
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
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{
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
		writeJSON(w, http.StatusUnauthorized, response{
			Error: &errorResponse{Code: "UNAUTHORIZED", Message: "user not authenticated"},
		})
		return
	}

	productID := chi.URLParam(r, "productId")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "product id is required"},
		})
		return
	}

	exists, err := h.repo.Exists(r.Context(), userID, productID)
	if err != nil {
		writeAppError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: WishlistExistsResponse{Exists: exists},
	})
}
