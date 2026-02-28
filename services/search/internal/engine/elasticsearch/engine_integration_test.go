package elasticsearch_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esengine "github.com/utafrali/EcommerceGo/services/search/internal/engine/elasticsearch"
	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
)

// testLogger returns a discard logger suitable for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestEngine creates an Elasticsearch engine for integration tests.
// It skips the test if ELASTICSEARCH_URL is not set.
func newTestEngine(t *testing.T) *esengine.Engine {
	t.Helper()

	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		t.Skip("ELASTICSEARCH_URL not set — skipping Elasticsearch integration tests")
	}

	// Use a unique test index per test run to avoid data conflicts.
	indexName := fmt.Sprintf("test_ecommerce_products_%d", time.Now().UnixNano())

	eng, err := esengine.New(esURL, indexName, testLogger())
	require.NoError(t, err, "failed to create Elasticsearch engine")

	// Cleanup: delete the test index when the test completes.
	t.Cleanup(func() {
		_ = eng.DeleteIndex(context.Background())
	})

	return eng
}

func newTestProduct(name, description string, price int64) domain.SearchableProduct {
	now := time.Now().UTC()
	return domain.SearchableProduct{
		ID:           uuid.New().String(),
		Name:         name,
		Slug:         "test-slug-" + uuid.New().String(),
		Description:  description,
		CategoryID:   "cat-1",
		CategoryName: "Electronics",
		BrandID:      "brand-1",
		BrandName:    "Acme",
		BasePrice:    price,
		Currency:     "USD",
		Status:       "published",
		ImageURL:     "https://example.com/image.jpg",
		Tags:         []string{"test"},
		Attributes:   map[string]string{"color": "red"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestES_Ping(t *testing.T) {
	eng := newTestEngine(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := eng.Ping(ctx)
	assert.NoError(t, err)
}

func TestES_IndexAndSearch(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p := newTestProduct("Wireless Bluetooth Headphones", "High quality noise cancelling headphones", 9999)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "bluetooth",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p.ID, result.Products[0].ID)
}

func TestES_IndexUpdatesExisting(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p := newTestProduct("Original Product", "Original description", 1000)
	require.NoError(t, eng.Index(ctx, &p))

	p.Name = "Updated Product"
	p.BasePrice = 2000
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "updated product",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, int64(2000), result.Products[0].BasePrice)
}

func TestES_Delete(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p := newTestProduct("Deletable Product", "Will be deleted", 999)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{Query: "deletable", Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)

	require.NoError(t, eng.Delete(ctx, p.ID))

	result, err = eng.Search(ctx, &domain.SearchQuery{Query: "deletable", Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
}

func TestES_DeleteNonExistent(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	err := eng.Delete(ctx, "non-existent-id")
	assert.NoError(t, err)
}

func TestES_BulkIndex(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	products := []domain.SearchableProduct{
		newTestProduct("Bulk Item Alpha", "First bulk item", 100),
		newTestProduct("Bulk Item Beta", "Second bulk item", 200),
		newTestProduct("Bulk Item Gamma", "Third bulk item", 300),
	}

	require.NoError(t, eng.BulkIndex(ctx, products))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "bulk item",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
}

func TestES_BulkIndex_Empty(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	err := eng.BulkIndex(ctx, []domain.SearchableProduct{})
	assert.NoError(t, err)
}

func TestES_FilterByCategory(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Laptop", "A fast laptop", 99999)
	p1.CategoryID = "cat-electronics"

	p2 := newTestProduct("Laptop Bag", "A nice bag for laptops", 2999)
	p2.CategoryID = "cat-accessories"

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2}))

	catID := "cat-electronics"
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:      "laptop",
		CategoryID: &catID,
		Page:       1,
		PerPage:    20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p1.ID, result.Products[0].ID)
}

