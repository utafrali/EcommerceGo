package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

func newUserTestFixture(t *testing.T) (*UserRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	repo := NewUserRepository(mock)
	return repo, mock
}

func sampleUser() *domain.User {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &domain.User{
		ID:            "u-1234",
		Email:         "alice@example.com",
		PasswordHash:  "hash-abc",
		FirstName:     "Alice",
		LastName:      "Smith",
		Phone:         "+1234567890",
		Role:          "customer",
		IsActive:      true,
		EmailVerified: false,
		OAuthProvider: "",
		OAuthID:       "",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// userColumns returns the 13 column names scanned by scanUser and inserted by Create.
func userColumns() []string {
	return []string{
		"id", "email", "password_hash", "first_name", "last_name",
		"phone", "role", "is_active", "email_verified",
		"oauth_provider", "oauth_id", "created_at", "updated_at",
	}
}

func userRow(u *domain.User) *pgxmock.Rows {
	return pgxmock.NewRows(userColumns()).AddRow(
		u.ID, u.Email, u.PasswordHash, u.FirstName, u.LastName,
		u.Phone, u.Role, u.IsActive, u.EmailVerified,
		u.OAuthProvider, u.OAuthID, u.CreatedAt, u.UpdatedAt,
	)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestUserRepository_Create_Success(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()

	mock.ExpectExec("INSERT INTO users").
		WithArgs(
			u.ID, u.Email, u.PasswordHash, u.FirstName, u.LastName,
			u.Phone, u.Role, u.IsActive, u.EmailVerified,
			u.OAuthProvider, u.OAuthID, u.CreatedAt, u.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()

	mock.ExpectExec("INSERT INTO users").
		WithArgs(
			u.ID, u.Email, u.PasswordHash, u.FirstName, u.LastName,
			u.Phone, u.Role, u.IsActive, u.EmailVerified,
			u.OAuthProvider, u.OAuthID, u.CreatedAt, u.UpdatedAt,
		).
		WillReturnError(fmt.Errorf("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.Create(context.Background(), u)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrAlreadyExists), "expected ErrAlreadyExists, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestUserRepository_GetByID_Success(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()

	mock.ExpectQuery("SELECT .+ FROM users WHERE id =").
		WithArgs(u.ID).
		WillReturnRows(userRow(u))

	got, err := repo.GetByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Email, got.Email)
	assert.Equal(t, u.FirstName, got.FirstName)
	assert.Equal(t, u.Role, got.Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM users WHERE id =").
		WithArgs("missing-id").
		WillReturnError(pgx.ErrNoRows)

	got, err := repo.GetByID(context.Background(), "missing-id")
	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound), "expected ErrNotFound, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByEmail
// ---------------------------------------------------------------------------

func TestUserRepository_GetByEmail_Success(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()

	mock.ExpectQuery("SELECT .+ FROM users WHERE email =").
		WithArgs(u.Email).
		WillReturnRows(userRow(u))

	got, err := repo.GetByEmail(context.Background(), u.Email)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Email, got.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM users WHERE email =").
		WithArgs("nobody@example.com").
		WillReturnError(pgx.ErrNoRows)

	got, err := repo.GetByEmail(context.Background(), "nobody@example.com")
	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserRepository_Update_Success(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()

	// Update sets UpdatedAt to time.Now().UTC(), so we use AnyArg for that column.
	mock.ExpectExec("UPDATE users").
		WithArgs(
			u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Phone,
			u.Role, u.IsActive, u.EmailVerified,
			u.OAuthProvider, u.OAuthID,
			pgxmock.AnyArg(), // updated_at
			u.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()

	mock.ExpectExec("UPDATE users").
		WithArgs(
			u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Phone,
			u.Role, u.IsActive, u.EmailVerified,
			u.OAuthProvider, u.OAuthID,
			pgxmock.AnyArg(),
			u.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), u)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound), "expected ErrNotFound, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update_DuplicateEmail(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	u := sampleUser()
	u.Email = "taken@example.com"

	mock.ExpectExec("UPDATE users").
		WithArgs(
			u.Email, u.PasswordHash, u.FirstName, u.LastName, u.Phone,
			u.Role, u.IsActive, u.EmailVerified,
			u.OAuthProvider, u.OAuthID,
			pgxmock.AnyArg(),
			u.ID,
		).
		WillReturnError(fmt.Errorf("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.Update(context.Background(), u)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrAlreadyExists), "expected ErrAlreadyExists, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestUserRepository_Delete_Success(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM users WHERE id =").
		WithArgs("u-1234").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(context.Background(), "u-1234")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	repo, mock := newUserTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM users WHERE id =").
		WithArgs("missing-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.Delete(context.Background(), "missing-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound), "expected ErrNotFound, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
