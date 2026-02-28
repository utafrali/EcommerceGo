package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/user/internal/auth"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
	"github.com/utafrali/EcommerceGo/services/user/internal/event"
	"github.com/utafrali/EcommerceGo/services/user/internal/repository"
)

// bcryptCost is the cost factor for bcrypt password hashing.
const bcryptCost = 12

// minPasswordLength is the minimum password length required.
const minPasswordLength = 8

// UserService implements the business logic for user and auth operations.
type UserService struct {
	userRepo         repository.UserRepository
	addressRepo      repository.AddressRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtManager       *auth.JWTManager
	producer         *event.Producer
	logger           *slog.Logger
}

// NewUserService creates a new user service.
func NewUserService(
	userRepo repository.UserRepository,
	addressRepo repository.AddressRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtManager *auth.JWTManager,
	producer *event.Producer,
	logger *slog.Logger,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		addressRepo:      addressRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
		producer:         producer,
		logger:           logger,
	}
}

// --- Auth Input/Output types ---

// RegisterInput holds the parameters for registering a new user.
type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// LoginInput holds the parameters for user login.
type LoginInput struct {
	Email    string
	Password string
}

// UpdateProfileInput holds the parameters for updating a user's profile.
type UpdateProfileInput struct {
	FirstName *string
	LastName  *string
	Phone     *string
}

// CreateAddressInput holds the parameters for creating a new address.
type CreateAddressInput struct {
	Label        string
	FirstName    string
	LastName     string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	CountryCode  string
	Phone        string
	IsDefault    bool
}

// UpdateAddressInput holds the parameters for updating an address.
type UpdateAddressInput struct {
	Label        *string
	FirstName    *string
	LastName     *string
	AddressLine1 *string
	AddressLine2 *string
	City         *string
	State        *string
	PostalCode   *string
	CountryCode  *string
	Phone        *string
}

// --- Auth Operations ---

// Register creates a new user account, hashes the password, and returns tokens.
func (s *UserService) Register(ctx context.Context, input RegisterInput) (*domain.User, *domain.TokenPair, error) {
	if input.Email == "" {
		return nil, nil, apperrors.InvalidInput("email is required")
	}
	if input.FirstName == "" {
		return nil, nil, apperrors.InvalidInput("first name is required")
	}
	if input.LastName == "" {
		return nil, nil, apperrors.InvalidInput("last name is required")
	}
	if err := validatePassword(input.Password); err != nil {
		return nil, nil, err
	}

	// Hash password with bcrypt.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New().String(),
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Role:         domain.RoleCustomer,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Generate tokens.
	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	// Publish registration event (non-blocking on failure).
	if err := s.producer.PublishUserRegistered(ctx, user); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish user.registered event",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "user registered",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return user, tokens, nil
}

// Login authenticates a user with email and password, returning tokens.
func (s *UserService) Login(ctx context.Context, input LoginInput) (*domain.User, *domain.TokenPair, error) {
	if input.Email == "" {
		return nil, nil, apperrors.InvalidInput("email is required")
	}
	if input.Password == "" {
		return nil, nil, apperrors.InvalidInput("password is required")
	}

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, apperrors.Unauthorized("invalid email or password")
	}

	if !user.IsActive {
		return nil, nil, apperrors.Unauthorized("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, apperrors.Unauthorized("invalid email or password")
	}

	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("generate tokens: %w", err)
	}

	s.logger.InfoContext(ctx, "user logged in",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return user, tokens, nil
}

// RefreshToken validates a refresh token and generates a new token pair.
func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	if refreshToken == "" {
		return nil, apperrors.InvalidInput("refresh token is required")
	}

	// Validate the JWT.
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, apperrors.Unauthorized("invalid or expired refresh token")
	}

	// Look up the stored token hash.
	tokenHash := hashToken(refreshToken)
	storedToken, err := s.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, apperrors.Unauthorized("refresh token not found")
	}

	// Check if the token has been revoked.
	if storedToken.RevokedAt != nil {
		return nil, apperrors.Unauthorized("refresh token has been revoked")
	}

	// Check if the token has expired.
	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return nil, apperrors.Unauthorized("refresh token has expired")
	}

	// Revoke the old refresh token.
	if err := s.refreshTokenRepo.Revoke(ctx, tokenHash); err != nil {
		s.logger.ErrorContext(ctx, "failed to revoke old refresh token",
			slog.String("user_id", claims.UserID),
			slog.String("error", err.Error()),
		)
	}

	// Fetch user to get current email/role for the new access token.
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for token refresh: %w", err)
	}

	tokens, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}

	s.logger.InfoContext(ctx, "tokens refreshed",
		slog.String("user_id", user.ID),
	)

	return tokens, nil
}

