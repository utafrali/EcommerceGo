package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/user/internal/domain"
)

func newAddressTestFixture(t *testing.T) (*AddressRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	repo := NewAddressRepository(mock)
	return repo, mock
}

func sampleAddress() *domain.Address {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &domain.Address{
		ID:           "addr-1",
		UserID:       "u-1234",
		Label:        "Home",
		FirstName:    "Alice",
		LastName:     "Smith",
		AddressLine1: "123 Main St",
		AddressLine2: "Apt 4",
		City:         "Springfield",
		State:        "IL",
		PostalCode:   "62701",
		CountryCode:  "US",
		Phone:        "+1234567890",
		IsDefault:    true,
		CreatedAt:    now,
	}
}

func addressColumns() []string {
	return []string{
		"id", "user_id", "label", "first_name", "last_name",
		"address_line1", "address_line2", "city", "state",
		"postal_code", "country_code", "phone", "is_default", "created_at",
	}
}

func addressRow(a *domain.Address) *pgxmock.Rows {
	return pgxmock.NewRows(addressColumns()).AddRow(
		a.ID, a.UserID, a.Label, a.FirstName, a.LastName,
		a.AddressLine1, a.AddressLine2, a.City, a.State,
		a.PostalCode, a.CountryCode, a.Phone, a.IsDefault, a.CreatedAt,
	)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAddressRepository_Create_Success(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	a := sampleAddress()

	mock.ExpectExec("INSERT INTO addresses").
		WithArgs(
			a.ID, a.UserID, a.Label, a.FirstName, a.LastName,
			a.AddressLine1, a.AddressLine2, a.City, a.State,
			a.PostalCode, a.CountryCode, a.Phone, a.IsDefault, a.CreatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), a)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestAddressRepository_GetByID_Success(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	a := sampleAddress()

	mock.ExpectQuery("SELECT .+ FROM addresses WHERE id =").
		WithArgs(a.ID).
		WillReturnRows(addressRow(a))

	got, err := repo.GetByID(context.Background(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)
	assert.Equal(t, a.UserID, got.UserID)
	assert.Equal(t, a.Label, got.Label)
	assert.Equal(t, a.City, got.City)
	assert.Equal(t, a.IsDefault, got.IsDefault)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddressRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM addresses WHERE id =").
		WithArgs("missing-addr").
		WillReturnError(pgx.ErrNoRows)

	got, err := repo.GetByID(context.Background(), "missing-addr")
	assert.Nil(t, got)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListByUserID
// ---------------------------------------------------------------------------

func TestAddressRepository_ListByUserID_Success(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	a1 := sampleAddress()
	a2 := &domain.Address{
		ID:           "addr-2",
		UserID:       "u-1234",
		Label:        "Work",
		FirstName:    "Alice",
		LastName:     "Smith",
		AddressLine1: "456 Oak Ave",
		AddressLine2: "",
		City:         "Chicago",
		State:        "IL",
		PostalCode:   "60601",
		CountryCode:  "US",
		Phone:        "+1987654321",
		IsDefault:    false,
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}

	rows := pgxmock.NewRows(addressColumns()).
		AddRow(
			a1.ID, a1.UserID, a1.Label, a1.FirstName, a1.LastName,
			a1.AddressLine1, a1.AddressLine2, a1.City, a1.State,
			a1.PostalCode, a1.CountryCode, a1.Phone, a1.IsDefault, a1.CreatedAt,
		).
		AddRow(
			a2.ID, a2.UserID, a2.Label, a2.FirstName, a2.LastName,
			a2.AddressLine1, a2.AddressLine2, a2.City, a2.State,
			a2.PostalCode, a2.CountryCode, a2.Phone, a2.IsDefault, a2.CreatedAt,
		)

	mock.ExpectQuery("SELECT .+ FROM addresses WHERE user_id =").
		WithArgs("u-1234").
		WillReturnRows(rows)

	got, err := repo.ListByUserID(context.Background(), "u-1234")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "addr-1", got[0].ID)
	assert.Equal(t, "addr-2", got[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddressRepository_ListByUserID_Empty(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	rows := pgxmock.NewRows(addressColumns())

	mock.ExpectQuery("SELECT .+ FROM addresses WHERE user_id =").
		WithArgs("u-no-addrs").
		WillReturnRows(rows)

	got, err := repo.ListByUserID(context.Background(), "u-no-addrs")
	require.NoError(t, err)
	assert.NotNil(t, got, "should return empty slice, not nil")
	assert.Len(t, got, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestAddressRepository_Update_Success(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	a := sampleAddress()

	mock.ExpectExec("UPDATE addresses").
		WithArgs(
			a.Label, a.FirstName, a.LastName,
			a.AddressLine1, a.AddressLine2,
			a.City, a.State, a.PostalCode, a.CountryCode, a.Phone,
			a.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), a)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddressRepository_Update_NotFound(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	a := sampleAddress()
	a.ID = "nonexistent"

	mock.ExpectExec("UPDATE addresses").
		WithArgs(
			a.Label, a.FirstName, a.LastName,
			a.AddressLine1, a.AddressLine2,
			a.City, a.State, a.PostalCode, a.CountryCode, a.Phone,
			a.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), a)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAddressRepository_Delete_Success(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM addresses WHERE id =").
		WithArgs("addr-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(context.Background(), "addr-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddressRepository_Delete_NotFound(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM addresses WHERE id =").
		WithArgs("missing-addr").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.Delete(context.Background(), "missing-addr")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// SetDefault
// ---------------------------------------------------------------------------

func TestAddressRepository_SetDefault_Success(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	userID := "u-1234"
	addressID := "addr-2"

	mock.ExpectBegin()

	// Step 1: SELECT FOR UPDATE (lock)
	mock.ExpectExec("SELECT id FROM addresses WHERE user_id =").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	// Step 2: Unset existing default
	mock.ExpectExec("UPDATE addresses SET is_default = false WHERE user_id =").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Step 3: Set new default
	mock.ExpectExec("UPDATE addresses SET is_default = true WHERE id =").
		WithArgs(addressID, userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectCommit()

	err := repo.SetDefault(context.Background(), userID, addressID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddressRepository_SetDefault_NotFound(t *testing.T) {
	repo, mock := newAddressTestFixture(t)
	defer mock.Close()

	userID := "u-1234"
	addressID := "addr-missing"

	mock.ExpectBegin()

	// Step 1: SELECT FOR UPDATE
	mock.ExpectExec("SELECT id FROM addresses WHERE user_id =").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))

	// Step 2: Unset existing default
	mock.ExpectExec("UPDATE addresses SET is_default = false WHERE user_id =").
		WithArgs(userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	// Step 3: Set new default — returns 0 rows affected
	mock.ExpectExec("UPDATE addresses SET is_default = true WHERE id =").
		WithArgs(addressID, userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	mock.ExpectRollback()

	err := repo.SetDefault(context.Background(), userID, addressID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound), "expected ErrNotFound, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
