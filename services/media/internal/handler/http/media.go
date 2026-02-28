package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/utafrali/EcommerceGo/pkg/httputil"
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
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "failed to parse multipart form: " + err.Error()},
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "file is required: " + err.Error()},
		})
		return
	}
	defer file.Close()

	ownerID := r.FormValue("owner_id")
	ownerType := r.FormValue("owner_type")
	altText := r.FormValue("alt_text")

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	input := &service.UploadMediaInput{
		OwnerID:     ownerID,
		OwnerType:   ownerType,
		FileName:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
		Data:        file,
		AltText:     altText,
	}

	media, err := h.service.UploadMedia(r.Context(), input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response{Data: media})
}

// GetMedia handles GET /api/v1/media/{id}.
func (h *MediaHandler) GetMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "media id is required"},
		})
		return
	}

	media, err := h.service.GetMedia(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: media})
}

// ListMediaByOwner handles GET /api/v1/media/owner/{ownerType}/{ownerId}.
func (h *MediaHandler) ListMediaByOwner(w http.ResponseWriter, r *http.Request) {
	ownerType := chi.URLParam(r, "ownerType")
	ownerID := chi.URLParam(r, "ownerId")

	if ownerType == "" || ownerID == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "owner type and owner id are required"},
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

	mediaFiles, total, err := h.service.ListMediaByOwner(r.Context(), ownerID, ownerType, page, perPage)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	httputil.WriteJSON(w, http.StatusOK, listResponse{
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
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "media id is required"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	var req UpdateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "invalid request body: " + err.Error()},
		})
		return
	}

	input := &service.UpdateMediaInput{
		AltText:   req.AltText,
		SortOrder: req.SortOrder,
	}

	media, err := h.service.UpdateMediaMetadata(r.Context(), id, input)
	if err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: media})
}

// DeleteMedia handles DELETE /api/v1/media/{id}.
func (h *MediaHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httputil.WriteJSON(w, http.StatusBadRequest, httputil.Response{
			Error: &httputil.ErrorResponse{Code: "INVALID_INPUT", Message: "media id is required"},
		})
		return
	}

	if err := h.service.DeleteMedia(r.Context(), id); err != nil {
		httputil.WriteError(w, r, err, h.logger)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.Response{Data: map[string]string{"id": id, "status": "deleted"}})
}

