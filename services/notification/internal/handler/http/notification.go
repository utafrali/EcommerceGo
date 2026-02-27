package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
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

type response struct {
	Data  any            `json:"data,omitempty"`
	Error *errorResponse `json:"error,omitempty"`
}

type errorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type listResponse struct {
	Data       any `json:"data"`
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// --- Handlers ---

// SendNotification handles POST /api/v1/notifications
func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 1 MB to prevent abuse.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	if err := validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
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
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: notification})
}

// GetNotification handles GET /api/v1/notifications/{id}
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "notification id is required"},
		})
		return
	}

	notification, err := h.service.GetNotification(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: notification})
}

// ListNotificationsByUser handles GET /api/v1/notifications/user/{userId}
func (h *NotificationHandler) ListNotificationsByUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "user id is required"},
		})
		return
	}

	page := 1
	perPage := 20

	if v := r.URL.Query().Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_PARAMETER", Message: "page must be a valid positive integer"},
			})
			return
		}
		page = p
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		pp, err := strconv.Atoi(v)
		if err != nil || pp < 1 || pp > 100 {
			writeJSON(w, http.StatusBadRequest, response{
				Error: &errorResponse{Code: "INVALID_PARAMETER", Message: "per_page must be a valid integer between 1 and 100"},
			})
			return
		}
		perPage = pp
	}

	notifications, total, err := h.service.ListNotificationsByUser(r.Context(), userID, page, perPage)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if notifications == nil {
		notifications = []domain.Notification{}
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       notifications,
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// MarkAsRead handles PUT /api/v1/notifications/{id}/read
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "notification id is required"},
		})
		return
	}

	notification, err := h.service.MarkAsRead(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: notification})
}

// RetryNotification handles POST /api/v1/notifications/{id}/retry
func (h *NotificationHandler) RetryNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "notification id is required"},
		})
		return
	}

	notification, err := h.service.RetryNotification(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: notification})
}

// --- Helpers ---

func (h *NotificationHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		writeJSON(w, appErr.Status, response{
			Error: &errorResponse{Code: appErr.Code, Message: appErr.Message},
		})
		return
	}

	status := apperrors.HTTPStatus(err)
	code := "INTERNAL_ERROR"
	message := "an internal error occurred"

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		code = "NOT_FOUND"
		message = "resource not found"
		status = http.StatusNotFound
	case errors.Is(err, apperrors.ErrInvalidInput):
		code = "INVALID_INPUT"
		message = err.Error()
		status = http.StatusBadRequest
	}

	if status == http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "internal error",
			slog.String("error", err.Error()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	writeJSON(w, status, response{
		Error: &errorResponse{Code: code, Message: message},
	})
}

func (h *NotificationHandler) writeValidationError(w http.ResponseWriter, err error) {
	var valErr *validator.ValidationError
	if errors.As(err, &valErr) {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{
				Code:    "VALIDATION_ERROR",
				Message: "request validation failed",
				Fields:  valErr.Fields(),
			},
		})
		return
	}

	writeJSON(w, http.StatusBadRequest, response{
		Error: &errorResponse{Code: "INVALID_INPUT", Message: err.Error()},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
