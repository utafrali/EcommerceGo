package memory

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
)

func newTestProduct(name, description string, price int64) domain.SearchableProduct {
	now := time.Now().UTC()
	return domain.SearchableProduct{
		ID:           uuid.New().String(),
		Name:         name,
		Slug:         "test-slug",
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

func TestEngine_SearchByText_Match(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Wireless Bluetooth Headphones", "High quality noise canceling headphones", 9999)
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

func TestEngine_SearchByText_NoMatch(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Wireless Bluetooth Headphones", "High quality headphones", 9999)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "keyboard",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Products)
}

func TestEngine_SearchByText_MatchesDescription(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Premium Audio Device", "Noise canceling bluetooth headphones with deep bass", 14999)
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

func TestEngine_SearchByText_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Wireless BLUETOOTH Headphones", "Audio device", 9999)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "bluetooth",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

func TestEngine_FilterByCategory(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Laptop", "A fast laptop", 99999)
	p1.CategoryID = "cat-electronics"

	p2 := newTestProduct("Laptop Bag", "A nice bag for laptops", 2999)
	p2.CategoryID = "cat-accessories"

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))

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

func TestEngine_FilterByBrand(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Running Shoes", "Comfortable running shoes", 8999)
	p1.BrandID = "brand-nike"

	p2 := newTestProduct("Running Shoes Pro", "Professional running shoes", 12999)
	p2.BrandID = "brand-adidas"

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))

	brandID := "brand-nike"
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "running",
		BrandID: &brandID,
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p1.ID, result.Products[0].ID)
}

func TestEngine_FilterByPriceRange(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Budget Phone", "A budget smartphone", 19999)
	p2 := newTestProduct("Mid Phone", "A mid-range smartphone", 49999)
	p3 := newTestProduct("Premium Phone", "A premium smartphone", 99999)

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))
	require.NoError(t, eng.Index(ctx, &p3))

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

func TestEngine_FilterByStatus(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Active Widget", "A widget that is active", 999)
	p1.Status = "published"

	p2 := newTestProduct("Draft Widget", "A widget in draft", 999)
	p2.Status = "draft"

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))

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

func TestEngine_SortByPriceAsc(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Item A", "An item", 5000)
	p2 := newTestProduct("Item B", "An item", 1000)
	p3 := newTestProduct("Item C", "An item", 3000)

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))
	require.NoError(t, eng.Index(ctx, &p3))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "item",
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

func TestEngine_SortByPriceDesc(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Item A", "An item", 5000)
	p2 := newTestProduct("Item B", "An item", 1000)
	p3 := newTestProduct("Item C", "An item", 3000)

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))
	require.NoError(t, eng.Index(ctx, &p3))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "item",
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

func TestEngine_SortByNewest(t *testing.T) {
	ctx := context.Background()
	eng := New()

	now := time.Now().UTC()

	p1 := newTestProduct("Old Item", "An item", 1000)
	p1.CreatedAt = now.Add(-48 * time.Hour)

	p2 := newTestProduct("New Item", "An item", 2000)
	p2.CreatedAt = now

	p3 := newTestProduct("Middle Item", "An item", 1500)
	p3.CreatedAt = now.Add(-24 * time.Hour)

	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))
	require.NoError(t, eng.Index(ctx, &p3))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "item",
		SortBy:  domain.SortNewest,
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, p2.ID, result.Products[0].ID)
	assert.Equal(t, p3.ID, result.Products[1].ID)
	assert.Equal(t, p1.ID, result.Products[2].ID)
}

func TestEngine_Pagination(t *testing.T) {
	ctx := context.Background()
	eng := New()

	// Index 5 products.
	for i := 0; i < 5; i++ {
		p := newTestProduct("Widget", "A test widget", int64(1000*(i+1)))
		require.NoError(t, eng.Index(ctx, &p))
	}

	// Page 1, 2 per page.
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "widget",
		Page:    1,
		PerPage: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Products, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PerPage)

	// Page 3, 2 per page (only 1 item left).
	result, err = eng.Search(ctx, &domain.SearchQuery{
		Query:   "widget",
		Page:    3,
		PerPage: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Products, 1)

	// Page beyond results.
	result, err = eng.Search(ctx, &domain.SearchQuery{
		Query:   "widget",
		Page:    10,
		PerPage: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Empty(t, result.Products)
}

func TestEngine_IndexAndSearch_RoundTrip(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Unique Gadget XYZ", "A one-of-a-kind gadget", 4999)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "unique gadget",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, p.ID, result.Products[0].ID)
	assert.Equal(t, "Unique Gadget XYZ", result.Products[0].Name)
	assert.Equal(t, int64(4999), result.Products[0].BasePrice)
}

func TestEngine_DeleteAndSearch(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Deletable Product", "Will be deleted", 999)
	require.NoError(t, eng.Index(ctx, &p))

	// Verify it exists.
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "deletable",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)

	// Delete it.
	require.NoError(t, eng.Delete(ctx, p.ID))

	// Verify it is gone.
	result, err = eng.Search(ctx, &domain.SearchQuery{
		Query:   "deletable",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Products)
}

func TestEngine_BulkIndex(t *testing.T) {
	ctx := context.Background()
	eng := New()

	products := []domain.SearchableProduct{
		newTestProduct("Bulk Item One", "First bulk item", 100),
		newTestProduct("Bulk Item Two", "Second bulk item", 200),
		newTestProduct("Bulk Item Three", "Third bulk item", 300),
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

func TestEngine_EmptyQuery_ReturnsAll(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p1 := newTestProduct("Alpha", "First product", 100)
	p2 := newTestProduct("Beta", "Second product", 200)
	require.NoError(t, eng.Index(ctx, &p1))
	require.NoError(t, eng.Index(ctx, &p2))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
}

func TestEngine_IndexUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Original Name", "Original description", 1000)
	require.NoError(t, eng.Index(ctx, &p))

	// Update the same product (same ID).
	p.Name = "Updated Name"
	p.BasePrice = 2000
	require.NoError(t, eng.Index(ctx, &p))

	// Search for original name should not find it.
	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "original name",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)

	// Search for updated name should find it.
	result, err = eng.Search(ctx, &domain.SearchQuery{
		Query:   "updated name",
		Page:    1,
		PerPage: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, int64(2000), result.Products[0].BasePrice)
}

func TestEngine_DeleteNonExistent(t *testing.T) {
	ctx := context.Background()
	eng := New()

	// Deleting a non-existent ID should not error.
	err := eng.Delete(ctx, "non-existent-id")
	assert.NoError(t, err)
}

func TestEngine_SearchReturnsMetadata(t *testing.T) {
	ctx := context.Background()
	eng := New()

	p := newTestProduct("Metadata Test", "Testing metadata fields", 5555)
	require.NoError(t, eng.Index(ctx, &p))

	result, err := eng.Search(ctx, &domain.SearchQuery{
		Query:   "metadata",
		Page:    1,
		PerPage: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PerPage)
	assert.GreaterOrEqual(t, result.TookMs, int64(0))
}
