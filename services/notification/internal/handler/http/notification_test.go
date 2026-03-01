package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"context"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/pkg/httputil"
	"github.com/utafrali/EcommerceGo/services/notification/internal/domain"
	"github.com/utafrali/EcommerceGo/services/notification/internal/event"
	"github.com/utafrali/EcommerceGo/services/notification/internal/sender"
	"github.com/utafrali/EcommerceGo/services/notification/internal/service"
)

// listResponse mirrors httputil.PaginatedResponse for test decoding.
type listResponse = httputil.PaginatedResponse[domain.Notification]

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockNotificationRepo struct {
	mock.Mock
}

func (m *mockNotificationRepo) Create(ctx context.Context, notification *domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *mockNotificationRepo) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationRepo) Update(ctx context.Context, notification *domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *mockNotificationRepo) ListByUserID(ctx context.Context, userID string, offset, limit int) ([]domain.Notification, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]domain.Notification), args.Int(1), args.Error(2)
}

func (m *mockNotificationRepo) ListPending(ctx context.Context, limit int) ([]domain.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]domain.Notification), args.Error(1)
}

func (m *mockNotificationRepo) ListFailed(ctx context.Context, limit int) ([]domain.Notification, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]domain.Notification), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock sender
// ---------------------------------------------------------------------------

type mockSender struct {
	mock.Mock
}

func (m *mockSender) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockSender) Send(ctx context.Context, notification *domain.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// noopProducer creates an event.Producer with a nil Kafka client so Publish
// calls are no-ops.
func noopProducer() *event.Producer {
	return event.NewProducer(nil, testLogger())
}

// buildService creates a real NotificationService backed by mock dependencies.
func buildService(repo *mockNotificationRepo, senders map[string]sender.Sender) *service.NotificationService {
	return service.NewNotificationService(repo, senders, noopProducer(), testLogger())
}

// buildHandler creates a NotificationHandler backed by mock dependencies.
func buildHandler(repo *mockNotificationRepo, senders map[string]sender.Sender) *NotificationHandler {
	svc := buildService(repo, senders)
	return NewNotificationHandler(svc, testLogger())
}

// setupNotificationRouter mounts notification routes on a chi router,
// mirroring the production router layout.
func setupNotificationRouter(h *NotificationHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Use(ContentTypeJSON)
		r.Post("/", h.SendNotification)
		r.Get("/{id}", h.GetNotification)
		r.Get("/user/{userId}", h.ListNotificationsByUser)
		r.Put("/{id}/read", h.MarkAsRead)
		r.Post("/{id}/retry", h.RetryNotification)
	})
	return r
}

