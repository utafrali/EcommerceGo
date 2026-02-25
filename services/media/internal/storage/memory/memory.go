package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/utafrali/EcommerceGo/services/media/internal/storage"
)

// fileEntry stores metadata about an uploaded file in memory.
type fileEntry struct {
	Key         string
	ContentType string
	Size        int64
	URL         string
}

// Storage implements storage.Storage using an in-memory map.
// It stores metadata only (no actual file bytes) for testing purposes.
type Storage struct {
	mu      sync.RWMutex
	files   map[string]*fileEntry
	baseURL string
}

// New creates a new in-memory storage instance.
func New(baseURL string) *Storage {
	return &Storage{
		files:   make(map[string]*fileEntry),
		baseURL: baseURL,
	}
}

// Upload stores file metadata in memory and returns the generated URL.
func (s *Storage) Upload(_ context.Context, input *storage.UploadInput) (*storage.UploadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	url := fmt.Sprintf("%s/media/%s", s.baseURL, input.Key)

	s.files[input.Key] = &fileEntry{
		Key:         input.Key,
		ContentType: input.ContentType,
		Size:        input.Size,
		URL:         url,
	}

	return &storage.UploadResult{
		Key: input.Key,
		URL: url,
	}, nil
}

// Delete removes file metadata from memory.
func (s *Storage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.files[key]; !exists {
		return fmt.Errorf("file not found: %s", key)
	}

	delete(s.files, key)
	return nil
}

// GetURL returns the URL for the given key.
func (s *Storage) GetURL(_ context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.files[key]
	if !exists {
		return "", fmt.Errorf("file not found: %s", key)
	}

	return entry.URL, nil
}
