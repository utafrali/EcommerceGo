package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Content Type Validation Tests
// ============================================================================

func TestIsAllowedContentType_JPEG(t *testing.T) {
	assert.True(t, IsAllowedContentType("image/jpeg"))
}

func TestIsAllowedContentType_PNG(t *testing.T) {
	assert.True(t, IsAllowedContentType("image/png"))
}

func TestIsAllowedContentType_WebP(t *testing.T) {
	assert.True(t, IsAllowedContentType("image/webp"))
}

func TestIsAllowedContentType_GIF(t *testing.T) {
	assert.True(t, IsAllowedContentType("image/gif"))
}

func TestIsAllowedContentType_NotAllowed(t *testing.T) {
	assert.False(t, IsAllowedContentType("image/bmp"))
	assert.False(t, IsAllowedContentType("application/pdf"))
	assert.False(t, IsAllowedContentType("text/plain"))
	assert.False(t, IsAllowedContentType(""))
}

func TestIsAllowedContentType_CaseSensitive(t *testing.T) {
	assert.False(t, IsAllowedContentType("IMAGE/JPEG"))
	assert.False(t, IsAllowedContentType("Image/Png"))
}

// ============================================================================
// MaxFileSize Tests
// ============================================================================

func TestMaxFileSize_Is10MB(t *testing.T) {
	expected := int64(10 * 1024 * 1024)
	assert.Equal(t, expected, MaxFileSize)
}

// ============================================================================
// Owner Type Validation Tests
// ============================================================================

func TestValidOwnerTypes_ContainsAll(t *testing.T) {
	types := ValidOwnerTypes()
	expected := []string{OwnerTypeProduct, OwnerTypeUser, OwnerTypeCategory}
	assert.ElementsMatch(t, expected, types)
}

func TestIsValidOwnerType_Valid(t *testing.T) {
	for _, ot := range ValidOwnerTypes() {
		assert.True(t, IsValidOwnerType(ot), "expected %q to be valid", ot)
	}
}

func TestIsValidOwnerType_Invalid(t *testing.T) {
	assert.False(t, IsValidOwnerType("unknown"))
	assert.False(t, IsValidOwnerType(""))
	assert.False(t, IsValidOwnerType("PRODUCT"))
}

// ============================================================================
// MediaFile Struct Tests
// ============================================================================

func TestMediaFile_SizeInBytes(t *testing.T) {
	m := MediaFile{Size: 5 * 1024 * 1024}
	assert.Equal(t, int64(5*1024*1024), m.Size)
}

func TestMediaFile_WithinSizeLimit(t *testing.T) {
	m := MediaFile{Size: MaxFileSize - 1}
	assert.True(t, m.Size < MaxFileSize)
}

func TestMediaFile_ExceedsSizeLimit(t *testing.T) {
	m := MediaFile{Size: MaxFileSize + 1}
	assert.True(t, m.Size > MaxFileSize)
}

func TestMediaFile_MetadataMap(t *testing.T) {
	m := MediaFile{
		Metadata: map[string]any{"width": 800, "height": 600},
	}
	assert.Equal(t, 800, m.Metadata["width"])
	assert.Equal(t, 600, m.Metadata["height"])
}

func TestMediaFile_SortOrder(t *testing.T) {
	m := MediaFile{SortOrder: 2}
	assert.Equal(t, 2, m.SortOrder)
}
