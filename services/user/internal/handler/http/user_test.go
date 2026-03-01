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
	"github.com/utafrali/EcommerceGo/pkg/middleware"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
	"github.com/utafrali/EcommerceGo/services/user/internal/event"
	"github.com/utafrali/EcommerceGo/services/user/internal/service"
)

// ============================================================================
// Mock Repositories
// ============================================================================

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockAddressRepo struct {
	mock.Mock
}

func (m *mockAddressRepo) Create(ctx context.Context, address *domain.Address) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

func (m *mockAddressRepo) GetByID(ctx context.Context, id string) (*domain.Address, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Address), args.Error(1)
}

func (m *mockAddressRepo) ListByUserID(ctx context.Context, userID string) ([]domain.Address, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Address), args.Error(1)
}

func (m *mockAddressRepo) Update(ctx context.Context, address *domain.Address) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

func (m *mockAddressRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAddressRepo) SetDefault(ctx context.Context, userID, addressID string) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

type mockRefreshTokenRepo struct {
	mock.Mock
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, tokenHash, expiresAt)
	return args.Error(0)
}

func (m *mockRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *mockRefreshTokenRepo) RevokeByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

// ============================================================================
// Test Helpers
// ============================================================================

func userTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func userTestEventProducer() *event.Producer {
	logger := userTestLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:19092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func userTestService(userRepo *mockUserRepo, addrRepo *mockAddressRepo) *service.UserService {
	logger := userTestLogger()
	producer := userTestEventProducer()
	refreshRepo := new(mockRefreshTokenRepo)
	return service.NewUserService(userRepo, addrRepo, refreshRepo, nil, producer, logger)
}

func userTestHandler(userRepo *mockUserRepo, addrRepo *mockAddressRepo) *UserHandler {
	svc := userTestService(userRepo, addrRepo)
	return NewUserHandler(svc, userTestLogger())
}

// fakeTokenValidator returns a middleware.TokenValidator that always succeeds
// and injects the given userID into the request context.
func fakeTokenValidator(userID string) middleware.TokenValidator {
	return func(token string) (*middleware.Claims, error) {
		return &middleware.Claims{UserID: userID, Email: "test@example.com", Role: "customer"}, nil
	}
}

// setupUserRouter creates a chi router that mirrors the production routes for
// user profile and address endpoints, using a fake token validator for auth.
func setupUserRouter(handler *UserHandler, userID string) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Use(middleware.Auth(fakeTokenValidator(userID)))
		r.Get("/me", handler.GetProfile)
		r.Put("/me", handler.UpdateProfile)
		r.Get("/me/addresses", handler.ListAddresses)
		r.Post("/me/addresses", handler.CreateAddress)
		r.Put("/me/addresses/{id}", handler.UpdateAddress)
		r.Delete("/me/addresses/{id}", handler.DeleteAddress)
	})
	return r
}

// setupUserRouterNoAuth creates a chi router WITHOUT auth middleware so
// unauthenticated requests can be tested.
func setupUserRouterNoAuth(handler *UserHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Get("/me", handler.GetProfile)
		r.Put("/me", handler.UpdateProfile)
		r.Get("/me/addresses", handler.ListAddresses)
		r.Post("/me/addresses", handler.CreateAddress)
		r.Put("/me/addresses/{id}", handler.UpdateAddress)
		r.Delete("/me/addresses/{id}", handler.DeleteAddress)
	})
	return r
}

func decodeUserResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.Response {
	t.Helper()
	var resp httputil.Response
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	return resp
}

const testUserID = "550e8400-e29b-41d4-a716-446655440001"
const testAddressID = "550e8400-e29b-41d4-a716-446655440002"

func sampleUser() *domain.User {
	now := time.Now().UTC()
	return &domain.User{
		ID:            testUserID,
		Email:         "test@example.com",
		PasswordHash:  "$2a$12$hashedpassword",
		FirstName:     "John",
		LastName:      "Doe",
		Phone:         "+1234567890",
		Role:          "customer",
		IsActive:      true,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func sampleAddress() *domain.Address {
	return &domain.Address{
		ID:           testAddressID,
		UserID:       testUserID,
		Label:        "Home",
		FirstName:    "John",
		LastName:     "Doe",
		AddressLine1: "123 Main St",
		AddressLine2: "Apt 4",
		City:         "New York",
		State:        "NY",
		PostalCode:   "10001",
		CountryCode:  "US",
		Phone:        "+1234567890",
		IsDefault:    true,
		CreatedAt:    time.Now().UTC(),
	}
}

// ============================================================================
// GetProfile Tests
// ============================================================================

func TestGetProfile_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	user := sampleUser()
	userRepo.On("GetByID", mock.Anything, testUserID).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	userRepo.AssertExpectations(t)
}

func TestGetProfile_Unauthorized(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouterNoAuth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "UNAUTHORIZED", resp.Error.Code)
}

func TestGetProfile_NotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	userRepo.On("GetByID", mock.Anything, testUserID).Return(nil, apperrors.NotFound("user", testUserID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "NOT_FOUND", resp.Error.Code)
}

func TestGetProfile_InternalError(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	userRepo.On("GetByID", mock.Anything, testUserID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ============================================================================
// UpdateProfile Tests
// ============================================================================

func TestUpdateProfile_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	user := sampleUser()
	userRepo.On("GetByID", mock.Anything, testUserID).Return(user, nil)
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)

	firstName := "Jane"
	body := UpdateProfileRequest{FirstName: &firstName}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	userRepo.AssertExpectations(t)
}

