package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
	"github.com/utafrali/EcommerceGo/services/media/internal/event"
	"github.com/utafrali/EcommerceGo/services/media/internal/repository"
	"github.com/utafrali/EcommerceGo/services/media/internal/storage"
)

// safeIDPattern matches only alphanumeric characters, hyphens, and underscores.
var safeIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// MediaService implements the business logic for media operations.
type MediaService struct {
	repo     repository.MediaRepository
	storage  storage.Storage
	producer *event.Producer
	logger   *slog.Logger
}

// NewMediaService creates a new media service.
func NewMediaService(
	repo repository.MediaRepository,
	store storage.Storage,
	producer *event.Producer,
	logger *slog.Logger,
) *MediaService {
	return &MediaService{
		repo:     repo,
		storage:  store,
		producer: producer,
		logger:   logger,
	}
}

// UploadMediaInput holds the parameters for uploading a media file.
type UploadMediaInput struct {
	OwnerID     string
	OwnerType   string
	FileName    string
	ContentType string
	Size        int64
	Data        io.Reader
	AltText     string
}

// UpdateMediaInput holds the parameters for updating media metadata.
type UpdateMediaInput struct {
	AltText   *string
	SortOrder *int
}

// UploadMedia validates the input, uploads to storage, and saves metadata.
func (s *MediaService) UploadMedia(ctx context.Context, input *UploadMediaInput) (*domain.MediaFile, error) {
	// Validate content type.
	if !domain.IsAllowedContentType(input.ContentType) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("content type %q is not allowed", input.ContentType))
	}

	// Validate file size.
	if input.Size > domain.MaxFileSize {
		return nil, apperrors.InvalidInput(fmt.Sprintf("file size %d exceeds maximum allowed size of %d bytes", input.Size, domain.MaxFileSize))
	}

	if input.Size <= 0 {
		return nil, apperrors.InvalidInput("file size must be greater than zero")
	}

	if input.FileName == "" {
		return nil, apperrors.InvalidInput("file name is required")
	}

	if input.OwnerID == "" {
		return nil, apperrors.InvalidInput("owner id is required")
	}

	if input.OwnerType == "" {
		return nil, apperrors.InvalidInput("owner type is required")
	}

	// Validate owner type against allowed set to prevent path traversal.
	if !domain.IsValidOwnerType(input.OwnerType) {
		return nil, apperrors.InvalidInput(fmt.Sprintf("owner type %q is not allowed", input.OwnerType))
	}

	// Sanitize owner ID: only allow alphanumeric, hyphens, underscores.
	if !safeIDPattern.MatchString(input.OwnerID) {
		return nil, apperrors.InvalidInput("owner id contains invalid characters")
	}

	// Generate a unique file key.
	id := uuid.New().String()
	key := fmt.Sprintf("%s/%s/%s", input.OwnerType, input.OwnerID, id)

	// Upload to storage.
	result, err := s.storage.Upload(ctx, &storage.UploadInput{
		Key:         key,
		ContentType: input.ContentType,
		Size:        input.Size,
		Data:        input.Data,
	})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	now := time.Now().UTC()
	media := &domain.MediaFile{
		ID:           id,
		OwnerID:      input.OwnerID,
		OwnerType:    input.OwnerType,
		FileName:     key,
		OriginalName: input.FileName,
		ContentType:  input.ContentType,
		Size:         input.Size,
		URL:          result.URL,
		AltText:      input.AltText,
		SortOrder:    0,
		Metadata:     make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		// Attempt to clean up the uploaded file on DB failure.
		if delErr := s.storage.Delete(ctx, key); delErr != nil {
			s.logger.ErrorContext(ctx, "failed to clean up storage after db error",
				slog.String("key", key),
				slog.String("error", delErr.Error()),
			)
		}
		return nil, fmt.Errorf("create media record: %w", err)
	}

	// Publish event; errors are logged but do not fail the operation.
	if err := s.producer.PublishMediaUploaded(ctx, media); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish media.uploaded event",
			slog.String("media_id", media.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "media uploaded",
		slog.String("media_id", media.ID),
		slog.String("owner_id", media.OwnerID),
		slog.String("owner_type", media.OwnerType),
		slog.String("content_type", media.ContentType),
		slog.Int64("size", media.Size),
	)

	return media, nil
}

// GetMedia retrieves a media file by its ID.
func (s *MediaService) GetMedia(ctx context.Context, id string) (*domain.MediaFile, error) {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get media by id: %w", err)
	}
	return media, nil
}

// ListMediaByOwner returns a paginated list of media files for a given owner.
func (s *MediaService) ListMediaByOwner(ctx context.Context, ownerID, ownerType string, page, perPage int) ([]domain.MediaFile, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	mediaFiles, total, err := s.repo.ListByOwner(ctx, ownerID, ownerType, offset, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list media by owner: %w", err)
	}

	return mediaFiles, total, nil
}

// DeleteMedia removes a media file from storage and the database.
func (s *MediaService) DeleteMedia(ctx context.Context, id string) error {
	// Verify the media exists before deleting.
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get media for delete: %w", err)
	}

	// Delete from storage.
	if err := s.storage.Delete(ctx, media.FileName); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete from storage",
			slog.String("media_id", id),
			slog.String("key", media.FileName),
			slog.String("error", err.Error()),
		)
		// Continue with DB deletion even if storage delete fails.
	}

	// Delete from database.
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete media: %w", err)
	}

	// Publish event; errors are logged but do not fail the operation.
	if err := s.producer.PublishMediaDeleted(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish media.deleted event",
			slog.String("media_id", id),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "media deleted",
		slog.String("media_id", id),
	)

	return nil
}

// UpdateMediaMetadata updates the alt text and/or sort order of a media file.
func (s *MediaService) UpdateMediaMetadata(ctx context.Context, id string, input *UpdateMediaInput) (*domain.MediaFile, error) {
	media, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get media for update: %w", err)
	}

	if input.AltText != nil {
		media.AltText = *input.AltText
	}

	if input.SortOrder != nil {
		media.SortOrder = *input.SortOrder
	}

	media.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("update media: %w", err)
	}

	s.logger.InfoContext(ctx, "media metadata updated",
		slog.String("media_id", media.ID),
	)

	return media, nil
}