func TestES_FilterByBrand(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Running Shoes Nike", "Comfortable running shoes", 8999)
	p1.BrandID = "brand-nike"

	p2 := newTestProduct("Running Shoes Adidas", "Professional running shoes", 12999)
	p2.BrandID = "brand-adidas"

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2}))

	brandID := "brand-nike"
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "running shoes",
		BrandID: &brandID,
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p1.ID, result.Products[0].ID)
}

func TestES_FilterByPriceRange(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Budget Phone", "A budget smartphone", 19999)
	p2 := newTestProduct("Mid Phone", "A mid-range smartphone", 49999)
	p3 := newTestProduct("Premium Phone", "A premium smartphone", 99999)

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2, p3}))

	minPrice := int64(20000)
	maxPrice := int64(60000)
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:    "phone",
		MinPrice: &minPrice,
		MaxPrice: &maxPrice,
		Page:     1,
		PerPage:  20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p2.ID, result.Products[0].ID)
}

func TestES_FilterByStatus(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Published Widget", "A published widget", 999)
	p1.Status = "published"

	p2 := newTestProduct("Draft Widget", "A draft widget", 999)
	p2.Status = "draft"

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2}))

	status := "published"
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "widget",
		Status:  &status,
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p1.ID, result.Products[0].ID)
}

func TestES_SortByPriceAsc(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Sort Item A", "An item", 5000)
	p2 := newTestProduct("Sort Item B", "An item", 1000)
	p3 := newTestProduct("Sort Item C", "An item", 3000)

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2, p3}))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "sort item",
		SortBy:  domain.SortPriceAsc,
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, int64(1000), result.Products[0].BasePrice)
	assert.Equal(t, int64(3000), result.Products[1].BasePrice)
	assert.Equal(t, int64(5000), result.Products[2].BasePrice)
}

func TestES_SortByPriceDesc(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Desc Sort Item A", "An item", 5000)
	p2 := newTestProduct("Desc Sort Item B", "An item", 1000)
	p3 := newTestProduct("Desc Sort Item C", "An item", 3000)

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2, p3}))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "desc sort item",
		SortBy:  domain.SortPriceDesc,
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, int64(5000), result.Products[0].BasePrice)
	assert.Equal(t, int64(3000), result.Products[1].BasePrice)
	assert.Equal(t, int64(1000), result.Products[2].BasePrice)
}

func TestES_Pagination(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	var products []domain.SearchableProduct
	for i := 0; i < 5; i++ {
		products = append(products, newTestProduct("Paginated Item", "A test item for pagination", int64(1000*(i+1))))
	}
	require.NoError(t, eng.BulkIndex(ctx, products))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "paginated item",
		Page:    1,
		PerPage: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Products, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PerPage)
}

func TestES_EmptyQuery_ReturnsAll(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p1 := newTestProduct("Empty Query Alpha", "First product", 100)
	p2 := newTestProduct("Empty Query Beta", "Second product", 200)

	require.NoError(t, eng.BulkIndex(ctx, []domain.SearchableProduct{p1, p2}))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, 2)
}

func TestES_Suggest(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p := newTestProduct("Suggestible Laptop", "A great laptop for suggestions", 50000)
	p.Status = "published"
	require.NoError(t, eng.Index(ctx, &p))

	suggestions, err := eng.Suggest(ctx, "Suggestible", 5)
	require.NoError(t, err)
	// Suggestions may or may not match depending on analyzer; at minimum no error.
	assert.NotNil(t, suggestions)
}

func TestES_SearchReturnsMetadata(t *testing.T) {
	eng := newTestEngine(t)
	ctx := context.Background()

	p := newTestProduct("Metadata Check Product", "Testing metadata fields in ES", 5555)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "metadata check",
		Page:    1,
		PerPage: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PerPage)
	assert.GreaterOrEqual(t, result.TookMs, int64(0))
}

func TestES_DefaultIndexName(t *testing.T) {
	assert.Equal(t, "ecommerce_products", esengine.DefaultIndexName)
}
