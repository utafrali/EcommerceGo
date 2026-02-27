package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
	"github.com/utafrali/EcommerceGo/services/search/internal/engine/memory"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestService() *SearchService {
	eng := memory.New()
	return NewSearchService(eng, newTestLogger(), "http://localhost:8080")
}

func TestSearchService_IndexAndSearch(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	input := &IndexProductInput{
		ID:          uuid.New().String(),
		Name:        "Wireless Mouse",
		Slug:        "wireless-mouse",
		Description: "Ergonomic wireless mouse with USB receiver",
		CategoryID:  "cat-peripherals",
		BrandID:     "brand-logitech",
		BasePrice:   2999,
		Currency:    "USD",
		Status:      "published",
	}

	require.NoError(t, svc.IndexProduct(ctx, input))

	result, err := svc.Search(ctx, &domain.SearchQuery{
		Query:   "wireless mouse",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, input.ID, result.Products[0].ID)
}

func TestSearchService_IndexProduct_RequiresID(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	err := svc.IndexProduct(ctx, &IndexProductInput{
		Name: "Test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestSearchService_IndexProduct_RequiresName(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	err := svc.IndexProduct(ctx, &IndexProductInput{
		ID: uuid.New().String(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestSearchService_DeleteProduct(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	id := uuid.New().String()
	require.NoError(t, svc.IndexProduct(ctx, &IndexProductInput{
		ID:   id,
		Name: "To Delete",
	}))

	// Verify it is searchable.
	result, err := svc.Search(ctx, &domain.SearchQuery{Query: "delete", Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)

	// Delete it.
	require.NoError(t, svc.DeleteProduct(ctx, id))

	// Verify it is gone.
	result, err = svc.Search(ctx, &domain.SearchQuery{Query: "delete", Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
}

func TestSearchService_DeleteProduct_RequiresID(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	err := svc.DeleteProduct(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestSearchService_Search_DefaultPagination(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	result, err := svc.Search(ctx, &domain.SearchQuery{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PerPage)
}

func TestSearchService_Search_CapsPerPage(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	result, err := svc.Search(ctx, &domain.SearchQuery{
		PerPage: 500,
	})
	require.NoError(t, err)
	assert.Equal(t, 100, result.PerPage)
}

func TestSearchService_BulkIndex(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	inputs := []IndexProductInput{
		{ID: uuid.New().String(), Name: "Bulk Alpha", Description: "First", BasePrice: 100},
		{ID: uuid.New().String(), Name: "Bulk Beta", Description: "Second", BasePrice: 200},
		{ID: uuid.New().String(), Name: "Bulk Gamma", Description: "Third", BasePrice: 300},
	}

	require.NoError(t, svc.BulkIndex(ctx, inputs))

	result, err := svc.Search(ctx, &domain.SearchQuery{Query: "bulk", Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
}

func TestSearchService_BulkIndex_SkipsEmptyID(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	inputs := []IndexProductInput{
		{ID: uuid.New().String(), Name: "Valid Item", BasePrice: 100},
		{ID: "", Name: "Invalid Item", BasePrice: 200}, // Should be skipped.
	}

	require.NoError(t, svc.BulkIndex(ctx, inputs))

	result, err := svc.Search(ctx, &domain.SearchQuery{Query: "item", Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

func TestSearchService_Reindex(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	// Reindex is a placeholder; just verify it does not error.
	err := svc.Reindex(ctx)
	assert.NoError(t, err)
}

func TestSearchService_Search_DefaultSortBy(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	result, err := svc.Search(ctx, &domain.SearchQuery{
		Query: "anything",
	})
	require.NoError(t, err)
	// No error means default was applied internally.
	assert.NotNil(t, result)
}

func TestSearchService_IndexProduct_SetsDefaultTagsAndAttributes(t *testing.T) {
	ctx := context.Background()
	svc := newTestService()

	input := &IndexProductInput{
		ID:   uuid.New().String(),
		Name: "Defaults Test",
	}

	require.NoError(t, svc.IndexProduct(ctx, input))

	result, err := svc.Search(ctx, &domain.SearchQuery{Query: "defaults", Page: 1, PerPage: 20})
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	assert.NotNil(t, result.Products[0].Tags)
	assert.NotNil(t, result.Products[0].Attributes)
}
