package storage

import (
	"context"
	"io"
)

// Storage defines the interface for file storage operations.
type Storage interface {
	// Upload stores a file and returns the result with key and URL.
	Upload(ctx context.Context, input *UploadInput) (*UploadResult, error)

	// Delete removes a file by its key.
	Delete(ctx context.Context, key string) error

	// GetURL returns the public URL for the given key.
	GetURL(ctx context.Context, key string) (string, error)
}

// UploadInput holds the parameters for uploading a file.
type UploadInput struct {
	Key         string
	ContentType string
	Size        int64
	Data        io.Reader
}

// UploadResult holds the result of a successful upload.
type UploadResult struct {
	Key string
	URL string
}
