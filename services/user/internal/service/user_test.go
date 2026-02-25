package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/user/internal/auth"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
	"github.com/utafrali/EcommerceGo/services/user/internal/event"
)

// --- Mock User Repository ---

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Mock Address Repository ---

type mockAddressRepository struct {
	mock.Mock
}

func (m *mockAddressRepository) Create(ctx context.Context, address *domain.Address) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

func (m *mockAddressRepository) GetByID(ctx context.Context, id string) (*domain.Address, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Address), args.Error(1)
}

func (m *mockAddressRepository) ListByUserID(ctx context.Context, userID string) ([]domain.Address, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Address), args.Error(1)
}

func (m *mockAddressRepository) Update(ctx context.Context, address *domain.Address) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

func (m *mockAddressRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockAddressRepository) SetDefault(ctx context.Context, userID, addressID string) error {
	args := m.Called(ctx, userID, addressID)
	return args.Error(0)
}

// --- Mock Refresh Token Repository ---

type mockRefreshTokenRepository struct {
	mock.Mock
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, tokenHash, expiresAt)
	return args.Error(0)
}

func (m *mockRefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *mockRefreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockRefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	args := m.Called(ctx, tokenHash)
	return args.Error(0)
}

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret-key-for-testing", 15*time.Minute, 7*24*time.Hour)
}

func newTestEventProducer() *event.Producer {
	logger := newTestLogger()
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	return event.NewProducer(kafkaProducer, logger)
}

func newTestService(
	userRepo *mockUserRepository,
	addressRepo *mockAddressRepository,
	refreshTokenRepo *mockRefreshTokenRepository,
) *UserService {
	logger := newTestLogger()
	jwtManager := newTestJWTManager()
	producer := newTestEventProducer()
	return NewUserService(userRepo, addressRepo, refreshTokenRepo, jwtManager, producer, logger)
}

func strPtr(s string) *string {
	return &s
}

// hashForTest creates a bcrypt hash with cost 4 for fast tests.
func hashForTest(password string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		panic(err)
	}
	return string(h)
}

// --- Register Tests ---

func TestRegister_Success(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	refreshTokenRepo.On("Create", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

	input := RegisterInput{
		Email:     "john@example.com",
		Password:  "SecurePass123",
		FirstName: "John",
		LastName:  "Doe",
	}

	user, tokens, err := svc.Register(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, domain.RoleCustomer, user.Role)
	assert.True(t, user.IsActive)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)

	userRepo.AssertExpectations(t)
	refreshTokenRepo.AssertExpectations(t)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	userRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).
		Return(apperrors.AlreadyExists("user", "email", "john@example.com"))

	input := RegisterInput{
		Email:     "john@example.com",
		Password:  "SecurePass123",
		FirstName: "John",
		LastName:  "Doe",
	}

	user, tokens, err := svc.Register(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)

	userRepo.AssertExpectations(t)
}