func TestUpdateProfile_Unauthorized(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouterNoAuth(handler)

	body := `{"first_name":"Jane"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateProfile_InvalidJSON(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	userRepo.On("GetByID", mock.Anything, testUserID).Return(nil, apperrors.NotFound("user", testUserID))

	firstName := "Jane"
	body := UpdateProfileRequest{FirstName: &firstName}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateProfile_ValidationError(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	// Phone field with validation tag max=20 — exceed it
	longPhone := "012345678901234567890" // 21 chars
	body := UpdateProfileRequest{Phone: &longPhone}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ============================================================================
// ListAddresses Tests
// ============================================================================

func TestListAddresses_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addresses := []domain.Address{*sampleAddress()}
	addrRepo.On("ListByUserID", mock.Anything, testUserID).Return(addresses, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/addresses", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	addrRepo.AssertExpectations(t)
}

func TestListAddresses_Empty(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("ListByUserID", mock.Anything, testUserID).Return([]domain.Address{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/addresses", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
}

func TestListAddresses_Unauthorized(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouterNoAuth(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/addresses", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListAddresses_InternalError(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("ListByUserID", mock.Anything, testUserID).Return(nil, assert.AnError)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/addresses", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ============================================================================
// CreateAddress Tests
// ============================================================================

func TestCreateAddress_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Address")).Return(nil)

	body := CreateAddressRequest{
		Label:        "Home",
		FirstName:    "John",
		LastName:     "Doe",
		AddressLine1: "123 Main St",
		City:         "New York",
		PostalCode:   "10001",
		CountryCode:  "US",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	addrRepo.AssertExpectations(t)
}

func TestCreateAddress_WithDefault(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Address")).Return(nil)
	addrRepo.On("SetDefault", mock.Anything, testUserID, mock.AnythingOfType("string")).Return(nil)

	body := CreateAddressRequest{
		FirstName:    "John",
		LastName:     "Doe",
		AddressLine1: "123 Main St",
		City:         "New York",
		PostalCode:   "10001",
		CountryCode:  "US",
		IsDefault:    true,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	addrRepo.AssertExpectations(t)
}

func TestCreateAddress_Unauthorized(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouterNoAuth(handler)

	body := `{"first_name":"John","last_name":"Doe","address_line1":"123 Main St","city":"NY","postal_code":"10001","country_code":"US"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateAddress_InvalidJSON(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
}

func TestCreateAddress_ValidationError_MissingRequired(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	// Missing required fields: first_name, last_name, address_line1, city, postal_code, country_code
	body := CreateAddressRequest{Label: "Home"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateAddress_ValidationError_InvalidCountryCode(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	body := CreateAddressRequest{
		FirstName:    "John",
		LastName:     "Doe",
		AddressLine1: "123 Main St",
		City:         "New York",
		PostalCode:   "10001",
		CountryCode:  "USA", // must be len=2
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateAddress_InternalError(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Address")).Return(assert.AnError)

	body := CreateAddressRequest{
		FirstName:    "John",
		LastName:     "Doe",
		AddressLine1: "123 Main St",
		City:         "New York",
		PostalCode:   "10001",
		CountryCode:  "US",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/addresses", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ============================================================================
// UpdateAddress Tests
// ============================================================================

func TestUpdateAddress_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addr := sampleAddress()
	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(addr, nil)
	addrRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Address")).Return(nil)

	newCity := "Los Angeles"
	body := UpdateAddressRequest{City: &newCity}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+testAddressID, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	addrRepo.AssertExpectations(t)
}

func TestUpdateAddress_Unauthorized(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouterNoAuth(handler)

	body := `{"city":"LA"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+testAddressID, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateAddress_InvalidUUID(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	body := `{"city":"LA"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/not-a-uuid", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestUpdateAddress_NotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(nil, apperrors.NotFound("address", testAddressID))

	newCity := "LA"
	body := UpdateAddressRequest{City: &newCity}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+testAddressID, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpdateAddress_InvalidJSON(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+testAddressID, bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateAddress_OwnershipMismatch(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	// Address belongs to a different user
	addr := sampleAddress()
	addr.UserID = "different-user-id"
	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(addr, nil)

	newCity := "LA"
	body := UpdateAddressRequest{City: &newCity}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/addresses/"+testAddressID, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ============================================================================
// DeleteAddress Tests
// ============================================================================

func TestDeleteAddress_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addr := sampleAddress()
	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(addr, nil)
	addrRepo.On("Delete", mock.Anything, testAddressID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+testAddressID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
	addrRepo.AssertExpectations(t)
}

func TestDeleteAddress_Unauthorized(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouterNoAuth(handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+testAddressID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteAddress_InvalidUUID(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	resp := decodeUserResponse(t, rec)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "INVALID_PARAMETER", resp.Error.Code)
}

func TestDeleteAddress_NotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(nil, apperrors.NotFound("address", testAddressID))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+testAddressID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteAddress_OwnershipMismatch(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addr := sampleAddress()
	addr.UserID = "other-user"
	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(addr, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+testAddressID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteAddress_InternalError(t *testing.T) {
	userRepo := new(mockUserRepo)
	addrRepo := new(mockAddressRepo)
	handler := userTestHandler(userRepo, addrRepo)
	router := setupUserRouter(handler, testUserID)

	addr := sampleAddress()
	addrRepo.On("GetByID", mock.Anything, testAddressID).Return(addr, nil)
	addrRepo.On("Delete", mock.Anything, testAddressID).Return(assert.AnError)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/addresses/"+testAddressID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
