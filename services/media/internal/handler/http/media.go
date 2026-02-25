package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
	"github.com/utafrali/EcommerceGo/services/media/internal/service"
)

// MediaHandler handles HTTP requests for media endpoints.
type MediaHandler struct {
	service *service.MediaService
	logger  *slog.Logger
}

// NewMediaHandler creates a new media HTTP handler.
func NewMediaHandler(svc *service.MediaService, logger *slog.Logger) *MediaHandler {
	return &MediaHandler{
		service: svc,
		logger:  logger,
	}
}

// --- Request DTOs ---

// UpdateMediaRequest is the JSON request body for updating media metadata.
type UpdateMediaRequest struct {
	AltText   *string `json:"alt_text"`
	SortOrder *int    `json:"sort_order"`
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

// UploadMedia handles POST /api/v1/media (multipart/form-data).
func (h *MediaHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with max file size limit.
	maxSize := domain.MaxFileSize + (1 << 20) // Add 1MB overhead for form fields.
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(domain.MaxFileSize); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "failed to parse multipart form: " + err.Error()},
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "file is required: " + err.Error()},
		})
		return
	}
	defer file.Close()

	ownerID := r.FormValue("owner_id")
	ownerType := r.FormValue("owner_type")
	altText := r.FormValue("alt_text")

	input := &service.UploadMediaInput{
		OwnerID:     ownerID,
		OwnerType:   ownerType,
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Data:        file,
		AltText:     altText,
	}

	media, err := h.service.UploadMedia(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: media})
}

// GetMedia handles GET /api/v1/media/{id}.
func (h *MediaHandler) GetMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "media id is required"},
		})
		return
	}

	media, err := h.service.GetMedia(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: media})
}

// ListMediaByOwner handles GET /api/v1/media/owner/{ownerType}/{ownerId}.
func (h *MediaHandler) ListMediaByOwner(w http.ResponseWriter, r *http.Request) {
	ownerType := chi.URLParam(r, "ownerType")
	ownerID := chi.URLParam(r, "ownerId")

	if ownerType == "" || ownerID == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "owner type and owner id are required"},
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

	mediaFiles, total, err := h.service.ListMediaByOwner(r.Context(), ownerID, ownerType, page, perPage)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, listResponse{
		Data:       mediaFiles,
		TotalCount: total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// UpdateMediaMetadata handles PUT /api/v1/media/{id}.
func (h *MediaHandler) UpdateMediaMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "media id is required"},
		})
		return
	}

	var req UpdateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	input := &service.UpdateMediaInput{
		AltText:   req.AltText,
		SortOrder: req.SortOrder,
	}

	media, err := h.service.UpdateMediaMetadata(r.Context(), id, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: media})
}

// DeleteMedia handles DELETE /api/v1/media/{id}.
func (h *MediaHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{
			Error: &errorResponse{Code: "INVALID_INPUT", Message: "media id is required"},
		})
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id); err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"id": id, "status": "deleted"}})
}

// --- Helpers ---

func (h *MediaHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
