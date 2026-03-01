package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/utafrali/EcommerceGo/pkg/database"
	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository"
)

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := database.NewMockPool()
	require.NoError(t, err)
	return mock
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func int64Ptr(n int64) *int64 { return &n }

var now = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

// ─── Product column definitions ─────────────────────────────────────────────

var productColumns = []string{
	"id", "name", "slug", "description", "brand_id", "category_id",
	"status", "base_price", "currency", "metadata", "created_at", "updated_at",
}

var productColumnsWithCount = []string{
	"id", "name", "slug", "description", "brand_id", "category_id",
	"status", "base_price", "currency", "metadata", "created_at", "updated_at",
	"total_count",
}

func sampleProduct() domain.Product {
	meta := map[string]any{"color": "red"}
	return domain.Product{
		ID:          "prod-1",
		Name:        "Widget",
		Slug:        "widget",
		Description: "A fine widget",
		BrandID:     strPtr("brand-1"),
		CategoryID:  strPtr("cat-1"),
		Status:      domain.ProductStatusPublished,
		BasePrice:   9999,
		Currency:    "USD",
		Metadata:    meta,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func productRow(p domain.Product) []any {
	metaJSON, _ := json.Marshal(p.Metadata)
	return []any{
		p.ID, p.Name, p.Slug, p.Description, p.BrandID, p.CategoryID,
		p.Status, p.BasePrice, p.Currency, metaJSON, p.CreatedAt, p.UpdatedAt,
	}
}

// ─── Banner column definitions ──────────────────────────────────────────────

var bannerColumns = []string{
	"id", "title", "subtitle", "image_url", "link_url", "link_type",
	"position", "sort_order", "is_active", "starts_at", "ends_at",
	"created_at", "updated_at",
}

var bannerColumnsWithCount = []string{
	"id", "title", "subtitle", "image_url", "link_url", "link_type",
	"position", "sort_order", "is_active", "starts_at", "ends_at",
	"created_at", "updated_at", "total_count",
}

func sampleBanner() domain.Banner {
	startsAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	endsAt := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	return domain.Banner{
		ID:        "banner-1",
		Title:     "Summer Sale",
		Subtitle:  strPtr("Up to 50% off"),
		ImageURL:  "https://cdn.example.com/banner.jpg",
		LinkURL:   "/sale",
		LinkType:  domain.BannerLinkTypeInternal,
		Position:  domain.BannerPositionHeroSlider,
		SortOrder: 1,
		IsActive:  true,
		StartsAt:  &startsAt,
		EndsAt:    &endsAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func bannerRow(b domain.Banner) []any {
	return []any{
		b.ID, b.Title, b.Subtitle, b.ImageURL, b.LinkURL, b.LinkType,
		b.Position, b.SortOrder, b.IsActive, b.StartsAt, b.EndsAt,
		b.CreatedAt, b.UpdatedAt,
	}
}

// ─── Brand column definitions ───────────────────────────────────────────────

var brandColumns = []string{
	"id", "name", "slug", "logo_url", "created_at", "updated_at",
}

func sampleBrand() domain.Brand {
	return domain.Brand{
		ID:        "brand-1",
		Name:      "Acme",
		Slug:      "acme",
		LogoURL:   strPtr("https://cdn.example.com/acme.png"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func brandRow(b domain.Brand) []any {
	return []any{b.ID, b.Name, b.Slug, b.LogoURL, b.CreatedAt, b.UpdatedAt}
}

// ─── Category column definitions ────────────────────────────────────────────

var catColumns = []string{
	"id", "name", "slug", "parent_id", "sort_order", "is_active",
	"image_url", "icon_url", "description", "level", "product_count",
	"created_at", "updated_at",
}

func sampleCategory() domain.Category {
	return domain.Category{
		ID:           "cat-1",
		Name:         "Electronics",
		Slug:         "electronics",
		ParentID:     nil,
		SortOrder:    0,
		IsActive:     true,
		ImageURL:     strPtr("https://cdn.example.com/electronics.jpg"),
		IconURL:      strPtr("https://cdn.example.com/electronics-icon.svg"),
		Description:  strPtr("Electronic goods"),
		Level:        0,
		ProductCount: 42,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func categoryRow(c domain.Category) []any {
	return []any{
		c.ID, c.Name, c.Slug, c.ParentID, c.SortOrder, c.IsActive,
		c.ImageURL, c.IconURL, c.Description, c.Level, c.ProductCount,
		c.CreatedAt, c.UpdatedAt,
	}
}

// ─── Review column definitions ──────────────────────────────────────────────

var reviewColumnsWithCount = []string{
	"id", "product_id", "user_id", "rating", "title", "body",
	"created_at", "updated_at", "total_count",
}

func sampleReview() domain.Review {
	return domain.Review{
		ID:        "review-1",
		ProductID: "prod-1",
		UserID:    "user-1",
		Rating:    5,
		Title:     "Amazing product",
		Body:      "Highly recommended.",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func reviewRow(r domain.Review) []any {
	return []any{
		r.ID, r.ProductID, r.UserID, r.Rating, r.Title, r.Body,
		r.CreatedAt, r.UpdatedAt,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ProductRepository
// ─────────────────────────────────────────────────────────────────────────────

func TestProductRepository_Create_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	metaJSON, _ := json.Marshal(p.Metadata)

	mock.ExpectExec("INSERT INTO products").
		WithArgs(
			p.ID, p.Name, p.Slug, p.Description, p.BrandID, p.CategoryID,
			p.Status, p.BasePrice, p.Currency, metaJSON, p.CreatedAt, p.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), &p)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_Create_UniqueViolation(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	metaJSON, _ := json.Marshal(p.Metadata)

	mock.ExpectExec("INSERT INTO products").
		WithArgs(
			p.ID, p.Name, p.Slug, p.Description, p.BrandID, p.CategoryID,
			p.Status, p.BasePrice, p.Currency, metaJSON, p.CreatedAt, p.UpdatedAt,
		).
		WillReturnError(errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.Create(context.Background(), &p)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_GetByID_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	mock.ExpectQuery("SELECT .+ FROM products WHERE id").
		WithArgs(p.ID).
		WillReturnRows(
			pgxmock.NewRows(productColumns).AddRow(productRow(p)...),
		)

	result, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, p.Name, result.Name)
	assert.Equal(t, p.Slug, result.Slug)
	assert.Equal(t, p.BasePrice, result.BasePrice)
	assert.Equal(t, p.BrandID, result.BrandID)
	assert.Equal(t, p.CategoryID, result.CategoryID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_GetByID_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM products WHERE id").
		WithArgs("missing-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "missing-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_GetBySlug_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	mock.ExpectQuery("SELECT .+ FROM products WHERE slug").
		WithArgs(p.Slug).
		WillReturnRows(
			pgxmock.NewRows(productColumns).AddRow(productRow(p)...),
		)

	result, err := repo.GetBySlug(context.Background(), p.Slug)
	require.NoError(t, err)
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, p.Slug, result.Slug)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_List_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	row := append(productRow(p), 1) // total_count = 1

	filter := repository.ProductFilter{
		Page:    1,
		PerPage: 20,
	}

	mock.ExpectQuery("SELECT .+ FROM products").
		WithArgs(20, 0). // limit, offset
		WillReturnRows(
			pgxmock.NewRows(productColumnsWithCount).AddRow(row...),
		)

	products, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, p.ID, products[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_List_WithFilters(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	row := append(productRow(p), 1)

	filter := repository.ProductFilter{
		CategoryID: strPtr("cat-1"),
		Status:     strPtr("published"),
		MinPrice:   int64Ptr(5000),
		Page:       1,
		PerPage:    10,
	}

	// category_id=$1, status=$2, base_price>=$3, LIMIT $4 OFFSET $5
	mock.ExpectQuery("SELECT .+ FROM products").
		WithArgs("cat-1", "published", int64(5000), 10, 0).
		WillReturnRows(
			pgxmock.NewRows(productColumnsWithCount).AddRow(row...),
		)

	products, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, 1, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_Update_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	metaJSON, _ := json.Marshal(p.Metadata)

	mock.ExpectExec("UPDATE products").
		WithArgs(
			p.Name, p.Slug, p.Description, p.BrandID, p.CategoryID,
			p.Status, p.BasePrice, p.Currency, metaJSON,
			pgxmock.AnyArg(), // updated_at is set inside Update
			p.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), &p)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_Update_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	p := sampleProduct()
	p.ID = "nonexistent-id"
	metaJSON, _ := json.Marshal(p.Metadata)

	mock.ExpectExec("UPDATE products").
		WithArgs(
			p.Name, p.Slug, p.Description, p.BrandID, p.CategoryID,
			p.Status, p.BasePrice, p.Currency, metaJSON,
			pgxmock.AnyArg(),
			p.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), &p)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_Delete_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	mock.ExpectExec("DELETE FROM products WHERE").
		WithArgs("prod-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(context.Background(), "prod-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_Delete_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewProductRepository(mock)

	mock.ExpectExec("DELETE FROM products WHERE").
		WithArgs("missing-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err := repo.Delete(context.Background(), "missing-id")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─────────────────────────────────────────────────────────────────────────────
// BannerRepository
// ─────────────────────────────────────────────────────────────────────────────

func TestBannerRepository_Create_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	b := sampleBanner()
	mock.ExpectExec("INSERT INTO banners").
		WithArgs(
			b.ID, b.Title, b.Subtitle, b.ImageURL, b.LinkURL, b.LinkType,
			b.Position, b.SortOrder, b.IsActive, b.StartsAt, b.EndsAt,
			b.CreatedAt, b.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), &b)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_GetByID_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	b := sampleBanner()
	mock.ExpectQuery("SELECT .+ FROM banners WHERE id").
		WithArgs(b.ID).
		WillReturnRows(
			pgxmock.NewRows(bannerColumns).AddRow(bannerRow(b)...),
		)

	result, err := repo.GetByID(context.Background(), b.ID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, result.ID)
	assert.Equal(t, b.Title, result.Title)
	assert.Equal(t, b.Position, result.Position)
	assert.Equal(t, b.IsActive, result.IsActive)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_GetByID_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM banners WHERE id").
		WithArgs("missing-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "missing-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_Update_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	b := sampleBanner()
	mock.ExpectExec("UPDATE banners").
		WithArgs(
			b.Title, b.Subtitle, b.ImageURL, b.LinkURL, b.LinkType,
			b.Position, b.SortOrder, b.IsActive, b.StartsAt, b.EndsAt,
			pgxmock.AnyArg(), // updated_at set inside Update
			b.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), &b)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_Update_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	b := sampleBanner()
	b.ID = "nonexistent-id"
	mock.ExpectExec("UPDATE banners").
		WithArgs(
			b.Title, b.Subtitle, b.ImageURL, b.LinkURL, b.LinkType,
			b.Position, b.SortOrder, b.IsActive, b.StartsAt, b.EndsAt,
			pgxmock.AnyArg(),
			b.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), &b)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_Delete_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	mock.ExpectExec("DELETE FROM banners WHERE").
		WithArgs("banner-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(context.Background(), "banner-1")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_List_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	b := sampleBanner()
	row := append(bannerRow(b), 1)

	filter := domain.BannerFilter{Page: 1, PerPage: 20}

	mock.ExpectQuery("SELECT .+ FROM banners").
		WithArgs(20, 0).
		WillReturnRows(
			pgxmock.NewRows(bannerColumnsWithCount).AddRow(row...),
		)

	banners, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, banners, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, b.ID, banners[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBannerRepository_List_WithFilters(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBannerRepository(mock)

	b := sampleBanner()
	row := append(bannerRow(b), 1)

	filter := domain.BannerFilter{
		Position: strPtr(domain.BannerPositionHeroSlider),
		IsActive: boolPtr(true),
		Page:     1,
		PerPage:  10,
	}

	// position=$1, is_active=$2, LIMIT $3 OFFSET $4
	mock.ExpectQuery("SELECT .+ FROM banners").
		WithArgs(domain.BannerPositionHeroSlider, true, 10, 0).
		WillReturnRows(
			pgxmock.NewRows(bannerColumnsWithCount).AddRow(row...),
		)

	banners, total, err := repo.List(context.Background(), filter)
	require.NoError(t, err)
	assert.Len(t, banners, 1)
	assert.Equal(t, 1, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─────────────────────────────────────────────────────────────────────────────
// BrandRepository
// ─────────────────────────────────────────────────────────────────────────────

func TestBrandRepository_ListAll_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBrandRepository(mock)

	b1 := sampleBrand()
	b2 := domain.Brand{
		ID:        "brand-2",
		Name:      "Globex",
		Slug:      "globex",
		LogoURL:   nil,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mock.ExpectQuery("SELECT .+ FROM brands ORDER BY name").
		WillReturnRows(
			pgxmock.NewRows(brandColumns).
				AddRow(brandRow(b1)...).
				AddRow(brandRow(b2)...),
		)

	brands, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, brands, 2)
	assert.Equal(t, b1.ID, brands[0].ID)
	assert.Equal(t, b2.ID, brands[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBrandRepository_ListAll_Empty(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewBrandRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM brands ORDER BY name").
		WillReturnRows(pgxmock.NewRows(brandColumns))

	brands, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []domain.Brand{}, brands)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─────────────────────────────────────────────────────────────────────────────
// CategoryRepository
// ─────────────────────────────────────────────────────────────────────────────

func TestCategoryRepository_Create_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c := sampleCategory()
	mock.ExpectExec("INSERT INTO categories").
		WithArgs(
			c.ID, c.Name, c.Slug, c.ParentID, c.SortOrder, c.IsActive,
			c.ImageURL, c.IconURL, c.Description, c.Level, c.ProductCount,
			c.CreatedAt, c.UpdatedAt,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), &c)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_Create_UniqueViolation(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c := sampleCategory()
	mock.ExpectExec("INSERT INTO categories").
		WithArgs(
			c.ID, c.Name, c.Slug, c.ParentID, c.SortOrder, c.IsActive,
			c.ImageURL, c.IconURL, c.Description, c.Level, c.ProductCount,
			c.CreatedAt, c.UpdatedAt,
		).
		WillReturnError(errors.New("ERROR: duplicate key value violates unique constraint (SQLSTATE 23505)"))

	err := repo.Create(context.Background(), &c)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrAlreadyExists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_GetByID_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c := sampleCategory()
	mock.ExpectQuery("SELECT .+ FROM categories WHERE id").
		WithArgs(c.ID).
		WillReturnRows(
			pgxmock.NewRows(catColumns).AddRow(categoryRow(c)...),
		)

	result, err := repo.GetByID(context.Background(), c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, c.Name, result.Name)
	assert.Equal(t, c.Slug, result.Slug)
	assert.Equal(t, c.Level, result.Level)
	assert.Equal(t, c.ProductCount, result.ProductCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_GetByID_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM categories WHERE id").
		WithArgs("missing-id").
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetByID(context.Background(), "missing-id")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_Update_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c := sampleCategory()
	mock.ExpectExec("UPDATE categories").
		WithArgs(
			c.Name, c.Slug, c.ParentID, c.SortOrder, c.IsActive,
			c.ImageURL, c.IconURL, c.Description, c.Level, c.ProductCount,
			pgxmock.AnyArg(), // updated_at set inside Update
			c.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Update(context.Background(), &c)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_Update_NotFound(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c := sampleCategory()
	c.ID = "nonexistent-id"
	mock.ExpectExec("UPDATE categories").
		WithArgs(
			c.Name, c.Slug, c.ParentID, c.SortOrder, c.IsActive,
			c.ImageURL, c.IconURL, c.Description, c.Level, c.ProductCount,
			pgxmock.AnyArg(),
			c.ID,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.Update(context.Background(), &c)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_Delete_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c := sampleCategory()

	// Delete calls GetByID first (QueryRow SELECT), then Exec UPDATE reparent, then Exec DELETE.
	mock.ExpectQuery("SELECT .+ FROM categories WHERE id").
		WithArgs(c.ID).
		WillReturnRows(
			pgxmock.NewRows(catColumns).AddRow(categoryRow(c)...),
		)

	mock.ExpectExec("UPDATE categories SET parent_id").
		WithArgs(c.ParentID, c.ID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	mock.ExpectExec("DELETE FROM categories WHERE id").
		WithArgs(c.ID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.Delete(context.Background(), c.ID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCategoryRepository_ListAll_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewCategoryRepository(mock)

	c1 := sampleCategory()
	c2 := domain.Category{
		ID:           "cat-2",
		Name:         "Clothing",
		Slug:         "clothing",
		ParentID:     nil,
		SortOrder:    1,
		IsActive:     true,
		ImageURL:     nil,
		IconURL:      nil,
		Description:  nil,
		Level:        0,
		ProductCount: 10,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectQuery("SELECT .+ FROM categories WHERE is_active").
		WillReturnRows(
			pgxmock.NewRows(catColumns).
				AddRow(categoryRow(c1)...).
				AddRow(categoryRow(c2)...),
		)

	categories, err := repo.ListAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, categories, 2)
	assert.Equal(t, c1.ID, categories[0].ID)
	assert.Equal(t, c2.ID, categories[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─────────────────────────────────────────────────────────────────────────────
// ReviewRepository
// ─────────────────────────────────────────────────────────────────────────────

func TestReviewRepository_Create_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewReviewRepository(mock)

	r := sampleReview()
	mock.ExpectExec("INSERT INTO product_reviews").
		WithArgs(r.ID, r.ProductID, r.UserID, r.Rating, r.Title, r.Body, r.CreatedAt, r.UpdatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.Create(context.Background(), &r)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewRepository_ListByProductID_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewReviewRepository(mock)

	r := sampleReview()
	row := append(reviewRow(r), 1) // total_count = 1

	mock.ExpectQuery("SELECT .+ FROM product_reviews WHERE product_id").
		WithArgs("prod-1", 20, 0). // productID, limit, offset
		WillReturnRows(
			pgxmock.NewRows(reviewColumnsWithCount).AddRow(row...),
		)

	reviews, total, err := repo.ListByProductID(context.Background(), "prod-1", 1, 20)
	require.NoError(t, err)
	assert.Len(t, reviews, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, r.ID, reviews[0].ID)
	assert.Equal(t, r.Rating, reviews[0].Rating)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewRepository_ListByProductID_Empty(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewReviewRepository(mock)

	mock.ExpectQuery("SELECT .+ FROM product_reviews WHERE product_id").
		WithArgs("prod-no-reviews", 20, 0).
		WillReturnRows(pgxmock.NewRows(reviewColumnsWithCount))

	reviews, total, err := repo.ListByProductID(context.Background(), "prod-no-reviews", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, []domain.Review{}, reviews)
	assert.Equal(t, 0, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewRepository_GetSummary_Success(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewReviewRepository(mock)

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("prod-1").
		WillReturnRows(
			pgxmock.NewRows([]string{"avg", "count"}).AddRow(4.56, 12),
		)

	summary, err := repo.GetSummary(context.Background(), "prod-1")
	require.NoError(t, err)
	assert.Equal(t, 4.6, summary.AverageRating) // rounded to 1 decimal
	assert.Equal(t, 12, summary.TotalCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewRepository_GetSummary_NoReviews(t *testing.T) {
	mock := newMock(t)
	defer mock.Close()
	repo := NewReviewRepository(mock)

	mock.ExpectQuery("SELECT COALESCE").
		WithArgs("prod-empty").
		WillReturnRows(
			pgxmock.NewRows([]string{"avg", "count"}).AddRow(0.0, 0),
		)

	summary, err := repo.GetSummary(context.Background(), "prod-empty")
	require.NoError(t, err)
	assert.Equal(t, 0.0, summary.AverageRating)
	assert.Equal(t, 0, summary.TotalCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}