// decodeResponse reads the response body into an httputil.Response.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// decodeListResponse reads the response body into a listResponse.
func decodeListResponse(t *testing.T, rec *httptest.ResponseRecorder) listResponse {
	t.Helper()
	var resp listResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

// validUUID is a constant UUID used in tests.
const validUUID = "550e8400-e29b-41d4-a716-446655440001"

// sampleNotification returns a domain.Notification suitable for test assertions.
func sampleNotification() *domain.Notification {
	now := time.Now().UTC()
	return &domain.Notification{
		ID:         validUUID,
		UserID:     "550e8400-e29b-41d4-a716-446655440002",
		Type:       domain.NotificationTypeEmail,
		Channel:    domain.ChannelEmail,
		Subject:    "Test Subject",
		Body:       "Hello, this is a test notification.",
		Status:     domain.NotificationStatusPending,
		Priority:   domain.NotificationPriorityNormal,
		Metadata:   map[string]any{"key": "value"},
		RetryCount: 0,
		MaxRetries: domain.DefaultMaxRetries,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// validSendJSON returns a valid SendNotificationRequest as JSON bytes.
func validSendJSON() []byte {
	body := SendNotificationRequest{
		UserID:  "550e8400-e29b-41d4-a716-446655440002",
		Type:    "email",
		Channel: "email",
		Subject: "Test Subject",
		Body:    "Hello, this is a test notification.",
	}
	b, _ := json.Marshal(body)
	return b
}

// defaultSenders returns a map with a mock email sender that succeeds.
func defaultSenders() (map[string]sender.Sender, *mockSender) {
	ms := new(mockSender)
	ms.On("Name").Return("email")
	ms.On("Send", mock.Anything, mock.Anything).Return(nil)
	senders := map[string]sender.Sender{
		"email": ms,
	}
	return senders, ms
}

// ============================================================================
// POST /api/v1/notifications -- SendNotification
// ============================================================================

func TestSendNotification_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	senders, _ := defaultSenders()
	h := buildHandler(repo, senders)
	router := setupNotificationRouter(h)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader(validSendJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestSendNotification_InvalidJSON(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid request body")
}

func TestSendNotification_ValidationError_MissingFields(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	// Missing required fields: user_id, type, channel, body.
	body := SendNotificationRequest{
		Subject: "only subject provided",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestSendNotification_InvalidChannelType(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	body := SendNotificationRequest{
		UserID:  validUUID,
		Type:    "email",
		Channel: "email",
		Body:    "some body",
		// Invalid priority triggers validation error (oneof=low normal high urgent).
		Priority: "super_urgent",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	// The validator should catch the invalid priority.
	assert.Contains(t, resp.Error.Code, "VALIDATION_ERROR")
}

func TestSendNotification_InvalidType(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	// The "type" field has validate:"required,oneof=email sms push".
	// Passing "telegram" should fail validation.
	body := SendNotificationRequest{
		UserID:  validUUID,
		Type:    "telegram",
		Channel: "email",
		Body:    "some body",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Error.Code)
}

func TestSendNotification_ServiceError(t *testing.T) {
	repo := new(mockNotificationRepo)
	senders, _ := defaultSenders()
	h := buildHandler(repo, senders)
	router := setupNotificationRouter(h)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Notification")).
		Return(fmt.Errorf("database connection lost"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader(validSendJSON()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/notifications/{id} -- GetNotification
// ============================================================================

func TestGetNotification_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	notif := sampleNotification()
	repo.On("GetByID", mock.Anything, validUUID).Return(notif, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+validUUID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestGetNotification_InvalidUUID(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestGetNotification_NotFound(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	repo.On("GetByID", mock.Anything, validUUID).
		Return(nil, apperrors.NotFound("notification", validUUID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/"+validUUID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// GET /api/v1/notifications/user/{userId} -- ListNotificationsByUser
// ============================================================================

func TestListNotificationsByUser_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	userID := "550e8400-e29b-41d4-a716-446655440002"
	notifs := []domain.Notification{*sampleNotification()}

	// Default pagination: page=1, perPage=20, offset=0.
	repo.On("ListByUserID", mock.Anything, userID, 0, 20).Return(notifs, 1, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/user/"+userID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeListResponse(t, rec)
	assert.Equal(t, 1, resp.TotalCount)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PerPage)
	assert.False(t, resp.HasNext)
	repo.AssertExpectations(t)
}

func TestListNotificationsByUser_WithPagination(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	userID := "550e8400-e29b-41d4-a716-446655440002"
	notifs := []domain.Notification{*sampleNotification()}

	// page=2, per_page=5 => offset=(2-1)*5=5
	repo.On("ListByUserID", mock.Anything, userID, 5, 5).Return(notifs, 12, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/user/"+userID+"?page=2&per_page=5", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeListResponse(t, rec)
	assert.Equal(t, 12, resp.TotalCount)
	assert.Equal(t, 2, resp.Page)
	assert.Equal(t, 5, resp.PerPage)
	assert.Equal(t, 3, resp.TotalPages) // ceil(12/5)=3
	assert.True(t, resp.HasNext)        // page 2 < 3
	repo.AssertExpectations(t)
}

func TestListNotificationsByUser_InvalidUUID(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/user/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "invalid UUID")
}

func TestListNotificationsByUser_InvalidPage(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	userID := "550e8400-e29b-41d4-a716-446655440002"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/user/"+userID+"?page=-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "page")
}

func TestListNotificationsByUser_InvalidPerPage(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	userID := "550e8400-e29b-41d4-a716-446655440002"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/user/"+userID+"?per_page=999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "per_page")
}

func TestListNotificationsByUser_EmptyList(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	userID := "550e8400-e29b-41d4-a716-446655440002"

	// Return nil slice -- handler should normalize to empty array.
	repo.On("ListByUserID", mock.Anything, userID, 0, 20).Return([]domain.Notification(nil), 0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/user/"+userID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeListResponse(t, rec)
	assert.Equal(t, 0, resp.TotalCount)
	// Data should be an empty array, not null.
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

// ============================================================================
// PUT /api/v1/notifications/{id}/read -- MarkAsRead
// ============================================================================

func TestMarkAsRead_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	notif := sampleNotification()
	notif.Status = domain.NotificationStatusSent
	repo.On("GetByID", mock.Anything, validUUID).Return(notif, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/"+validUUID+"/read", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestMarkAsRead_InvalidUUID(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/bad-id/read", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestMarkAsRead_NotFound(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	repo.On("GetByID", mock.Anything, validUUID).
		Return(nil, apperrors.NotFound("notification", validUUID))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/"+validUUID+"/read", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// POST /api/v1/notifications/{id}/retry -- RetryNotification
// ============================================================================

func TestRetryNotification_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	senders, _ := defaultSenders()
	h := buildHandler(repo, senders)
	router := setupNotificationRouter(h)

	notif := sampleNotification()
	notif.Status = domain.NotificationStatusFailed
	notif.RetryCount = 1
	notif.MaxRetries = 3

	repo.On("GetByID", mock.Anything, validUUID).Return(notif, nil)
	// Update called twice: once to set pending+retry, once after send success.
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Notification")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validUUID+"/retry", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	repo.AssertExpectations(t)
}

func TestRetryNotification_InvalidUUID(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/xyz/retry", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestRetryNotification_ServiceError_NotFailed(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	// Notification is in "sent" status -- cannot be retried.
	notif := sampleNotification()
	notif.Status = domain.NotificationStatusSent

	repo.On("GetByID", mock.Anything, validUUID).Return(notif, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validUUID+"/retry", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "only failed notifications can be retried")
	repo.AssertExpectations(t)
}

func TestRetryNotification_MaxRetriesExceeded(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	notif := sampleNotification()
	notif.Status = domain.NotificationStatusFailed
	notif.RetryCount = 3
	notif.MaxRetries = 3

	repo.On("GetByID", mock.Anything, validUUID).Return(notif, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validUUID+"/retry", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "maximum retry")
	repo.AssertExpectations(t)
}

func TestRetryNotification_NotFound(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	repo.On("GetByID", mock.Anything, validUUID).
		Return(nil, apperrors.NotFound("notification", validUUID))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+validUUID+"/retry", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeResponse(t, rec)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
	repo.AssertExpectations(t)
}

// ============================================================================
// ContentTypeJSON middleware
// ============================================================================

func TestContentTypeJSON_RejectsNonJSON(t *testing.T) {
	repo := new(mockNotificationRepo)
	h := buildHandler(repo, nil)
	router := setupNotificationRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/", bytes.NewReader(validSendJSON()))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

// ============================================================================
// Table-driven: all endpoints that accept {id} reject invalid UUIDs
// ============================================================================

func TestAllIDEndpoints_RejectInvalidUUID(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/notifications/bad-uuid"},
		{http.MethodPut, "/api/v1/notifications/bad-uuid/read"},
		{http.MethodPost, "/api/v1/notifications/bad-uuid/retry"},
	}

	for _, ep := range endpoints {
		name := fmt.Sprintf("%s %s", ep.method, ep.path)
		t.Run(name, func(t *testing.T) {
			repo := new(mockNotificationRepo)
			h := buildHandler(repo, nil)
			router := setupNotificationRouter(h)

			req := httptest.NewRequest(ep.method, ep.path, nil)
			if ep.method == http.MethodPut || ep.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code, "expected 400 for invalid UUID")
			resp := decodeResponse(t, rec)
			require.NotNil(t, resp.Error)
			assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
		})
	}
}