// ForgotPassword initiates a password reset by publishing a reset event.
func (s *UserService) ForgotPassword(ctx context.Context, email string) error {
	if email == "" {
		return apperrors.InvalidInput("email is required")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Do not reveal whether the email exists.
		s.logger.InfoContext(ctx, "password reset requested for unknown email",
			slog.String("email", email),
		)
		return nil
	}

	// Publish password reset event (notification service will send the email).
	if err := s.producer.PublishUserPasswordReset(ctx, user.ID, user.Email); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish user.password_reset event",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "password reset requested",
		slog.String("user_id", user.ID),
		slog.String("email", user.Email),
	)

	return nil
}

// ResetPassword resets a user's password using a token.
// In a full implementation, the token would be validated against a stored reset token.
// For now, this validates the JWT refresh token as a placeholder mechanism.
func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token == "" {
		return apperrors.InvalidInput("reset token is required")
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	// Validate the token (using refresh token validation as placeholder).
	claims, err := s.jwtManager.ValidateRefreshToken(token)
	if err != nil {
		return apperrors.Unauthorized("invalid or expired reset token")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("get user for password reset: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	// Revoke all existing refresh tokens for this user.
	if err := s.refreshTokenRepo.RevokeByUserID(ctx, user.ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to revoke refresh tokens after password reset",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "password reset completed",
		slog.String("user_id", user.ID),
	)

	return nil
}

// ChangePassword allows an authenticated user to change their password.
func (s *UserService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if currentPassword == "" {
		return apperrors.InvalidInput("current password is required")
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	if currentPassword == newPassword {
		return apperrors.InvalidInput("new password must be different from current password")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user for password change: %w", err)
	}

	// Verify current password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return apperrors.Unauthorized("current password is incorrect")
	}

	// Hash new password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	// Revoke all existing refresh tokens for this user (force re-login for security).
	if err := s.refreshTokenRepo.RevokeByUserID(ctx, user.ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to revoke refresh tokens after password change",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "password changed",
		slog.String("user_id", user.ID),
	)

	return nil
}

// --- Profile Operations ---

// GetProfile retrieves a user by their ID.
func (s *UserService) GetProfile(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user profile: %w", err)
	}
	return user, nil
}

// UpdateProfile updates a user's profile fields.
func (s *UserService) UpdateProfile(ctx context.Context, userID string, input UpdateProfileInput) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user for update: %w", err)
	}

	if input.FirstName != nil {
		if *input.FirstName == "" {
			return nil, apperrors.InvalidInput("first name must not be empty")
		}
		user.FirstName = *input.FirstName
	}

	if input.LastName != nil {
		if *input.LastName == "" {
			return nil, apperrors.InvalidInput("last name must not be empty")
		}
		user.LastName = *input.LastName
	}

	if input.Phone != nil {
		user.Phone = *input.Phone
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	// Publish user updated event (non-blocking on failure).
	if err := s.producer.PublishUserUpdated(ctx, user); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish user.updated event",
			slog.String("user_id", user.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "user profile updated",
		slog.String("user_id", user.ID),
	)

	return user, nil
}

// --- Address Operations ---

