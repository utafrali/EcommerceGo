package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
	"github.com/utafrali/EcommerceGo/services/media/internal/event"
	"github.com/utafrali/EcommerceGo/services/media/internal/repository"
	"github.com/utafrali/EcommerceGo/services/media/internal/service"
	"github.com/utafrali/EcommerceGo/services/media/internal/storage"
)

// listResponse is a type alias for the standardized PaginatedResponse.
type listResponse = httputil.PaginatedResponse[domain.MediaFile]

// Ensure interfaces are satisfied at compile time.
var _ repository.MediaRepository = (*mockMediaRepository)(nil)
var _ storage.Storage = (*mockStorage)(nil)

// --- Mock MediaRepository ---

type mockMediaRepository struct {
	mock.Mock
}

func (m *mockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *mockMediaRepository) GetByID(ctx context.Context, id string) (*domain.MediaFile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MediaFile), args.Error(1)
}

func (m *mockMediaRepository) ListByOwner(ctx context.Context, ownerID, ownerType string, offset, limit int) ([]domain.MediaFile, int, error) {
	args := m.Called(ctx, ownerID, ownerType, offset, limit)
	return args.Get(0).([]domain.MediaFile), args.Int(1), args.Error(2)
}

func (m *mockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *mockMediaRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Mock Storage ---

type mockStorage struct {
	mock.Mock
}

func (m *mockStorage) Upload(ctx context.Context, input *storage.UploadInput) (*storage.UploadResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockStorage) GetURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

// --- Test Helpers ---

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEventProducer() *event.Producer {
	logger := testLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func newTestService(repo *mockMediaRepository, store *mockStorage) *service.MediaService {
	return service.NewMediaService(repo, store, testEventProducer(), testLogger())
}

func newTestHandler(repo *mockMediaRepository, store *mockStorage) *MediaHandler {
	svc := newTestService(repo, store)
	return NewMediaHandler(svc, testLogger())
}

func setupMediaRouter(handler *MediaHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/media", func(r chi.Router) {
		r.Post("/", handler.UploadMedia)
		r.Get("/{id}", handler.GetMedia)
		r.Get("/owner/{ownerType}/{ownerId}", handler.ListMediaByOwner)
		r.Put("/{id}", handler.UpdateMediaMetadata)
		r.Delete("/{id}", handler.DeleteMedia)
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

const testMediaID = "550e8400-e29b-41d4-a716-446655440001"
const testOwnerID = "550e8400-e29b-41d4-a716-446655440002"

func sampleMediaFile() *domain.MediaFile {
	now := time.Now().UTC()
	return &domain.MediaFile{
		ID:           testMediaID,
		OwnerID:      testOwnerID,
		OwnerType:    "product",
		FileName:     "product/" + testOwnerID + "/" + testMediaID,
		OriginalName: "test.jpg",
		ContentType:  "image/jpeg",
		Size:         1024,
		URL:          "http://localhost/media/product/" + testOwnerID + "/" + testMediaID,
		AltText:      "Test image",
		SortOrder:    0,
		Metadata:     make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// createMultipartUpload builds a multipart form body with the given file data and fields.
// It sets the file part Content-Type to image/jpeg so the service accepts it.
func createMultipartUpload(fileName string, fileData []byte, fields map[string]string) (*bytes.Buffer, string) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	if fileName != "" {
		// Use CreatePart with explicit Content-Type instead of CreateFormFile
		// (which defaults to application/octet-stream).
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
		h.Set("Content-Type", "image/jpeg")
		part, _ := writer.CreatePart(h)
		_, _ = part.Write(fileData)
	}

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}

	_ = writer.Close()
	return body, writer.FormDataContentType()
}

// ============================================================================
// POST /api/v1/media - Upload Media
// ============================================================================

func TestUploadMedia_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	// Mock storage upload.
	store.On("Upload", mock.Anything, mock.AnythingOfType("*storage.UploadInput")).
		Return(&storage.UploadResult{
			Key: "product/" + testOwnerID + "/some-id",
			URL: "http://localhost/media/product/" + testOwnerID + "/some-id",
		}, nil)

	// Mock repository create.
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	body, contentType := createMultipartUpload("test.jpg", []byte("fake image data"), map[string]string{
		"owner_id":   testOwnerID,
		"owner_type": "product",
		"alt_text":   "A test image",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	store.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestUploadMedia_MissingFile(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	// Create a multipart form WITHOUT the "file" field.
	body, contentType := createMultipartUpload("", nil, map[string]string{
		"owner_id":   testOwnerID,
		"owner_type": "product",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "file is required")
}

func TestUploadMedia_InvalidOwnerType(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	body, contentType := createMultipartUpload("test.jpg", []byte("fake image data"), map[string]string{
		"owner_id":   testOwnerID,
		"owner_type": "invalid_type",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// The service should reject the invalid owner type.
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "not allowed")
}

func TestUploadMedia_ServiceError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	// Mock storage to return an error.
	store.On("Upload", mock.Anything, mock.AnythingOfType("*storage.UploadInput")).
		Return(nil, fmt.Errorf("storage unavailable"))

	body, contentType := createMultipartUpload("test.jpg", []byte("fake image data"), map[string]string{
		"owner_id":   testOwnerID,
		"owner_type": "product",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	store.AssertExpectations(t)
}

func TestUploadMedia_MissingOwnerID(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	body, contentType := createMultipartUpload("test.jpg", []byte("fake image data"), map[string]string{
		"owner_type": "product",
		// owner_id intentionally omitted
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "owner id is required")
}

func TestUploadMedia_MissingOwnerType(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	body, contentType := createMultipartUpload("test.jpg", []byte("fake image data"), map[string]string{
		"owner_id": testOwnerID,
		// owner_type intentionally omitted
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "owner type is required")
}

// ============================================================================
// GET /api/v1/media/{id} - Get Media
// ============================================================================

func TestGetMedia_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/"+testMediaID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	// Verify the returned data contains the media ID.
	dataMap, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, testMediaID, dataMap["id"])
	repo.AssertExpectations(t)
}

func TestGetMedia_InvalidUUID(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/not-a-valid-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetMedia_NotFound(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	repo.On("GetByID", mock.Anything, testMediaID).
		Return(nil, apperrors.NotFound("media", testMediaID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/"+testMediaID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/media/owner/{ownerType}/{ownerId} - List by Owner
// ============================================================================

func TestListMediaByOwner_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media1 := sampleMediaFile()
	media2 := *sampleMediaFile()
	media2.ID = "550e8400-e29b-41d4-a716-446655440099"
	mediaList := []domain.MediaFile{*media1, media2}

	// Default pagination: page=1, perPage=20, offset=0.
	repo.On("ListByOwner", mock.Anything, testOwnerID, "product", 0, 20).
		Return(mediaList, 2, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/owner/product/"+testOwnerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Decode the list response envelope.
	var listResp listResponse
	err := json.NewDecoder(rec.Body).Decode(&listResp)
	require.NoError(t, err)
	assert.Equal(t, 2, listResp.TotalCount)
	assert.Equal(t, 1, listResp.Page)
	assert.Equal(t, 20, listResp.PerPage)
	assert.Equal(t, 1, listResp.TotalPages)
	repo.AssertExpectations(t)
}

func TestListMediaByOwner_WithPagination(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	mediaList := []domain.MediaFile{*sampleMediaFile()}

	// page=2, per_page=5 => offset = (2-1)*5 = 5.
	repo.On("ListByOwner", mock.Anything, testOwnerID, "product", 5, 5).
		Return(mediaList, 8, nil)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/owner/product/%s?page=2&per_page=5", testOwnerID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var listResp listResponse
	err := json.NewDecoder(rec.Body).Decode(&listResp)
	require.NoError(t, err)
	assert.Equal(t, 8, listResp.TotalCount)
	assert.Equal(t, 2, listResp.Page)
	assert.Equal(t, 5, listResp.PerPage)
	assert.Equal(t, 2, listResp.TotalPages) // 8/5 = 1.6 => 2
	repo.AssertExpectations(t)
}

func TestListMediaByOwner_InvalidPage(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/owner/product/%s?page=abc", testOwnerID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "page")
}

func TestListMediaByOwner_InvalidPerPage(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/owner/product/%s?per_page=200", testOwnerID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "per_page")
}

func TestListMediaByOwner_NegativePage(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/owner/product/%s?page=-1", testOwnerID), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestListMediaByOwner_ServiceError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	repo.On("ListByOwner", mock.Anything, testOwnerID, "product", 0, 20).
		Return([]domain.MediaFile(nil), 0, fmt.Errorf("database connection lost"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/owner/product/"+testOwnerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// PUT /api/v1/media/{id} - Update Media Metadata
// ============================================================================

func TestUpdateMediaMetadata_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	altText := "Updated alt text"
	sortOrder := 5
	reqBody := UpdateMediaRequest{
		AltText:   &altText,
		SortOrder: &sortOrder,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+testMediaID, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_InvalidJSON(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+testMediaID, bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestUpdateMediaMetadata_InvalidUUID(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	body := `{"alt_text": "test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/bad-uuid", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestUpdateMediaMetadata_NotFound(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	repo.On("GetByID", mock.Anything, testMediaID).
		Return(nil, apperrors.NotFound("media", testMediaID))

	altText := "test"
	reqBody := UpdateMediaRequest{AltText: &altText}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+testMediaID, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_EmptyBody(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	// Empty JSON object: no fields to update.
	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+testMediaID, bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

// ============================================================================
// DELETE /api/v1/media/{id} - Delete Media
// ============================================================================

func TestDeleteMedia_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	store.On("Delete", mock.Anything, media.FileName).Return(nil)
	repo.On("Delete", mock.Anything, testMediaID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/"+testMediaID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	// Verify deleted ID is returned.
	dataMap, ok := resp.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, testMediaID, dataMap["id"])
	assert.Equal(t, "deleted", dataMap["status"])

	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestDeleteMedia_InvalidUUID(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestDeleteMedia_NotFound(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	repo.On("GetByID", mock.Anything, testMediaID).
		Return(nil, apperrors.NotFound("media", testMediaID))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/"+testMediaID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

func TestDeleteMedia_StorageDeleteError_StillSucceeds(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	// Storage delete fails, but the delete should still succeed (logs error and continues).
	store.On("Delete", mock.Anything, media.FileName).Return(fmt.Errorf("storage error"))
	repo.On("Delete", mock.Anything, testMediaID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/"+testMediaID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestDeleteMedia_RepoDeleteError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	store.On("Delete", mock.Anything, media.FileName).Return(nil)
	repo.On("Delete", mock.Anything, testMediaID).Return(fmt.Errorf("database error"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/media/"+testMediaID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

// ============================================================================
// Multipart form edge cases
// ============================================================================

func TestUploadMedia_InvalidMultipartForm(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	// Send a request with Content-Type multipart/form-data but invalid body.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", bytes.NewReader([]byte("not a valid multipart form")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=nonexistent")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

// ============================================================================
// Table-driven tests for pagination parameters
// ============================================================================

func TestListMediaByOwner_PaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		expectCode int
		errCode    string
	}{
		{
			name:       "page=0 (invalid, must be >= 1)",
			query:      "page=0",
			expectCode: http.StatusBadRequest,
			errCode:    "INVALID_PARAMETER",
		},
		{
			name:       "per_page=0 (invalid, must be >= 1)",
			query:      "per_page=0",
			expectCode: http.StatusBadRequest,
			errCode:    "INVALID_PARAMETER",
		},
		{
			name:       "per_page=101 (exceeds max 100)",
			query:      "per_page=101",
			expectCode: http.StatusBadRequest,
			errCode:    "INVALID_PARAMETER",
		},
		{
			name:       "page=abc (not a number)",
			query:      "page=abc",
			expectCode: http.StatusBadRequest,
			errCode:    "INVALID_PARAMETER",
		},
		{
			name:       "per_page=xyz (not a number)",
			query:      "per_page=xyz",
			expectCode: http.StatusBadRequest,
			errCode:    "INVALID_PARAMETER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockMediaRepository)
			store := new(mockStorage)
			handler := newTestHandler(repo, store)
			router := setupMediaRouter(handler)

			url := fmt.Sprintf("/api/v1/media/owner/product/%s?%s", testOwnerID, tt.query)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectCode, rec.Code)
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tt.errCode, resp.Error.Code)
		})
	}
}

// ============================================================================
// Table-driven tests for upload validation via service layer
// ============================================================================

func TestUploadMedia_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]string
		fileData []byte
		fileName string
		errMsg   string
	}{
		{
			name:     "missing owner_id",
			fields:   map[string]string{"owner_type": "product"},
			fileData: []byte("fake"),
			fileName: "test.jpg",
			errMsg:   "owner id is required",
		},
		{
			name:     "missing owner_type",
			fields:   map[string]string{"owner_id": testOwnerID},
			fileData: []byte("fake"),
			fileName: "test.jpg",
			errMsg:   "owner type is required",
		},
		{
			name:     "invalid owner_type",
			fields:   map[string]string{"owner_id": testOwnerID, "owner_type": "bogus"},
			fileData: []byte("fake"),
			fileName: "test.jpg",
			errMsg:   "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockMediaRepository)
			store := new(mockStorage)
			handler := newTestHandler(repo, store)
			router := setupMediaRouter(handler)

			body, contentType := createMultipartUpload(tt.fileName, tt.fileData, tt.fields)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/media", body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Contains(t, resp.Error.Message, tt.errMsg)
		})
	}
}

// ============================================================================
// Update with partial fields (only alt_text, only sort_order)
// ============================================================================

func TestUpdateMediaMetadata_OnlyAltText(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	altText := "Only alt text updated"
	reqBody := UpdateMediaRequest{AltText: &altText}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+testMediaID, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_OnlySortOrder(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	media := sampleMediaFile()
	repo.On("GetByID", mock.Anything, testMediaID).Return(media, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	sortOrder := 10
	reqBody := UpdateMediaRequest{SortOrder: &sortOrder}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/media/"+testMediaID, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	repo.AssertExpectations(t)
}

// ============================================================================
// ListMediaByOwner - empty result set
// ============================================================================

func TestListMediaByOwner_EmptyResult(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	handler := newTestHandler(repo, store)
	router := setupMediaRouter(handler)

	repo.On("ListByOwner", mock.Anything, testOwnerID, "product", 0, 20).
		Return([]domain.MediaFile{}, 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/media/owner/product/"+testOwnerID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var listResp listResponse
	err := json.NewDecoder(rec.Body).Decode(&listResp)
	require.NoError(t, err)
	assert.Equal(t, 0, listResp.TotalCount)
	assert.Equal(t, 1, listResp.Page)
	repo.AssertExpectations(t)
}
