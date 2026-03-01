package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httputil"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/domain"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/event"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/repository"
	"github.com/utafrali/EcommerceGo/services/campaign/internal/service"
)

// ============================================================================
// Mock repository
// ============================================================================

type mockCampaignRepository struct {
	mock.Mock
}

func (m *mockCampaignRepository) Create(ctx context.Context, campaign *domain.Campaign) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *mockCampaignRepository) GetByID(ctx context.Context, id string) (*domain.Campaign, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepository) GetByCode(ctx context.Context, code string) (*domain.Campaign, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *mockCampaignRepository) List(ctx context.Context, filter repository.CampaignFilter) ([]domain.Campaign, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Campaign), args.Int(1), args.Error(2)
}

func (m *mockCampaignRepository) Update(ctx context.Context, campaign *domain.Campaign) error {
	args := m.Called(ctx, campaign)
	return args.Error(0)
}

func (m *mockCampaignRepository) IncrementUsage(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockCampaignRepository) RecordUsage(ctx context.Context, usage *domain.CampaignUsage) error {
	args := m.Called(ctx, usage)
	return args.Error(0)
}

func (m *mockCampaignRepository) GetStackingRules(ctx context.Context, campaignID string) ([]domain.StackingRule, error) {
	args := m.Called(ctx, campaignID)
	return args.Get(0).([]domain.StackingRule), args.Error(1)
}