// CreateAddress creates a new address for the user.
func (s *UserService) CreateAddress(ctx context.Context, userID string, input *CreateAddressInput) (*domain.Address, error) {
	if input.FirstName == "" {
		return nil, apperrors.InvalidInput("first name is required")
	}
	if input.LastName == "" {
		return nil, apperrors.InvalidInput("last name is required")
	}
	if input.AddressLine1 == "" {
		return nil, apperrors.InvalidInput("address line 1 is required")
	}
	if input.City == "" {
		return nil, apperrors.InvalidInput("city is required")
	}
	if input.PostalCode == "" {
		return nil, apperrors.InvalidInput("postal code is required")
	}
	if input.CountryCode == "" || len(input.CountryCode) != 2 {
		return nil, apperrors.InvalidInput("country code must be a 2-letter ISO code")
	}

	now := time.Now().UTC()
	address := &domain.Address{
		ID:           uuid.New().String(),
		UserID:       userID,
		Label:        input.Label,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		AddressLine1: input.AddressLine1,
		AddressLine2: input.AddressLine2,
		City:         input.City,
		State:        input.State,
		PostalCode:   input.PostalCode,
		CountryCode:  input.CountryCode,
		Phone:        input.Phone,
		IsDefault:    input.IsDefault,
		CreatedAt:    now,
	}

	if err := s.addressRepo.Create(ctx, address); err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}

	// If this is the default, update the default setting.
	if input.IsDefault {
		if err := s.addressRepo.SetDefault(ctx, userID, address.ID); err != nil {
			s.logger.ErrorContext(ctx, "failed to set default address",
				slog.String("address_id", address.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	s.logger.InfoContext(ctx, "address created",
		slog.String("user_id", userID),
		slog.String("address_id", address.ID),
	)

	return address, nil
}

// ListAddresses returns all addresses for the given user.
func (s *UserService) ListAddresses(ctx context.Context, userID string) ([]domain.Address, error) {
	addresses, err := s.addressRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	return addresses, nil
}

// UpdateAddress updates an existing address.
func (s *UserService) UpdateAddress(ctx context.Context, userID, addressID string, input *UpdateAddressInput) (*domain.Address, error) {
	address, err := s.addressRepo.GetByID(ctx, addressID)
	if err != nil {
		return nil, fmt.Errorf("get address for update: %w", err)
	}

	// Verify ownership.
	if address.UserID != userID {
		return nil, apperrors.NotFound("address", addressID)
	}

	if input.Label != nil {
		address.Label = *input.Label
	}
	if input.FirstName != nil {
		if *input.FirstName == "" {
			return nil, apperrors.InvalidInput("first name must not be empty")
		}
		address.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		if *input.LastName == "" {
			return nil, apperrors.InvalidInput("last name must not be empty")
		}
		address.LastName = *input.LastName
	}
	if input.AddressLine1 != nil {
		if *input.AddressLine1 == "" {
			return nil, apperrors.InvalidInput("address line 1 must not be empty")
		}
		address.AddressLine1 = *input.AddressLine1
	}
	if input.AddressLine2 != nil {
		address.AddressLine2 = *input.AddressLine2
	}
	if input.City != nil {
		if *input.City == "" {
			return nil, apperrors.InvalidInput("city must not be empty")
		}
		address.City = *input.City
	}
	if input.State != nil {
		address.State = *input.State
	}
	if input.PostalCode != nil {
		if *input.PostalCode == "" {
			return nil, apperrors.InvalidInput("postal code must not be empty")
		}
		address.PostalCode = *input.PostalCode
	}
	if input.CountryCode != nil {
		if len(*input.CountryCode) != 2 {
			return nil, apperrors.InvalidInput("country code must be a 2-letter ISO code")
		}
		address.CountryCode = *input.CountryCode
	}
	if input.Phone != nil {
		address.Phone = *input.Phone
	}

	if err := s.addressRepo.Update(ctx, address); err != nil {
		return nil, fmt.Errorf("update address: %w", err)
	}

	s.logger.InfoContext(ctx, "address updated",
		slog.String("user_id", userID),
		slog.String("address_id", addressID),
	)

	return address, nil
}

// DeleteAddress removes an address for the user.
func (s *UserService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	address, err := s.addressRepo.GetByID(ctx, addressID)
	if err != nil {
		return fmt.Errorf("get address for delete: %w", err)
	}

	// Verify ownership.
	if address.UserID != userID {
		return apperrors.NotFound("address", addressID)
	}

	if err := s.addressRepo.Delete(ctx, addressID); err != nil {
		return fmt.Errorf("delete address: %w", err)
	}

	s.logger.InfoContext(ctx, "address deleted",
		slog.String("user_id", userID),
		slog.String("address_id", addressID),
	)

	return nil
}

// SetDefaultAddress marks the specified address as the user's default.
func (s *UserService) SetDefaultAddress(ctx context.Context, userID, addressID string) error {
	address, err := s.addressRepo.GetByID(ctx, addressID)
	if err != nil {
		return fmt.Errorf("get address for set default: %w", err)
	}

	// Verify ownership.
	if address.UserID != userID {
		return apperrors.NotFound("address", addressID)
	}

	if err := s.addressRepo.SetDefault(ctx, userID, addressID); err != nil {
		return fmt.Errorf("set default address: %w", err)
	}

	s.logger.InfoContext(ctx, "default address updated",
		slog.String("user_id", userID),
		slog.String("address_id", addressID),
	)

	return nil
}

// --- Helpers ---

// generateTokenPair creates an access/refresh token pair and stores the refresh token hash.
func (s *UserService) generateTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store the refresh token hash in the database.
	tokenHash := hashToken(refreshToken)
	refreshClaims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token for expiry: %w", err)
	}

	if err := s.refreshTokenRepo.Create(ctx, user.ID, tokenHash, refreshClaims.ExpiresAt.Time); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// hashToken returns the SHA256 hex digest of the given token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// validatePassword checks that the password meets minimum complexity requirements.
func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return apperrors.InvalidInput(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return apperrors.InvalidInput("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}