func TestRegister_WeakPassword_TooShort(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	input := RegisterInput{
		Email:     "john@example.com",
		Password:  "Ab1",
		FirstName: "John",
		LastName:  "Doe",
	}

	user, tokens, err := svc.Register(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestRegister_WeakPassword_NoUppercase(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	input := RegisterInput{
		Email:     "john@example.com",
		Password:  "securepass123",
		FirstName: "John",
		LastName:  "Doe",
	}

	user, tokens, err := svc.Register(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestRegister_WeakPassword_NoDigit(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	input := RegisterInput{
		Email:     "john@example.com",
		Password:  "SecurePassword",
		FirstName: "John",
		LastName:  "Doe",
	}

	user, tokens, err := svc.Register(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestRegister_MissingEmail(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	input := RegisterInput{
		Email:     "",
		Password:  "SecurePass123",
		FirstName: "John",
		LastName:  "Doe",
	}

	user, tokens, err := svc.Register(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

// --- Login Tests ---

func TestLogin_Success(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	existing := &domain.User{
		ID:           "user-123",
		Email:        "john@example.com",
		PasswordHash: hashForTest("SecurePass123"),
		FirstName:    "John",
		LastName:     "Doe",
		Role:         domain.RoleCustomer,
		IsActive:     true,
	}

	userRepo.On("GetByEmail", ctx, "john@example.com").Return(existing, nil)
	refreshTokenRepo.On("Create", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

	input := LoginInput{
		Email:    "john@example.com",
		Password: "SecurePass123",
	}

	user, tokens, err := svc.Login(ctx, input)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "john@example.com", user.Email)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)

	userRepo.AssertExpectations(t)
	refreshTokenRepo.AssertExpectations(t)
}

func TestLogin_WrongPassword(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	existing := &domain.User{
		ID:           "user-123",
		Email:        "john@example.com",
		PasswordHash: hashForTest("CorrectPass123"),
		Role:         domain.RoleCustomer,
		IsActive:     true,
	}

	userRepo.On("GetByEmail", ctx, "john@example.com").Return(existing, nil)

	input := LoginInput{
		Email:    "john@example.com",
		Password: "WrongPass456",
	}

	user, tokens, err := svc.Login(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)

	userRepo.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	userRepo.On("GetByEmail", ctx, "notfound@example.com").Return(nil, apperrors.ErrNotFound)

	input := LoginInput{
		Email:    "notfound@example.com",
		Password: "AnyPass123",
	}

	user, tokens, err := svc.Login(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)

	userRepo.AssertExpectations(t)
}

func TestLogin_InactiveUser(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	existing := &domain.User{
		ID:           "user-123",
		Email:        "john@example.com",
		PasswordHash: hashForTest("SecurePass123"),
		Role:         domain.RoleCustomer,
		IsActive:     false,
	}

	userRepo.On("GetByEmail", ctx, "john@example.com").Return(existing, nil)

	input := LoginInput{
		Email:    "john@example.com",
		Password: "SecurePass123",
	}

	user, tokens, err := svc.Login(ctx, input)

	assert.Nil(t, user)
	assert.Nil(t, tokens)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrUnauthorized)

	userRepo.AssertExpectations(t)
}

// --- GetProfile Tests ---

func TestGetProfile_Success(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	expected := &domain.User{
		ID:        "user-123",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      domain.RoleCustomer,
		IsActive:  true,
	}

	userRepo.On("GetByID", ctx, "user-123").Return(expected, nil)

	user, err := svc.GetProfile(ctx, "user-123")

	require.NoError(t, err)
	assert.Equal(t, expected, user)

	userRepo.AssertExpectations(t)
}

func TestGetProfile_NotFound(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	userRepo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	user, err := svc.GetProfile(ctx, "nonexistent")

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	userRepo.AssertExpectations(t)
}

// --- UpdateProfile Tests ---

func TestUpdateProfile_Success(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	existing := &domain.User{
		ID:        "user-123",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      domain.RoleCustomer,
		IsActive:  true,
	}

	userRepo.On("GetByID", ctx, "user-123").Return(existing, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	input := UpdateProfileInput{
		FirstName: strPtr("Jonathan"),
		Phone:     strPtr("+1234567890"),
	}

	user, err := svc.UpdateProfile(ctx, "user-123", input)

	require.NoError(t, err)
	assert.Equal(t, "Jonathan", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, "+1234567890", user.Phone)

	userRepo.AssertExpectations(t)
}

func TestUpdateProfile_EmptyFirstName(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	existing := &domain.User{
		ID:        "user-123",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	userRepo.On("GetByID", ctx, "user-123").Return(existing, nil)

	emptyName := ""
	input := UpdateProfileInput{
		FirstName: &emptyName,
	}

	user, err := svc.UpdateProfile(ctx, "user-123", input)

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)

	userRepo.AssertExpectations(t)
}

func TestUpdateProfile_EmailConflict(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	existing := &domain.User{
		ID:        "user-123",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	userRepo.On("GetByID", ctx, "user-123").Return(existing, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*domain.User")).
		Return(apperrors.AlreadyExists("user", "email", "john@example.com"))

	input := UpdateProfileInput{
		FirstName: strPtr("Jonathan"),
	}

	user, err := svc.UpdateProfile(ctx, "user-123", input)

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)

	userRepo.AssertExpectations(t)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	userRepo := new(mockUserRepository)
	addressRepo := new(mockAddressRepository)
	refreshTokenRepo := new(mockRefreshTokenRepository)
	svc := newTestService(userRepo, addressRepo, refreshTokenRepo)
	ctx := context.Background()

	userRepo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	input := UpdateProfileInput{
		FirstName: strPtr("New Name"),
	}

	user, err := svc.UpdateProfile(ctx, "nonexistent", input)

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	userRepo.AssertExpectations(t)
}

// --- JWT Tests ---

func TestJWT_GenerateAndValidateAccessToken(t *testing.T) {
	jwtManager := newTestJWTManager()

	token, err := jwtManager.GenerateAccessToken("user-123", "john@example.com", "customer")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtManager.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "john@example.com", claims.Email)
	assert.Equal(t, "customer", claims.Role)
}

func TestJWT_GenerateAndValidateRefreshToken(t *testing.T) {
	jwtManager := newTestJWTManager()

	token, err := jwtManager.GenerateRefreshToken("user-123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwtManager.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
}

func TestJWT_InvalidToken(t *testing.T) {
	jwtManager := newTestJWTManager()

	claims, err := jwtManager.ValidateAccessToken("invalid-token")
	assert.Nil(t, claims)
	assert.Error(t, err)
}

func TestJWT_WrongSecret(t *testing.T) {
	jwtManager1 := auth.NewJWTManager("secret-1", 15*time.Minute, 7*24*time.Hour)
	jwtManager2 := auth.NewJWTManager("secret-2", 15*time.Minute, 7*24*time.Hour)

	token, err := jwtManager1.GenerateAccessToken("user-123", "john@example.com", "customer")
	require.NoError(t, err)

	claims, err := jwtManager2.ValidateAccessToken(token)
	assert.Nil(t, claims)
	assert.Error(t, err)
}

// --- Password Validation Tests ---

func TestValidatePassword_Valid(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"standard", "SecurePass123"},
		{"with special chars", "P@ssw0rd!XY"},
		{"exactly 8 chars", "Abcdef1g"},
		{"long password", "VeryLongSecurePassword123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			assert.NoError(t, err)
		})
	}
}

func TestValidatePassword_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"too short", "Ab1"},
		{"no uppercase", "securepass123"},
		{"no lowercase", "SECUREPASS123"},
		{"no digit", "SecurePassword"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			assert.Error(t, err)
			assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
		})
	}
}

// --- Hash Token Test ---

func TestHashToken(t *testing.T) {
	token1 := "some-token-value"
	token2 := "different-token-value"

	hash1 := hashToken(token1)
	hash2 := hashToken(token2)

	assert.NotEmpty(t, hash1)
	assert.NotEmpty(t, hash2)
	assert.NotEqual(t, hash1, hash2)

	// Same input should produce same hash.
	assert.Equal(t, hash1, hashToken(token1))
}
