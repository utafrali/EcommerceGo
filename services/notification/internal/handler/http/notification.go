package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/pkg/validator"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
	"github.com/utafrali/EcommerceGo/services/notification/internal/service"
)

// NotificationHandler handles HTTP requests for notification endpoints.
type NotificationHandler struct {
	service *service.NotificationService
	logger  *slog.Logger
}

// NewNotificationHandler creates a new notification HTTP handler.
func NewNotificationHandler(svc *service.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// SendNotificationRequest is the JSON request body for sending a notification.
type SendNotificationRequest struct {
	UserID   string         `json:"user_id" validate:"required,uuid"`
	Type     string         `json:"type" validate:"required,oneof=email sms push"`
	Channel  string         `json:"channel" validate:"required"`
	Subject  string         `json:"subject"`
	Body     string         `json:"body" validate:"required"`
	Priority string         `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	Metadata map[string]any `json:"metadata"`
}

// --- Response envelope ---

type listResponse struct {
	Data       any `json:"data"`
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
}

// --- Handlers ---

// SendNotification handles POST /api/v1/notifications
func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1 MB to prevent abuse.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req SendNotificationRequest
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

	input := &service.SendNotificationInput{
		UserID:   req.UserID,
		Type:     req.Type,
		Channel:  req.Channel,
		Subject:  req.Subject,
		Body:     req.Body,
		Priority: req.Priority,
		Metadata: req.Metadata,
	}

	notification, err := h.service.SendNotification(r.Context(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: notification})
}

// GetNotification handles GET /api/v1/notifications/{id}
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "notification id is required"},
		})
		return
	}

	notification, err := h.service.GetNotification(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: notification})
}

// ListNotificationsByUser handles GET /api/v1/notifications/user/{userId}
func (h *NotificationHandler) ListNotificationsByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "user id is required"},
		})
		return
	}

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

	notifications, total, err := h.service.ListNotificationsByUser(r.Context(), userID, page, perPage)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	if notifications == nil {
		notifications = []domain.Notification{}
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	httputil.WriteJSON(w, http.StatusOK, listResponse{
		Data:       notifications,
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
	})
}

// MarkAsRead handles PUT /api/v1/notifications/{id}/read
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "notification id is required"},
		})
		return
	}

	notification, err := h.service.MarkAsRead(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: notification})
}

// RetryNotification handles POST /api/v1/notifications/{id}/retry
func (h *NotificationHandler) RetryNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "notification id is required"},
		})
		return
	}

	notification, err := h.service.RetryNotification(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: notification})
}

