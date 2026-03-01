package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
)

// =============================================================================
// Mock BannerRepository
// =============================================================================

type mockBannerRepo struct {
	mock.Mock
}

func (m *mockBannerRepo) Create(ctx context.Context, banner *domain.Banner) error {
	args := m.Called(ctx, banner)
	return args.Error(0)
}

func (m *mockBannerRepo) GetByID(ctx context.Context, id string) (*domain.Banner, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Banner), args.Error(1)
}

func (m *mockBannerRepo) Update(ctx context.Context, banner *domain.Banner) error {
	args := m.Called(ctx, banner)
	return args.Error(0)
}

func (m *mockBannerRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockBannerRepo) List(ctx context.Context, filter domain.BannerFilter) ([]domain.Banner, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Banner), args.Int(1), args.Error(2)
}

// =============================================================================
// Test helpers
// =============================================================================

func bannerTestHandler(repo *mockBannerRepo) *BannerHandler {
	return NewBannerHandler(repo, productTestLogger())
}

func bannerRouter(handler *BannerHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/banners", func(r chi.Router) {
		r.Get("/", handler.ListBanners)
		r.Get("/{id}", handler.GetBanner)
		r.Post("/", handler.CreateBanner)
		r.Put("/{id}", handler.UpdateBanner)
		r.Delete("/{id}", handler.DeleteBanner)
	})
	return r
}

func decodeBannerResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

