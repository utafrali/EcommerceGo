package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/utafrali/EcommerceGo/pkg/errors"
	"github.com/utafrali/EcommerceGo/services/product/internal/domain"
	"github.com/utafrali/EcommerceGo/services/product/internal/event"
	"github.com/utafrali/EcommerceGo/services/product/internal/repository"
)

// slugRegexp matches characters not allowed in a slug.
var slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)

// ProductService implements the business logic for product operations.
type ProductService struct {
	repo     repository.ProductRepository
	producer *event.Producer
	logger   *slog.Logger
}

// NewProductService creates a new product service.
func NewProductService(repo repository.ProductRepository, producer *event.Producer, logger *slog.Logger) *ProductService {
	return &ProductService{
		repo:     repo,
		producer: producer,
		logger:   logger,
	}
}

// CreateProductInput holds the parameters for creating a product.
type CreateProductInput struct {
	Name        string
	Description string
	BrandID     *string
	CategoryID  *string
	BasePrice   int64
	Currency    string
	Metadata    map[string]any
}

// UpdateProductInput holds the parameters for updating a product.
type UpdateProductInput struct {
	Name        *string
	Description *string
	BrandID     *string
	CategoryID  *string
	Status      *string
	BasePrice   *int64
	Currency    *string
	Metadata    map[string]any
}

// CreateProduct creates a new product with the given input.
func (s *ProductService) CreateProduct(ctx context.Context, input *CreateProductInput) (*domain.Product, error) {
	if input.Name == "" {
		return nil, apperrors.InvalidInput("product name is required")
	}
	if input.BasePrice < 0 {
		return nil, apperrors.InvalidInput("base price must not be negative")
	}
	if len(input.Currency) != 3 {
		return nil, apperrors.InvalidInput("currency must be a 3-letter ISO code")
	}

	now := time.Now().UTC()
	product := &domain.Product{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Slug:        generateSlug(input.Name),
		Description: input.Description,
		BrandID:     input.BrandID,
		CategoryID:  input.CategoryID,
		Status:      domain.ProductStatusDraft,
		BasePrice:   input.BasePrice,
		Currency:    strings.ToUpper(input.Currency),
		Metadata:    input.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if product.Metadata == nil {
		product.Metadata = make(map[string]any)
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	if err := s.producer.PublishProductCreated(ctx, product); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish product.created event",
			slog.String("product_id", product.ID),
			slog.String("error", err.Error()),
		)
		// Do not fail the operation if event publishing fails.
	}

	s.logger.InfoContext(ctx, "product created",
		slog.String("product_id", product.ID),
		slog.String("slug", product.Slug),
	)

	return product, nil
}

// GetProduct retrieves a product by its ID.
func (s *ProductService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return product, nil
}

// GetProductBySlug retrieves a product by its slug.
func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	product, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get product by slug: %w", err)
	}
	return product, nil
}

// GetProductDetail retrieves a product by ID and enriches it with images,
// variants, category, and brand information.
func (s *ProductService) GetProductDetail(ctx context.Context, id string) (*domain.ProductDetail, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return s.enrichProduct(ctx, product)
}

// GetProductDetailBySlug retrieves a product by slug and enriches it with
// images, variants, category, and brand information.
func (s *ProductService) GetProductDetailBySlug(ctx context.Context, slug string) (*domain.ProductDetail, error) {
	product, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get product by slug: %w", err)
	}
	return s.enrichProduct(ctx, product)
}