func (m *mockCampaignRepository) CreateStackingRule(ctx context.Context, rule *domain.StackingRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *mockCampaignRepository) DeleteStackingRule(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ============================================================================
// Test helpers
// ============================================================================

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEventProducer() *event.Producer {
	logger := testLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func testCampaignService(repo *mockCampaignRepository) *service.CampaignService {
	return service.NewCampaignService(repo, testEventProducer(), testLogger())
}

func testCampaignHandler(repo *mockCampaignRepository) *CampaignHandler {
	svc := testCampaignService(repo)
	return NewCampaignHandler(svc, testLogger())
}

// setupCampaignRouter creates a chi router matching production route layout.
func setupCampaignRouter(handler *CampaignHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/campaigns", func(r chi.Router) {
		r.Post("/", handler.CreateCampaign)
		r.Get("/", handler.ListCampaigns)
		r.Delete("/stacking-rules/{ruleId}", handler.DeleteStackingRule)
		r.Get("/{id}", handler.GetCampaign)
		r.Put("/{id}", handler.UpdateCampaign)
		r.Post("/{id}/deactivate", handler.DeactivateCampaign)
		r.Post("/{id}/stacking-rules", handler.CreateStackingRule)
		r.Get("/{id}/stacking-rules", handler.GetStackingRules)
	})
	r.Route("/api/v1/coupons", func(r chi.Router) {
		r.Post("/validate", handler.ValidateCoupon)
		r.Post("/apply", handler.ApplyCoupon)
	})
	return r
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// listResponse is a type alias for the standardized PaginatedResponse.
type listResponse = httputil.PaginatedResponse[domain.Campaign]

func decodeListResponse(t *testing.T, rec *httptest.ResponseRecorder) listResponse {
	t.Helper()
	var resp listResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// sampleCampaign returns a domain.Campaign suitable for test assertions.
func sampleCampaign() *domain.Campaign {
	now := time.Now().UTC()
	return &domain.Campaign{
		ID:                   "550e8400-e29b-41d4-a716-446655440001",
		Name:                 "Summer Sale",
		Description:          "10% off everything",
		Type:                 domain.CampaignTypePercentage,
		Status:               domain.CampaignStatusDraft,
		DiscountValue:        1000,
		MinOrderAmount:       5000,
		MaxDiscountAmount:    10000,
		Code:                 "SUMMER10",
		MaxUsageCount:        100,
		CurrentUsageCount:    0,
		IsStackable:          false,
		Priority:             1,
		StartDate:            now.Add(24 * time.Hour),
		EndDate:              now.Add(30 * 24 * time.Hour),
		ApplicableCategories: []string{},
		ApplicableProducts:   []string{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// validCreateCampaignJSON returns a valid JSON payload for CreateCampaign.
func validCreateCampaignJSON() []byte {
	now := time.Now().UTC()
	req := CreateCampaignRequest{
		Name:          "Summer Sale",
		Description:   "10% off everything",
		Type:          "percentage",
		DiscountValue: 1000,
		MinOrderAmount: 5000,
		MaxDiscountAmount: 10000,
		Code:          "SUMMER10",
		MaxUsageCount: 100,
		IsStackable:   false,
		Priority:      1,
		StartDate:     now.Add(24 * time.Hour).Format(time.RFC3339),
		EndDate:       now.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}
	b, _ := json.Marshal(req)
	return b
}

// ============================================================================
// POST /api/v1/campaigns - CreateCampaign
// ============================================================================

func TestCreateCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(validCreateCampaignJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestCreateCampaign_InvalidJSON(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader([]byte(`{invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestCreateCampaign_ValidationError_MissingName(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	reqBody := CreateCampaignRequest{
		// Name intentionally omitted
		Type:          "percentage",
		DiscountValue: 1000,
		StartDate:     now.Add(24 * time.Hour).Format(time.RFC3339),
		EndDate:       now.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestCreateCampaign_InvalidDateFormat(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	reqBody := CreateCampaignRequest{
		Name:          "Summer Sale",
		Type:          "percentage",
		DiscountValue: 1000,
		StartDate:     "2025-01-01",  // Not RFC3339
		EndDate:       "2025-12-31",  // Not RFC3339
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "start_date must be in RFC3339 format")
}

func TestCreateCampaign_InvalidEndDateFormat(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	reqBody := CreateCampaignRequest{
		Name:          "Summer Sale",
		Type:          "percentage",
		DiscountValue: 1000,
		StartDate:     now.Add(24 * time.Hour).Format(time.RFC3339),
		EndDate:       "not-a-date",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "end_date must be in RFC3339 format")
}

func TestCreateCampaign_EndDateBeforeStartDate(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	reqBody := CreateCampaignRequest{
		Name:          "Summer Sale",
		Type:          "percentage",
		DiscountValue: 1000,
		StartDate:     now.Add(30 * 24 * time.Hour).Format(time.RFC3339),
		EndDate:       now.Add(1 * 24 * time.Hour).Format(time.RFC3339),
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "end_date must be after start_date")
}

func TestCreateCampaign_InvalidCodePattern(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	reqBody := CreateCampaignRequest{
		Name:          "Summer Sale",
		Type:          "percentage",
		DiscountValue: 1000,
		Code:          "invalid lowercase code",  // lowercase not allowed
		StartDate:     now.Add(24 * time.Hour).Format(time.RFC3339),
		EndDate:       now.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "code must be 2-50 uppercase alphanumeric characters or hyphens")
}

func TestCreateCampaign_ServiceError(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Campaign")).
		Return(apperrors.ErrAlreadyExists)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", bytes.NewReader(validCreateCampaignJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// The service wraps the error: "create campaign: resource already exists"
	// httputil.WriteError maps ErrAlreadyExists to 409.
	assert.Equal(t, http.StatusConflict, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "ALREADY_EXISTS", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/campaigns - ListCampaigns
// ============================================================================

func TestListCampaigns_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaigns := []domain.Campaign{*sampleCampaign()}
	expectedFilter := repository.CampaignFilter{Page: 1, PerPage: 20}
	repo.On("List", mock.Anything, expectedFilter).Return(campaigns, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	listResp := decodeListResponse(t, rec)
	assert.Equal(t, 1, listResp.TotalCount)
	assert.Equal(t, 1, listResp.Page)
	assert.Equal(t, 20, listResp.PerPage)
	assert.Equal(t, 1, listResp.TotalPages)
	assert.False(t, listResp.HasNext)
	repo.AssertExpectations(t)
}

func TestListCampaigns_WithPagination(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaigns := []domain.Campaign{*sampleCampaign()}
	expectedFilter := repository.CampaignFilter{Page: 2, PerPage: 10}
	repo.On("List", mock.Anything, expectedFilter).Return(campaigns, 25, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?page=2&per_page=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	listResp := decodeListResponse(t, rec)
	assert.Equal(t, 25, listResp.TotalCount)
	assert.Equal(t, 2, listResp.Page)
	assert.Equal(t, 10, listResp.PerPage)
	assert.Equal(t, 3, listResp.TotalPages)
	assert.True(t, listResp.HasNext)
	repo.AssertExpectations(t)
}

func TestListCampaigns_FilterByStatus(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaigns := []domain.Campaign{*sampleCampaign()}
	status := "active"
	expectedFilter := repository.CampaignFilter{Page: 1, PerPage: 20, Status: &status}
	repo.On("List", mock.Anything, expectedFilter).Return(campaigns, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?status=active", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListCampaigns_FilterByType(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaigns := []domain.Campaign{*sampleCampaign()}
	campaignType := "percentage"
	expectedFilter := repository.CampaignFilter{Page: 1, PerPage: 20, Type: &campaignType}
	repo.On("List", mock.Anything, expectedFilter).Return(campaigns, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?type=percentage", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	repo.AssertExpectations(t)
}

func TestListCampaigns_InvalidStatus(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?status=bogus", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "status must be one of")
}

func TestListCampaigns_InvalidPage(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?page=0", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListCampaigns_InvalidPerPage(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns?per_page=999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "per_page must be a valid integer between 1 and 100")
}

// ============================================================================
// GET /api/v1/campaigns/{id} - GetCampaign
// ============================================================================

func TestGetCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaign := sampleCampaign()
	repo.On("GetByID", mock.Anything, campaign.ID).Return(campaign, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+campaign.ID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetCampaign_InvalidUUID(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetCampaign_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, id).Return(nil, apperrors.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/campaigns/"+id, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// PUT /api/v1/campaigns/{id} - UpdateCampaign
// ============================================================================

func TestUpdateCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaign := sampleCampaign()
	repo.On("GetByID", mock.Anything, campaign.ID).Return(campaign, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	newName := "Updated Sale"
	updateReq := UpdateCampaignRequest{
		Name: &newName,
	}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+campaign.ID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestUpdateCampaign_InvalidJSON(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440001"

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+id, bytes.NewReader([]byte(`{bad json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestUpdateCampaign_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, id).Return(nil, apperrors.ErrNotFound)

	newName := "Updated Sale"
	updateReq := UpdateCampaignRequest{
		Name: &newName,
	}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestUpdateCampaign_InvalidUUID(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	updateReq := UpdateCampaignRequest{}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/not-a-uuid", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestUpdateCampaign_InvalidStartDateFormat(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440001"
	badDate := "2025-01-01"
	updateReq := UpdateCampaignRequest{
		StartDate: &badDate,
	}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "start_date must be in RFC3339 format")
}

func TestUpdateCampaign_InvalidEndDateFormat(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440001"
	badDate := "not-a-date"
	updateReq := UpdateCampaignRequest{
		EndDate: &badDate,
	}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "end_date must be in RFC3339 format")
}

func TestUpdateCampaign_EndDateBeforeStartDate(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440001"
	now := time.Now().UTC()
	startDate := now.Add(30 * 24 * time.Hour).Format(time.RFC3339)
	endDate := now.Add(1 * 24 * time.Hour).Format(time.RFC3339)
	updateReq := UpdateCampaignRequest{
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "end_date must be after start_date")
}

func TestUpdateCampaign_InvalidCodePattern(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440001"
	badCode := "lowercase-code!"
	updateReq := UpdateCampaignRequest{
		Code: &badCode,
	}
	b, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/campaigns/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "code must be 2-50 uppercase alphanumeric characters or hyphens")
}

// ============================================================================
// POST /api/v1/campaigns/{id}/deactivate - DeactivateCampaign
// ============================================================================

func TestDeactivateCampaign_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	campaign := sampleCampaign()
	campaign.Status = domain.CampaignStatusActive
	repo.On("GetByID", mock.Anything, campaign.ID).Return(campaign, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Campaign")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+campaign.ID+"/deactivate", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestDeactivateCampaign_InvalidUUID(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bad-id/deactivate", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestDeactivateCampaign_NotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	id := "550e8400-e29b-41d4-a716-446655440099"
	repo.On("GetByID", mock.Anything, id).Return(nil, apperrors.ErrNotFound)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+id+"/deactivate", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// POST /api/v1/coupons/validate - ValidateCoupon
// ============================================================================

func TestValidateCoupon_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	campaign := sampleCampaign()
	campaign.Status = domain.CampaignStatusActive
	campaign.StartDate = now.Add(-24 * time.Hour)
	campaign.EndDate = now.Add(30 * 24 * time.Hour)

	repo.On("GetByCode", mock.Anything, "SUMMER10").Return(campaign, nil)

	validateReq := ValidateCouponRequest{
		Code:        "SUMMER10",
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "550e8400-e29b-41d4-a716-446655440010",
	}
	b, _ := json.Marshal(validateReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestValidateCoupon_InvalidJSON(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestValidateCoupon_ValidationError_MissingCode(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	validateReq := ValidateCouponRequest{
		// Code is missing
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "550e8400-e29b-41d4-a716-446655440010",
	}
	b, _ := json.Marshal(validateReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestValidateCoupon_ValidationError_InvalidCurrency(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	validateReq := ValidateCouponRequest{
		Code:        "SUMMER10",
		OrderAmount: 10000,
		Currency:    "TOOLONG",  // must be len=3
		UserID:      "550e8400-e29b-41d4-a716-446655440010",
	}
	b, _ := json.Marshal(validateReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestValidateCoupon_ValidationError_InvalidUserID(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	validateReq := ValidateCouponRequest{
		Code:        "SUMMER10",
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "not-a-uuid",  // must be uuid
	}
	b, _ := json.Marshal(validateReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

// ============================================================================
// POST /api/v1/coupons/apply - ApplyCoupon
// ============================================================================

func TestApplyCoupon_Success(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	campaign := sampleCampaign()
	campaign.Status = domain.CampaignStatusActive
	campaign.StartDate = now.Add(-24 * time.Hour)
	campaign.EndDate = now.Add(30 * 24 * time.Hour)

	userID := "550e8400-e29b-41d4-a716-446655440010"
	orderID := "550e8400-e29b-41d4-a716-446655440020"

	// ValidateCoupon calls GetByCode. ApplyCoupon calls ValidateCoupon then GetByCode again.
	repo.On("GetByCode", mock.Anything, "SUMMER10").Return(campaign, nil)
	repo.On("IncrementUsage", mock.Anything, campaign.ID).Return(true, nil)
	repo.On("RecordUsage", mock.Anything, mock.AnythingOfType("*domain.CampaignUsage")).Return(nil)

	applyReq := ApplyCouponRequest{
		Code:        "SUMMER10",
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      userID,
		OrderID:     orderID,
	}
	b, _ := json.Marshal(applyReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/apply", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestApplyCoupon_InvalidJSON(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/apply", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestApplyCoupon_ValidationError(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	// Missing required fields
	applyReq := ApplyCouponRequest{
		Code: "SUMMER10",
		// OrderAmount, Currency, UserID, OrderID all missing
	}
	b, _ := json.Marshal(applyReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/apply", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestApplyCoupon_ServiceError_CouponNotFound(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	// ValidateCoupon calls GetByCode; if it returns ErrNotFound, validation returns
	// {Valid: false, Message: "coupon not found"} -- which is then returned as success
	// with valid=false. The ApplyCoupon handler then returns an InvalidInput error
	// because the coupon is invalid.
	repo.On("GetByCode", mock.Anything, "NONEXIST").Return(nil, apperrors.ErrNotFound)

	applyReq := ApplyCouponRequest{
		Code:        "NONEXIST",
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "550e8400-e29b-41d4-a716-446655440010",
		OrderID:     "550e8400-e29b-41d4-a716-446655440020",
	}
	b, _ := json.Marshal(applyReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/apply", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// ApplyCoupon validates first; coupon not found means validation.Valid=false
	// which causes service to return InvalidInput error => 400
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestApplyCoupon_ServiceError_UsageLimitReached(t *testing.T) {
	repo := new(mockCampaignRepository)
	handler := testCampaignHandler(repo)
	router := setupCampaignRouter(handler)

	now := time.Now().UTC()
	campaign := sampleCampaign()
	campaign.Status = domain.CampaignStatusActive
	campaign.StartDate = now.Add(-24 * time.Hour)
	campaign.EndDate = now.Add(30 * 24 * time.Hour)

	repo.On("GetByCode", mock.Anything, "SUMMER10").Return(campaign, nil)
	repo.On("IncrementUsage", mock.Anything, campaign.ID).Return(false, nil)

	applyReq := ApplyCouponRequest{
		Code:        "SUMMER10",
		OrderAmount: 10000,
		Currency:    "USD",
		UserID:      "550e8400-e29b-41d4-a716-446655440010",
		OrderID:     "550e8400-e29b-41d4-a716-446655440020",
	}
	b, _ := json.Marshal(applyReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/apply", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "coupon usage limit reached")
	repo.AssertExpectations(t)
}