func sampleBanner() *domain.Banner {
	now := time.Now().UTC()
	return &domain.Banner{
		ID:        "550e8400-e29b-41d4-a716-446655440010",
		Title:     "Summer Sale",
		ImageURL:  "https://example.com/banner.jpg",
		LinkURL:   "/sale",
		LinkType:  domain.BannerLinkTypeInternal,
		Position:  domain.BannerPositionHeroSlider,
		SortOrder: 1,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func validCreateBannerJSON() []byte {
	body := CreateBannerRequest{
		Title:    "New Banner",
		ImageURL: "https://example.com/new-banner.jpg",
		LinkURL:  "/products",
		LinkType: "internal",
		Position: "hero_slider",
	}
	b, _ := json.Marshal(body)
	return b
}

// =============================================================================
// POST /api/v1/banners - CreateBanner
// =============================================================================

func TestCreateBanner_Success(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Banner")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/banners", bytes.NewReader(validCreateBannerJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeBannerResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestCreateBanner_InvalidJSON(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/banners", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestCreateBanner_ValidationError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	// Missing required fields: title, image_url, link_url, link_type, position
	body := CreateBannerRequest{}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/banners", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreateBanner_ServiceError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Banner")).
		Return(apperrors.Internal(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/banners", bytes.NewReader(validCreateBannerJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// =============================================================================
// GET /api/v1/banners - ListBanners
// =============================================================================

func TestListBanners_Success(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	banners := []domain.Banner{*sampleBanner()}
	repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.BannerFilter) bool {
		return f.Page == 2 && f.PerPage == 5
	})).Return(banners, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners?page=2&per_page=5", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var paginatedResp httputil.PaginatedResponse[json.RawMessage]
	err := json.NewDecoder(rec.Body).Decode(&paginatedResp)
	require.NoError(t, err)
	assert.Equal(t, 1, paginatedResp.TotalCount)
	assert.Equal(t, 2, paginatedResp.Page)
	assert.Equal(t, 5, paginatedResp.PerPage)
	assert.Len(t, paginatedResp.Data, 1)
	repo.AssertExpectations(t)
}

func TestListBanners_DefaultPagination(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.BannerFilter) bool {
		return f.Page == 1 && f.PerPage == 20
	})).Return([]domain.Banner{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListBanners_WithPositionFilter(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.BannerFilter) bool {
		return f.Position != nil && *f.Position == "hero_slider"
	})).Return([]domain.Banner{*sampleBanner()}, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners?position=hero_slider", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListBanners_WithIsActiveFilter(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.BannerFilter) bool {
		return f.IsActive != nil && *f.IsActive == true
	})).Return([]domain.Banner{*sampleBanner()}, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners?is_active=true", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListBanners_ServiceError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	repo.On("List", mock.Anything, mock.Anything).
		Return([]domain.Banner(nil), 0, apperrors.Internal(nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// =============================================================================
// GET /api/v1/banners/{id} - GetBanner
// =============================================================================

func TestGetBanner_Success(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	banner := sampleBanner()
	repo.On("GetByID", mock.Anything, banner.ID).Return(banner, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners/"+banner.ID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeBannerResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetBanner_NotFound(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, bannerID).
		Return(nil, apperrors.NotFound("banner", bannerID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners/"+bannerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestGetBanner_ServiceError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440010"
	repo.On("GetByID", mock.Anything, bannerID).
		Return(nil, apperrors.Internal(nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banners/"+bannerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// =============================================================================
// PUT /api/v1/banners/{id} - UpdateBanner
// =============================================================================

func TestUpdateBanner_Success(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	banner := sampleBanner()
	repo.On("GetByID", mock.Anything, banner.ID).Return(banner, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Banner")).Return(nil)

	newTitle := "Updated Banner Title"
	body := UpdateBannerRequest{Title: &newTitle}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/banners/"+banner.ID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeBannerResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestUpdateBanner_InvalidJSON(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440010"

	req := httptest.NewRequest(http.MethodPut, "/api/v1/banners/"+bannerID, bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestUpdateBanner_NotFound(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, bannerID).
		Return(nil, apperrors.NotFound("banner", bannerID))

	newTitle := "Updated"
	body := UpdateBannerRequest{Title: &newTitle}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/banners/"+bannerID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestUpdateBanner_ValidationError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440010"

	// Invalid link_type value
	badLinkType := "invalid_type"
	body := UpdateBannerRequest{LinkType: &badLinkType}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/banners/"+bannerID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
}

func TestUpdateBanner_RepoUpdateError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	banner := sampleBanner()
	repo.On("GetByID", mock.Anything, banner.ID).Return(banner, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Banner")).
		Return(apperrors.Internal(nil))

	newTitle := "Updated"
	body := UpdateBannerRequest{Title: &newTitle}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/banners/"+banner.ID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	repo.AssertExpectations(t)
}

// =============================================================================
// DELETE /api/v1/banners/{id} - DeleteBanner
// =============================================================================

func TestDeleteBanner_Success(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440010"
	repo.On("Delete", mock.Anything, bannerID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/banners/"+bannerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeBannerResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestDeleteBanner_NotFound(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("Delete", mock.Anything, bannerID).
		Return(apperrors.NotFound("banner", bannerID))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/banners/"+bannerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestDeleteBanner_ServiceError(t *testing.T) {
	repo := new(mockBannerRepo)
	handler := bannerTestHandler(repo)
	router := bannerRouter(handler)

	bannerID := "550e8400-e29b-41d4-a716-446655440010"
	repo.On("Delete", mock.Anything, bannerID).
		Return(apperrors.Internal(nil))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/banners/"+bannerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeBannerResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

// =============================================================================
// Table-driven: CreateBanner validation edge cases
// =============================================================================

func TestCreateBanner_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		expectStatus  int
		expectErrCode string
	}{
		{
			name:          "empty body",
			body:          `{}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "missing title",
			body:          `{"image_url": "https://example.com/img.jpg", "link_url": "/sale", "link_type": "internal", "position": "hero_slider"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "invalid image_url",
			body:          `{"title": "Test", "image_url": "not-a-url", "link_url": "/sale", "link_type": "internal", "position": "hero_slider"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "invalid link_type",
			body:          `{"title": "Test", "image_url": "https://example.com/img.jpg", "link_url": "/sale", "link_type": "unknown", "position": "hero_slider"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "invalid position",
			body:          `{"title": "Test", "image_url": "https://example.com/img.jpg", "link_url": "/sale", "link_type": "internal", "position": "invalid_position"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
		{
			name:          "missing link_url",
			body:          `{"title": "Test", "image_url": "https://example.com/img.jpg", "link_type": "internal", "position": "hero_slider"}`,
			expectStatus:  http.StatusBadRequest,
			expectErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockBannerRepo)
			handler := bannerTestHandler(repo)
			router := bannerRouter(handler)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/banners", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectStatus, rec.Code)
			resp := decodeBannerResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tt.expectErrCode, resp.Error.Code)
		})
	}
}

// =============================================================================
// Table-driven: UpdateBanner partial updates
// =============================================================================

func TestUpdateBanner_PartialUpdates_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "update title only",
			body: `{"title": "New Title"}`,
		},
		{
			name: "update position only",
			body: `{"position": "mid_banner"}`,
		},
		{
			name: "update is_active only",
			body: `{"is_active": false}`,
		},
		{
			name: "update sort_order only",
			body: `{"sort_order": 5}`,
		},
		{
			name: "update link_type only",
			body: `{"link_type": "external"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockBannerRepo)
			handler := bannerTestHandler(repo)
			router := bannerRouter(handler)

			banner := sampleBanner()
			repo.On("GetByID", mock.Anything, banner.ID).Return(banner, nil)
			repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Banner")).Return(nil)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/banners/"+banner.ID, bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			resp := decodeBannerResponse(t, rec)
			assert.Nil(t, resp.Error)
			assert.NotNil(t, resp.Data)
			repo.AssertExpectations(t)
		})
	}
}
