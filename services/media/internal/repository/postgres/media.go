package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
)

// MediaRepository implements repository.MediaRepository using PostgreSQL.
type MediaRepository struct {
	pool *pgxpool.Pool
}

// NewMediaRepository creates a new PostgreSQL-backed media repository.
func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{pool: pool}
}

// Create inserts a new media file record into the database.
func (r *MediaRepository) Create(ctx context.Context, m *domain.MediaFile) error {
	metadataJSON, err := json.Marshal(m.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO media_files (id, owner_id, owner_type, file_name, original_name, content_type, size, url, thumbnail_url, alt_text, sort_order, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err = r.pool.Exec(ctx, query,
		m.ID,
		m.OwnerID,
		m.OwnerType,
		m.FileName,
		m.OriginalName,
		m.ContentType,
		m.Size,
		m.URL,
		m.ThumbnailURL,
		m.AltText,
		m.SortOrder,
		metadataJSON,
		m.CreatedAt,
		m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert media file: %w", err)
	}

	return nil
}

// GetByID retrieves a media file by its ID.
func (r *MediaRepository) GetByID(ctx context.Context, id string) (*domain.MediaFile, error) {
	query := `
		SELECT id, owner_id, owner_type, file_name, original_name, content_type, size, url, thumbnail_url, alt_text, sort_order, metadata, created_at, updated_at
		FROM media_files
		WHERE id = $1`

	return r.scanMediaFile(ctx, query, id)
}

// ListByOwner returns media files for a given owner with pagination.
func (r *MediaRepository) ListByOwner(ctx context.Context, ownerID, ownerType string, offset, limit int) ([]domain.MediaFile, int, error) {
	query := `
		SELECT id, owner_id, owner_type, file_name, original_name, content_type, size, url, thumbnail_url, alt_text, sort_order, metadata, created_at, updated_at,
			   count(*) OVER() AS total_count
		FROM media_files
		WHERE owner_id = $1 AND owner_type = $2
		ORDER BY sort_order ASC, created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, ownerID, ownerType, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list media files: %w", err)
	}
	defer rows.Close()

	var (
		mediaFiles []domain.MediaFile
		totalCount int
	)

	for rows.Next() {
		var (
			m            domain.MediaFile
			metadataJSON []byte
		)

		if err := rows.Scan(
			&m.ID,
			&m.OwnerID,
			&m.OwnerType,
			&m.FileName,
			&m.OriginalName,
			&m.ContentType,
			&m.Size,
			&m.URL,
			&m.ThumbnailURL,
			&m.AltText,
			&m.SortOrder,
			&metadataJSON,
			&m.CreatedAt,
			&m.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan media file row: %w", err)
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
				return nil, 0, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		mediaFiles = append(mediaFiles, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate media file rows: %w", err)
	}

	if mediaFiles == nil {
		mediaFiles = []domain.MediaFile{}
	}

	return mediaFiles, totalCount, nil
}

// Update modifies an existing media file record in the database.
func (r *MediaRepository) Update(ctx context.Context, m *domain.MediaFile) error {
	metadataJSON, err := json.Marshal(m.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	m.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE media_files
		SET alt_text = $1, sort_order = $2, metadata = $3, updated_at = $4
		WHERE id = $5`

	ct, err := r.pool.Exec(ctx, query,
		m.AltText,
		m.SortOrder,
		metadataJSON,
		m.UpdatedAt,
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("update media file: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("media_file", m.ID)
	}

	return nil
}

// Delete removes a media file record from the database by its ID.
func (r *MediaRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM media_files WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete media file: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return apperrors.NotFound("media_file", id)
	}

	return nil
}

// scanMediaFile is a helper that executes a query expected to return a single media file row.
func (r *MediaRepository) scanMediaFile(ctx context.Context, query string, args ...any) (*domain.MediaFile, error) {
	var (
		m            domain.MediaFile
		metadataJSON []byte
	)

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&m.ID,
		&m.OwnerID,
		&m.OwnerType,
		&m.FileName,
		&m.OriginalName,
		&m.ContentType,
		&m.Size,
		&m.URL,
		&m.ThumbnailURL,
		&m.AltText,
		&m.SortOrder,
		&metadataJSON,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan media file: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	return &m, nil
}