// enrichProduct loads images, variants, category, and brand for a product.
func (s *ProductService) enrichProduct(ctx context.Context, product *domain.Product) (*domain.ProductDetail, error) {
	detail := &domain.ProductDetail{
		Product: *product,
	}

	images, err := s.repo.GetImages(ctx, product.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load product images",
			slog.String("product_id", product.ID),
			slog.String("error", err.Error()),
		)
		images = []domain.ProductImage{}
	}
	detail.Images = images

	variants, err := s.repo.GetVariants(ctx, product.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load product variants",
			slog.String("product_id", product.ID),
			slog.String("error", err.Error()),
		)
		variants = []domain.ProductVariant{}
	}
	detail.Variants = variants

	if product.CategoryID != nil {
		category, err := s.repo.GetCategory(ctx, *product.CategoryID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to load product category",
				slog.String("product_id", product.ID),
				slog.String("category_id", *product.CategoryID),
				slog.String("error", err.Error()),
			)
		} else {
			detail.Category = category
		}
	}

	if product.BrandID != nil {
		brand, err := s.repo.GetBrand(ctx, *product.BrandID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to load product brand",
				slog.String("product_id", product.ID),
				slog.String("brand_id", *product.BrandID),
				slog.String("error", err.Error()),
			)
		} else {
			detail.Brand = brand
		}
	}

	return detail, nil
}

// ListProducts returns a filtered, paginated list of products with primary images.
func (s *ProductService) ListProducts(ctx context.Context, filter repository.ProductFilter) ([]domain.ProductListItem, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	products, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}

	// Batch-fetch primary images for all returned products.
	productIDs := make([]string, len(products))
	for i, p := range products {
		productIDs[i] = p.ID
	}

	primaryImages, err := s.repo.GetPrimaryImages(ctx, productIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load primary images for product list",
			slog.String("error", err.Error()),
		)
		primaryImages = map[string]domain.ProductImage{}
	}

	items := make([]domain.ProductListItem, len(products))
	for i, p := range products {
		items[i] = domain.ProductListItem{Product: p}
		if img, ok := primaryImages[p.ID]; ok {
			items[i].PrimaryImage = &img
		}
	}

	return items, total, nil
}

// UpdateProduct applies partial updates to an existing product.
func (s *ProductService) UpdateProduct(ctx context.Context, id string, input *UpdateProductInput) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product for update: %w", err)
	}

	if input.Name != nil {
		if *input.Name == "" {
			return nil, apperrors.InvalidInput("product name must not be empty")
		}
		product.Name = *input.Name
		product.Slug = generateSlug(*input.Name)
	}

	if input.Description != nil {
		product.Description = *input.Description
	}

	if input.BrandID != nil {
		product.BrandID = input.BrandID
	}

	if input.CategoryID != nil {
		product.CategoryID = input.CategoryID
	}

	if input.Status != nil {
		if !domain.IsValidStatus(*input.Status) {
			return nil, apperrors.InvalidInput(fmt.Sprintf("invalid status %q, must be one of: %s", *input.Status, strings.Join(domain.ValidStatuses(), ", ")))
		}
		product.Status = *input.Status
	}

	if input.BasePrice != nil {
		if *input.BasePrice < 0 {
			return nil, apperrors.InvalidInput("base price must not be negative")
		}
		product.BasePrice = *input.BasePrice
	}

	if input.Currency != nil {
		if len(*input.Currency) != 3 {
			return nil, apperrors.InvalidInput("currency must be a 3-letter ISO code")
		}
		product.Currency = strings.ToUpper(*input.Currency)
	}

	if input.Metadata != nil {
		product.Metadata = input.Metadata
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	if err := s.producer.PublishProductUpdated(ctx, product); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish product.updated event",
			slog.String("product_id", product.ID),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "product updated",
		slog.String("product_id", product.ID),
		slog.String("slug", product.Slug),
	)

	return product, nil
}

// DeleteProduct removes a product by its ID.
func (s *ProductService) DeleteProduct(ctx context.Context, id string) error {
	// Verify the product exists before deleting.
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("get product for delete: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	if err := s.producer.PublishProductDeleted(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to publish product.deleted event",
			slog.String("product_id", id),
			slog.String("error", err.Error()),
		)
	}

	s.logger.InfoContext(ctx, "product deleted",
		slog.String("product_id", id),
	)

	return nil
}

// generateSlug creates a URL-friendly slug from the given name.
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugRegexp.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
