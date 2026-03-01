package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Product Status Validation Tests
// ============================================================================

func TestValidStatuses_ContainsAll(t *testing.T) {
	statuses := ValidStatuses()
	expected := []string{ProductStatusDraft, ProductStatusPublished, ProductStatusArchived}
	assert.ElementsMatch(t, expected, statuses)
}

func TestIsValidStatus_ValidStatuses(t *testing.T) {
	for _, s := range ValidStatuses() {
		assert.True(t, IsValidStatus(s), "expected %q to be valid", s)
	}
}

func TestIsValidStatus_Invalid(t *testing.T) {
	assert.False(t, IsValidStatus("unknown"))
	assert.False(t, IsValidStatus(""))
	assert.False(t, IsValidStatus("DRAFT"))
}

// ============================================================================
// Product SortBy Validation Tests
// ============================================================================

func TestValidSortByValues_ContainsAll(t *testing.T) {
	values := ValidSortByValues()
	expected := []string{SortByNewest, SortByPriceAsc, SortByPriceDesc, SortByNameAsc, SortByNameDesc}
	assert.ElementsMatch(t, expected, values)
}

func TestIsValidSortBy_ValidValues(t *testing.T) {
	for _, v := range ValidSortByValues() {
		assert.True(t, IsValidSortBy(v), "expected %q to be valid", v)
	}
}

func TestIsValidSortBy_EmptyStringIsValid(t *testing.T) {
	assert.True(t, IsValidSortBy(""))
}

func TestIsValidSortBy_Invalid(t *testing.T) {
	assert.False(t, IsValidSortBy("unknown"))
	assert.False(t, IsValidSortBy("NEWEST"))
}

// ============================================================================
// Product Struct Tests
// ============================================================================

func TestProduct_BasePriceInCents(t *testing.T) {
	p := Product{BasePrice: 9999, Currency: "USD"}
	assert.Equal(t, int64(9999), p.BasePrice)
	assert.Equal(t, "USD", p.Currency)
}

func TestProduct_SlugField(t *testing.T) {
	p := Product{Name: "Test Product", Slug: "test-product"}
	assert.Equal(t, "test-product", p.Slug)
	assert.Equal(t, "Test Product", p.Name)
}

func TestProduct_CategoryAssignment(t *testing.T) {
	catID := "cat-123"
	p := Product{CategoryID: &catID}
	assert.NotNil(t, p.CategoryID)
	assert.Equal(t, "cat-123", *p.CategoryID)
}

func TestProduct_BrandAssignment(t *testing.T) {
	brandID := "brand-456"
	p := Product{BrandID: &brandID}
	assert.NotNil(t, p.BrandID)
	assert.Equal(t, "brand-456", *p.BrandID)
}

func TestProduct_NilCategoryAndBrand(t *testing.T) {
	p := Product{}
	assert.Nil(t, p.CategoryID)
	assert.Nil(t, p.BrandID)
}

// ============================================================================
// ProductVariant Tests
// ============================================================================

func TestProductVariant_NullablePrice(t *testing.T) {
	price := int64(2500)
	v := ProductVariant{Price: &price}
	assert.NotNil(t, v.Price)
	assert.Equal(t, int64(2500), *v.Price)
}

func TestProductVariant_NilPrice(t *testing.T) {
	v := ProductVariant{}
	assert.Nil(t, v.Price)
}

func TestProductVariant_Attributes(t *testing.T) {
	v := ProductVariant{
		Attributes: map[string]string{"color": "red", "size": "L"},
	}
	assert.Equal(t, "red", v.Attributes["color"])
	assert.Equal(t, "L", v.Attributes["size"])
}

// ============================================================================
// ProductImage Tests
// ============================================================================

func TestProductImage_SortOrder(t *testing.T) {
	img := ProductImage{SortOrder: 1, IsPrimary: true}
	assert.Equal(t, 1, img.SortOrder)
	assert.True(t, img.IsPrimary)
}

// ============================================================================
// Banner Tests
// ============================================================================

func TestValidBannerPositions_ContainsAll(t *testing.T) {
	positions := ValidBannerPositions()
	expected := []string{BannerPositionHeroSlider, BannerPositionMidBanner, BannerPositionCategoryBanner}
	assert.ElementsMatch(t, expected, positions)
}

func TestIsValidBannerPosition_Valid(t *testing.T) {
	for _, p := range ValidBannerPositions() {
		assert.True(t, IsValidBannerPosition(p), "expected %q to be valid", p)
	}
}

func TestIsValidBannerPosition_Invalid(t *testing.T) {
	assert.False(t, IsValidBannerPosition("unknown"))
	assert.False(t, IsValidBannerPosition(""))
}

func TestValidBannerLinkTypes_ContainsAll(t *testing.T) {
	types := ValidBannerLinkTypes()
	expected := []string{BannerLinkTypeInternal, BannerLinkTypeExternal}
	assert.ElementsMatch(t, expected, types)
}

func TestIsValidBannerLinkType_Valid(t *testing.T) {
	for _, lt := range ValidBannerLinkTypes() {
		assert.True(t, IsValidBannerLinkType(lt), "expected %q to be valid", lt)
	}
}

func TestIsValidBannerLinkType_Invalid(t *testing.T) {
	assert.False(t, IsValidBannerLinkType("unknown"))
	assert.False(t, IsValidBannerLinkType(""))
}
