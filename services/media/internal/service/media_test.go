package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	pkgkafka "github.com/utafrali/EcommerceGo/pkg/kafka"
	"github.com/utafrali/EcommerceGo/services/media/internal/domain"
	"github.com/utafrali/EcommerceGo/services/media/internal/event"
	"github.com/utafrali/EcommerceGo/services/media/internal/storage"
)

// --- Mock Repository ---

type mockMediaRepository struct {
	mock.Mock
}

func (m *mockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *mockMediaRepository) GetByID(ctx context.Context, id string) (*domain.MediaFile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.MediaFile), args.Error(1)
}

func (m *mockMediaRepository) ListByOwner(ctx context.Context, ownerID, ownerType string, offset, limit int) ([]domain.MediaFile, int, error) {
	args := m.Called(ctx, ownerID, ownerType, offset, limit)
	return args.Get(0).([]domain.MediaFile), args.Int(1), args.Error(2)
}

func (m *mockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *mockMediaRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Mock Storage ---

type mockStorage struct {
	mock.Mock
}

func (m *mockStorage) Upload(ctx context.Context, input *storage.UploadInput) (*storage.UploadResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.UploadResult), args.Error(1)
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockStorage) GetURL(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

// --- Test Helpers ---

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockMediaRepository, store *mockStorage) *MediaService {
	logger := newTestLogger()
	// Create a Kafka producer that will fail silently in tests (no real broker).
	kafkaCfg := pkgkafka.DefaultProducerConfig([]string{"localhost:9092"})
	kafkaProducer := pkgkafka.NewProducer(kafkaCfg, logger)
	producer := event.NewProducer(kafkaProducer, logger)
	return NewMediaService(repo, store, producer, logger)
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// --- Tests ---

func TestUploadMedia_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	store.On("Upload", ctx, mock.AnythingOfType("*storage.UploadInput")).
		Return(&storage.UploadResult{
			Key: "product/owner-123/some-uuid",
			URL: "http://localhost:8011/media/product/owner-123/some-uuid",
		}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake image data"),
		AltText:     "A product photo",
	}

	media, err := svc.UploadMedia(ctx, input)

	require.NoError(t, err)
	assert.NotEmpty(t, media.ID)
	assert.Equal(t, "owner-123", media.OwnerID)
	assert.Equal(t, "product", media.OwnerType)
	assert.Equal(t, "photo.jpg", media.OriginalName)
	assert.Equal(t, "image/jpeg", media.ContentType)
	assert.Equal(t, int64(1024), media.Size)
	assert.Equal(t, "A product photo", media.AltText)
	assert.NotEmpty(t, media.URL)
	assert.NotZero(t, media.CreatedAt)
	assert.NotZero(t, media.UpdatedAt)
	assert.NotNil(t, media.Metadata)

	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestUploadMedia_InvalidContentType(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "document.pdf",
		ContentType: "application/pdf",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUploadMedia_ExceedsMaxSize(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "huge.jpg",
		ContentType: "image/jpeg",
		Size:        domain.MaxFileSize + 1,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUploadMedia_ZeroSize(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "empty.jpg",
		ContentType: "image/jpeg",
		Size:        0,
		Data:        strings.NewReader(""),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUploadMedia_EmptyFileName(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUploadMedia_EmptyOwnerID(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	input := &UploadMediaInput{
		OwnerID:     "",
		OwnerType:   "product",
		FileName:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUploadMedia_EmptyOwnerType(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "",
		FileName:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUploadMedia_StorageError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	store.On("Upload", ctx, mock.AnythingOfType("*storage.UploadInput")).
		Return(nil, errors.New("storage unavailable"))

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload to storage")

	store.AssertExpectations(t)
}

func TestUploadMedia_RepositoryError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	store.On("Upload", ctx, mock.AnythingOfType("*storage.UploadInput")).
		Return(&storage.UploadResult{
			Key: "product/owner-123/some-uuid",
			URL: "http://localhost:8011/media/product/owner-123/some-uuid",
		}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.MediaFile")).
		Return(errors.New("database error"))

	// Storage.Delete should be called for cleanup on DB failure.
	store.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create media record")

	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestGetMedia_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	expected := &domain.MediaFile{
		ID:           "media-123",
		OwnerID:      "owner-123",
		OwnerType:    "product",
		FileName:     "product/owner-123/media-123",
		OriginalName: "photo.jpg",
		ContentType:  "image/jpeg",
		Size:         1024,
		URL:          "http://localhost:8011/media/product/owner-123/media-123",
	}

	repo.On("GetByID", ctx, "media-123").Return(expected, nil)

	media, err := svc.GetMedia(ctx, "media-123")

	require.NoError(t, err)
	assert.Equal(t, expected, media)

	repo.AssertExpectations(t)
}

func TestGetMedia_NotFound(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	media, err := svc.GetMedia(ctx, "nonexistent")

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestListMediaByOwner_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	expectedMedia := []domain.MediaFile{
		{ID: "1", OwnerID: "owner-123", OwnerType: "product", OriginalName: "photo1.jpg"},
		{ID: "2", OwnerID: "owner-123", OwnerType: "product", OriginalName: "photo2.jpg"},
	}

	repo.On("ListByOwner", ctx, "owner-123", "product", 0, 20).Return(expectedMedia, 2, nil)

	mediaFiles, total, err := svc.ListMediaByOwner(ctx, "owner-123", "product", 1, 20)

	require.NoError(t, err)
	assert.Len(t, mediaFiles, 2)
	assert.Equal(t, 2, total)

	repo.AssertExpectations(t)
}

func TestListMediaByOwner_DefaultPagination(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	repo.On("ListByOwner", ctx, "owner-123", "product", 0, 20).Return([]domain.MediaFile{}, 0, nil)

	mediaFiles, total, err := svc.ListMediaByOwner(ctx, "owner-123", "product", 0, 0)

	require.NoError(t, err)
	assert.Empty(t, mediaFiles)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestListMediaByOwner_CapPerPage(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	repo.On("ListByOwner", ctx, "owner-123", "product", 0, 100).Return([]domain.MediaFile{}, 0, nil)

	mediaFiles, total, err := svc.ListMediaByOwner(ctx, "owner-123", "product", 1, 500)

	require.NoError(t, err)
	assert.Empty(t, mediaFiles)
	assert.Equal(t, 0, total)

	repo.AssertExpectations(t)
}

func TestDeleteMedia_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	existing := &domain.MediaFile{
		ID:       "media-123",
		FileName: "product/owner-123/media-123",
	}

	repo.On("GetByID", ctx, "media-123").Return(existing, nil)
	store.On("Delete", ctx, "product/owner-123/media-123").Return(nil)
	repo.On("Delete", ctx, "media-123").Return(nil)

	err := svc.DeleteMedia(ctx, "media-123")

	require.NoError(t, err)
	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestDeleteMedia_NotFound(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	err := svc.DeleteMedia(ctx, "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_Success(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	existing := &domain.MediaFile{
		ID:        "media-123",
		AltText:   "Old alt text",
		SortOrder: 0,
		Metadata:  map[string]any{},
	}

	repo.On("GetByID", ctx, "media-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	input := &UpdateMediaInput{
		AltText:   strPtr("New alt text"),
		SortOrder: intPtr(5),
	}

	media, err := svc.UpdateMediaMetadata(ctx, "media-123", input)

	require.NoError(t, err)
	assert.Equal(t, "New alt text", media.AltText)
	assert.Equal(t, 5, media.SortOrder)

	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_NotFound(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	repo.On("GetByID", ctx, "nonexistent").Return(nil, apperrors.ErrNotFound)

	input := &UpdateMediaInput{
		AltText: strPtr("New alt text"),
	}

	media, err := svc.UpdateMediaMetadata(ctx, "nonexistent", input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_OnlyAltText(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	existing := &domain.MediaFile{
		ID:        "media-123",
		AltText:   "Old alt text",
		SortOrder: 3,
		Metadata:  map[string]any{},
	}

	repo.On("GetByID", ctx, "media-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	input := &UpdateMediaInput{
		AltText: strPtr("Updated alt text only"),
	}

	media, err := svc.UpdateMediaMetadata(ctx, "media-123", input)

	require.NoError(t, err)
	assert.Equal(t, "Updated alt text only", media.AltText)
	assert.Equal(t, 3, media.SortOrder) // Sort order should remain unchanged.

	repo.AssertExpectations(t)
}

func TestUpdateMediaMetadata_OnlySortOrder(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	existing := &domain.MediaFile{
		ID:        "media-123",
		AltText:   "Original alt text",
		SortOrder: 0,
		Metadata:  map[string]any{},
	}

	repo.On("GetByID", ctx, "media-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

	input := &UpdateMediaInput{
		SortOrder: intPtr(10),
	}

	media, err := svc.UpdateMediaMetadata(ctx, "media-123", input)

	require.NoError(t, err)
	assert.Equal(t, "Original alt text", media.AltText) // Alt text should remain unchanged.
	assert.Equal(t, 10, media.SortOrder)

	repo.AssertExpectations(t)
}

func TestUploadMedia_AllAllowedContentTypes(t *testing.T) {
	allowedTypes := []string{"image/jpeg", "image/png", "image/webp", "image/gif"}

	for _, ct := range allowedTypes {
		t.Run(ct, func(t *testing.T) {
			repo := new(mockMediaRepository)
			store := new(mockStorage)
			svc := newTestService(repo, store)
			ctx := context.Background()

			store.On("Upload", ctx, mock.AnythingOfType("*storage.UploadInput")).
				Return(&storage.UploadResult{
					Key: "product/owner-123/some-uuid",
					URL: "http://localhost:8011/media/product/owner-123/some-uuid",
				}, nil)

			repo.On("Create", ctx, mock.AnythingOfType("*domain.MediaFile")).Return(nil)

			input := &UploadMediaInput{
				OwnerID:     "owner-123",
				OwnerType:   "product",
				FileName:    "test-file",
				ContentType: ct,
				Size:        512,
				Data:        strings.NewReader("fake data"),
			}

			media, err := svc.UploadMedia(ctx, input)

			require.NoError(t, err)
			assert.Equal(t, ct, media.ContentType)

			repo.AssertExpectations(t)
			store.AssertExpectations(t)
		})
	}
}

func TestUploadMedia_StorageCleanupOnDBError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	store.On("Upload", ctx, mock.AnythingOfType("*storage.UploadInput")).
		Return(&storage.UploadResult{
			Key: "product/owner-123/some-uuid",
			URL: "http://localhost:8011/media/product/owner-123/some-uuid",
		}, nil)

	repo.On("Create", ctx, mock.AnythingOfType("*domain.MediaFile")).
		Return(errors.New("database error"))

	// Verify that storage.Delete is called when DB create fails.
	store.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)

	input := &UploadMediaInput{
		OwnerID:     "owner-123",
		OwnerType:   "product",
		FileName:    "photo.jpg",
		ContentType: "image/jpeg",
		Size:        1024,
		Data:        strings.NewReader("fake data"),
	}

	media, err := svc.UploadMedia(ctx, input)

	assert.Nil(t, media)
	assert.Error(t, err)

	// Ensure Delete was called on storage for cleanup.
	store.AssertCalled(t, "Delete", ctx, mock.AnythingOfType("string"))

	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestDeleteMedia_StorageErrorContinues(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	existing := &domain.MediaFile{
		ID:       "media-123",
		FileName: "product/owner-123/media-123",
	}

	repo.On("GetByID", ctx, "media-123").Return(existing, nil)
	store.On("Delete", ctx, "product/owner-123/media-123").Return(errors.New("storage error"))
	repo.On("Delete", ctx, "media-123").Return(nil)

	// Delete should succeed even if storage deletion fails.
	err := svc.DeleteMedia(ctx, "media-123")

	require.NoError(t, err)
	repo.AssertExpectations(t)
	store.AssertExpectations(t)
}

func TestUpdateMediaMetadata_RepositoryUpdateError(t *testing.T) {
	repo := new(mockMediaRepository)
	store := new(mockStorage)
	svc := newTestService(repo, store)
	ctx := context.Background()

	existing := &domain.MediaFile{
		ID:        "media-123",
		AltText:   "Old alt text",
		SortOrder: 0,
		Metadata:  map[string]any{},
	}

	repo.On("GetByID", ctx, "media-123").Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.MediaFile")).
		Return(errors.New("database error"))

	input := &UpdateMediaInput{
		AltText: strPtr("New alt text"),
	}

	media, err := svc.UpdateMediaMetadata(ctx, "media-123", input)

	assert.Nil(t, media)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update media")

	repo.AssertExpectations(t)
}
