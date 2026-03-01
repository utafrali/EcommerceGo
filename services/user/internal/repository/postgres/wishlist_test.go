package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
)

func newWishlistTestFixture(t *testing.T) (*WishlistRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	repo := NewWishlistRepository(mock)
	return repo, mock
}

// ---------------------------------------------------------------------------
// Add
// ---------------------------------------------------------------------------

func TestWishlistRepository_Add_Success(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO wishlists").
		WithArgs("user-1", "prod-1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Add(context.Background(), "user-1", "prod-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_Add_ExecError(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO wishlists").
		WithArgs("user-1", "prod-1").
		WillReturnError(errors.New("connection refused"))

	err := repo.Add(context.Background(), "user-1", "prod-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add to wishlist")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Remove
// ---------------------------------------------------------------------------

func TestWishlistRepository_Remove_Success(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM wishlists WHERE user_id =").
		WithArgs("user-1", "prod-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Remove(context.Background(), "user-1", "prod-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_Remove_NotFound(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM wishlists WHERE user_id =").
		WithArgs("user-1", "prod-missing").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.Remove(context.Background(), "user-1", "prod-missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrNotFound), "expected ErrNotFound, got: %v", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_Remove_ExecError(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM wishlists WHERE user_id =").
		WithArgs("user-1", "prod-1").
		WillReturnError(errors.New("database timeout"))

	err := repo.Remove(context.Background(), "user-1", "prod-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove from wishlist")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestWishlistRepository_List_Success(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Mock COUNT query.
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM wishlists WHERE user_id =").
		WithArgs("user-1").
		WillReturnRows(countRows)

	// Mock paginated SELECT query.
	selectRows := pgxmock.NewRows([]string{"user_id", "product_id", "created_at"}).
		AddRow("user-1", "prod-1", now).
		AddRow("user-1", "prod-2", now.Add(-time.Hour))
	mock.ExpectQuery("SELECT user_id, product_id, created_at FROM wishlists").
		WithArgs("user-1", 10, 0).
		WillReturnRows(selectRows)

	items, total, err := repo.List(context.Background(), "user-1", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, items, 2)
	assert.Equal(t, "prod-1", items[0].ProductID)
	assert.Equal(t, "user-1", items[0].UserID)
	assert.Equal(t, "prod-2", items[1].ProductID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_List_Empty(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	// Mock COUNT query returning 0.
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM wishlists WHERE user_id =").
		WithArgs("user-empty").
		WillReturnRows(countRows)

	// Mock SELECT returning no rows.
	selectRows := pgxmock.NewRows([]string{"user_id", "product_id", "created_at"})
	mock.ExpectQuery("SELECT user_id, product_id, created_at FROM wishlists").
		WithArgs("user-empty", 10, 0).
		WillReturnRows(selectRows)

	items, total, err := repo.List(context.Background(), "user-empty", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NotNil(t, items, "should return empty slice, not nil")
	assert.Len(t, items, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_List_CountError(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM wishlists WHERE user_id =").
		WithArgs("user-1").
		WillReturnError(errors.New("count query failed"))

	items, total, err := repo.List(context.Background(), "user-1", 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count wishlist items")
	assert.Nil(t, items)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_List_QueryError(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	// COUNT succeeds.
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(3)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM wishlists WHERE user_id =").
		WithArgs("user-1").
		WillReturnRows(countRows)

	// SELECT fails.
	mock.ExpectQuery("SELECT user_id, product_id, created_at FROM wishlists").
		WithArgs("user-1", 10, 0).
		WillReturnError(errors.New("select query failed"))

	items, total, err := repo.List(context.Background(), "user-1", 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list wishlist items")
	assert.Nil(t, items)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Exists
// ---------------------------------------------------------------------------

func TestWishlistRepository_Exists_True(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1", "prod-1").
		WillReturnRows(rows)

	exists, err := repo.Exists(context.Background(), "user-1", "prod-1")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_Exists_False(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1", "prod-missing").
		WillReturnRows(rows)

	exists, err := repo.Exists(context.Background(), "user-1", "prod-missing")
	require.NoError(t, err)
	assert.False(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWishlistRepository_Exists_Error(t *testing.T) {
	repo, mock := newWishlistTestFixture(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("user-1", "prod-1").
		WillReturnError(errors.New("query failed"))

	exists, err := repo.Exists(context.Background(), "user-1", "prod-1")
	require.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "check wishlist item exists")
	assert.NoError(t, mock.ExpectationsWereMet())
}
