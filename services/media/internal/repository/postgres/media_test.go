package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func setupRepo(t *testing.T) (*MediaRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	repo := NewMediaRepository(mock)
	return repo, mock
}

var mediaColumns = []string{
	"id", "owner_id", "owner_type", "file_name", "original_name",
	"content_type", "size", "url", "thumbnail_url", "alt_text",
	"sort_order", "metadata", "created_at", "updated_at",
}

var mediaColumnsWithCount = []string{
	"id", "owner_id", "owner_type", "file_name", "original_name",
	"content_type", "size", "url", "thumbnail_url", "alt_text",
	"sort_order", "metadata", "created_at", "updated_at", "total_count",
}

func sampleMediaFile() domain.MediaFile {
	thumb := "https://cdn.example.com/thumb/photo.jpg"
	return domain.MediaFile{
		ID:           "media-1",
		OwnerID:      "product-100",
		OwnerType:    domain.OwnerTypeProduct,
		FileName:     "abc123.jpg",
		OriginalName: "photo.jpg",
		ContentType:  "image/jpeg",
		Size:         204800,
		URL:          "https://cdn.example.com/abc123.jpg",
		ThumbnailURL: &thumb,
		AltText:      "A product photo",
		SortOrder:    1,
		Metadata:     map[string]any{"width": float64(1920), "height": float64(1080)},
		CreatedAt:    time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

func sampleMediaFile2() domain.MediaFile {
	return domain.MediaFile{
		ID:           "media-2",
		OwnerID:      "product-100",
		OwnerType:    domain.OwnerTypeProduct,
		FileName:     "def456.png",
		OriginalName: "banner.png",
		ContentType:  "image/png",
		Size:         512000,
		URL:          "https://cdn.example.com/def456.png",
		ThumbnailURL: nil,
		AltText:      "A banner image",
		SortOrder:    2,
		Metadata:     map[string]any{"format": "png"},
		CreatedAt:    time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
	}
}

func mustMarshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestMediaRepository_Create_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m := sampleMediaFile()
	metadataJSON := mustMarshalJSON(t, m.Metadata)

	mock.ExpectExec("INSERT INTO media_files").
		WithArgs(
			m.ID, m.OwnerID, m.OwnerType, m.FileName, m.OriginalName,
			m.ContentType, m.Size, m.URL, m.ThumbnailURL, m.AltText,
			m.SortOrder, metadataJSON, m.CreatedAt, m.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), &m)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_Create_ExecError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m := sampleMediaFile()
	metadataJSON := mustMarshalJSON(t, m.Metadata)

	mock.ExpectExec("INSERT INTO media_files").
		WithArgs(
			m.ID, m.OwnerID, m.OwnerType, m.FileName, m.OriginalName,
			m.ContentType, m.Size, m.URL, m.ThumbnailURL, m.AltText,
			m.SortOrder, metadataJSON, m.CreatedAt, m.UpdatedAt,
		).
		WillReturnError(errors.New("duplicate key"))

	err := repo.Create(context.Background(), &m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert media file")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestMediaRepository_GetByID_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m := sampleMediaFile()
	metadataJSON := mustMarshalJSON(t, m.Metadata)

	mock.ExpectQuery("SELECT .+ FROM media_files WHERE id").
		WithArgs(m.ID).
		WillReturnRows(
			pgxmock.NewRows(mediaColumns).
				AddRow(
					m.ID, m.OwnerID, m.OwnerType, m.FileName, m.OriginalName,
					m.ContentType, m.Size, m.URL, m.ThumbnailURL, m.AltText,
					m.SortOrder, metadataJSON, m.CreatedAt, m.UpdatedAt,
				),
		)

	result, err := repo.GetByID(context.Background(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, result.ID)
	assert.Equal(t, m.OwnerID, result.OwnerID)
	assert.Equal(t, m.OwnerType, result.OwnerType)
	assert.Equal(t, m.FileName, result.FileName)
	assert.Equal(t, m.OriginalName, result.OriginalName)
	assert.Equal(t, m.ContentType, result.ContentType)
	assert.Equal(t, m.Size, result.Size)
	assert.Equal(t, m.URL, result.URL)
	require.NotNil(t, result.ThumbnailURL)
	assert.Equal(t, *m.ThumbnailURL, *result.ThumbnailURL)
	assert.Equal(t, m.AltText, result.AltText)
	assert.Equal(t, m.SortOrder, result.SortOrder)
	assert.Equal(t, float64(1920), result.Metadata["width"])
	assert.Equal(t, float64(1080), result.Metadata["height"])
	assert.Equal(t, m.CreatedAt, result.CreatedAt)
	assert.Equal(t, m.UpdatedAt, result.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM media_files WHERE id").
		WithArgs("nonexistent-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "nonexistent-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_GetByID_ScanError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM media_files WHERE id").
		WithArgs("media-1").
		WillReturnError(errors.New("connection refused"))

	result, err := repo.GetByID(context.Background(), "media-1")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan media file")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// ListByOwner
// ---------------------------------------------------------------------------

func TestMediaRepository_ListByOwner_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m1 := sampleMediaFile()
	m2 := sampleMediaFile2()
	m1JSON := mustMarshalJSON(t, m1.Metadata)
	m2JSON := mustMarshalJSON(t, m2.Metadata)
	totalCount := 2

	mock.ExpectQuery("SELECT .+ FROM media_files WHERE owner_id").
		WithArgs(m1.OwnerID, m1.OwnerType, 10, 0).
		WillReturnRows(
			pgxmock.NewRows(mediaColumnsWithCount).
				AddRow(
					m1.ID, m1.OwnerID, m1.OwnerType, m1.FileName, m1.OriginalName,
					m1.ContentType, m1.Size, m1.URL, m1.ThumbnailURL, m1.AltText,
					m1.SortOrder, m1JSON, m1.CreatedAt, m1.UpdatedAt, totalCount,
				).
				AddRow(
					m2.ID, m2.OwnerID, m2.OwnerType, m2.FileName, m2.OriginalName,
					m2.ContentType, m2.Size, m2.URL, m2.ThumbnailURL, m2.AltText,
					m2.SortOrder, m2JSON, m2.CreatedAt, m2.UpdatedAt, totalCount,
				),
		)

	results, total, err := repo.ListByOwner(context.Background(), m1.OwnerID, m1.OwnerType, 0, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, totalCount, total)

	// Verify first item
	assert.Equal(t, m1.ID, results[0].ID)
	assert.Equal(t, m1.OwnerID, results[0].OwnerID)
	assert.Equal(t, m1.FileName, results[0].FileName)
	require.NotNil(t, results[0].ThumbnailURL)
	assert.Equal(t, *m1.ThumbnailURL, *results[0].ThumbnailURL)
	assert.Equal(t, float64(1920), results[0].Metadata["width"])

	// Verify second item
	assert.Equal(t, m2.ID, results[1].ID)
	assert.Nil(t, results[1].ThumbnailURL)
	assert.Equal(t, "png", results[1].Metadata["format"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_ListByOwner_Empty(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM media_files WHERE owner_id").
		WithArgs("owner-none", "product", 20, 0).
		WillReturnRows(pgxmock.NewRows(mediaColumnsWithCount))

	results, total, err := repo.ListByOwner(context.Background(), "owner-none", "product", 0, 20)
	require.NoError(t, err)
	assert.Equal(t, []domain.MediaFile{}, results)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_ListByOwner_QueryError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectQuery("SELECT .+ FROM media_files WHERE owner_id").
		WithArgs("owner-1", "product", 10, 0).
		WillReturnError(errors.New("connection timeout"))

	results, total, err := repo.ListByOwner(context.Background(), "owner-1", "product", 0, 10)
	assert.Nil(t, results)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list media files")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestMediaRepository_Update_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m := sampleMediaFile()
	m.AltText = "Updated alt text"
	m.SortOrder = 5
	m.Metadata = map[string]any{"width": float64(800), "height": float64(600)}

	metadataJSON := mustMarshalJSON(t, m.Metadata)

	mock.ExpectExec("UPDATE media_files").
		WithArgs(
			m.AltText, m.SortOrder, metadataJSON,
			pgxmock.AnyArg(), // UpdatedAt is set to time.Now().UTC() inside Update
			m.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), &m)
	assert.NoError(t, err)
	// Verify that UpdatedAt was refreshed
	assert.WithinDuration(t, time.Now().UTC(), m.UpdatedAt, 2*time.Second)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_Update_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m := sampleMediaFile()
	m.ID = "nonexistent-media"
	metadataJSON := mustMarshalJSON(t, m.Metadata)

	mock.ExpectExec("UPDATE media_files").
		WithArgs(
			m.AltText, m.SortOrder, metadataJSON,
			pgxmock.AnyArg(),
			m.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), &m)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_Update_ExecError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	m := sampleMediaFile()
	metadataJSON := mustMarshalJSON(t, m.Metadata)

	mock.ExpectExec("UPDATE media_files").
		WithArgs(
			m.AltText, m.SortOrder, metadataJSON,
			pgxmock.AnyArg(),
			m.ID,
		).
		WillReturnError(errors.New("db connection lost"))

	err := repo.Update(context.Background(), &m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update media file")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestMediaRepository_Delete_Success(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM media_files WHERE id").
		WithArgs("media-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(context.Background(), "media-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_Delete_NotFound(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM media_files WHERE id").
		WithArgs("nonexistent-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.Delete(context.Background(), "nonexistent-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMediaRepository_Delete_ExecError(t *testing.T) {
	repo, mock := setupRepo(t)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM media_files WHERE id").
		WithArgs("media-1").
		WillReturnError(errors.New("foreign key constraint"))

	err := repo.Delete(context.Background(), "media-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete media file")
	assert.NoError(t, mock.ExpectationsWereMet())
}
