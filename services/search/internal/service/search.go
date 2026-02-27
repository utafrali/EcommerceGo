package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
	"github.com/utafrali/EcommerceGo/services/search/internal/engine"
)

// SearchService implements the business logic for search operations.
type SearchService struct {
	engine            engine.SearchEngine
	logger            *slog.Logger
	productServiceURL string
}

// NewSearchService creates a new search service.
func NewSearchService(eng engine.SearchEngine, logger *slog.Logger, productServiceURL string) *SearchService {
	return &SearchService{
		engine:            eng,
		logger:            logger,
		productServiceURL: productServiceURL,
	}
}

// IndexProductInput holds the parameters for indexing a product.
type IndexProductInput struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"`
	Description  string            `json:"description"`
	CategoryID   string            `json:"category_id"`
	CategoryName string            `json:"category_name"`
	BrandID      string            `json:"brand_id"`
	BrandName    string            `json:"brand_name"`
	BasePrice    int64             `json:"base_price"`
	Currency     string            `json:"currency"`
	Status       string            `json:"status"`
	ImageURL     string            `json:"image_url"`
	Tags         []string          `json:"tags"`
	Attributes   map[string]string `json:"attributes"`
}

// IndexProduct indexes a single product in the search engine.
func (s *SearchService) IndexProduct(ctx context.Context, input *IndexProductInput) error {
	if input.ID == "" {
		return fmt.Errorf("index product: id is required")
	}
	if input.Name == "" {
		return fmt.Errorf("index product: name is required")
	}

	now := time.Now().UTC()
	product := &domain.SearchableProduct{
		ID:           input.ID,
		Name:         input.Name,
		Slug:         input.Slug,
		Description:  input.Description,
		CategoryID:   input.CategoryID,
		CategoryName: input.CategoryName,
		BrandID:      input.BrandID,
		BrandName:    input.BrandName,
		BasePrice:    input.BasePrice,
		Currency:     input.Currency,
		Status:       input.Status,
		ImageURL:     input.ImageURL,
		Tags:         input.Tags,
		Attributes:   input.Attributes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if product.Tags == nil {
		product.Tags = []string{}
	}
	if product.Attributes == nil {
		product.Attributes = make(map[string]string)
	}

	if err := s.engine.Index(ctx, product); err != nil {
		return fmt.Errorf("index product: %w", err)
	}

	s.logger.InfoContext(ctx, "product indexed",
		slog.String("product_id", input.ID),
		slog.String("name", input.Name),
	)

	return nil
}

// DeleteProduct removes a product from the search index.
func (s *SearchService) DeleteProduct(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("delete product: id is required")
	}

	if err := s.engine.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	s.logger.InfoContext(ctx, "product deleted from index",
		slog.String("product_id", id),
	)

	return nil
}

// Search executes a search query against the search engine.
func (s *SearchService) Search(ctx context.Context, query *domain.SearchQuery) (*domain.SearchResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PerPage <= 0 {
		query.PerPage = 20
	}
	if query.PerPage > 100 {
		query.PerPage = 100
	}
	if query.SortBy == "" {
		query.SortBy = domain.SortRelevance
	}

	result, err := s.engine.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	s.logger.DebugContext(ctx, "search executed",
		slog.String("query", query.Query),
		slog.Int("total", result.Total),
		slog.Int64("took_ms", result.TookMs),
	)

	return result, nil
}

// BulkIndex indexes multiple products in the search engine.
func (s *SearchService) BulkIndex(ctx context.Context, inputs []IndexProductInput) error {
	products := make([]domain.SearchableProduct, 0, len(inputs))
	now := time.Now().UTC()

	for _, input := range inputs {
		if input.ID == "" {
			continue
		}

		tags := input.Tags
		if tags == nil {
			tags = []string{}
		}
		attrs := input.Attributes
		if attrs == nil {
			attrs = make(map[string]string)
		}

		products = append(products, domain.SearchableProduct{
			ID:           input.ID,
			Name:         input.Name,
			Slug:         input.Slug,
			Description:  input.Description,
			CategoryID:   input.CategoryID,
			CategoryName: input.CategoryName,
			BrandID:      input.BrandID,
			BrandName:    input.BrandName,
			BasePrice:    input.BasePrice,
			Currency:     input.Currency,
			Status:       input.Status,
			ImageURL:     input.ImageURL,
			Tags:         tags,
			Attributes:   attrs,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	if err := s.engine.BulkIndex(ctx, products); err != nil {
		return fmt.Errorf("bulk index: %w", err)
	}

	s.logger.InfoContext(ctx, "bulk index completed",
		slog.Int("count", len(products)),
	)

	return nil
}

// Reindex fetches all products from the product service and reindexes them.
func (s *SearchService) Reindex(ctx context.Context) error {
	s.logger.InfoContext(ctx, "reindex started")

	page := 1
	perPage := 100
	totalIndexed := 0

	for {
		// Fetch products from Product Service via API Gateway
		url := fmt.Sprintf("%s/api/v1/products?page=%d&per_page=%d&status=published", s.productServiceURL, page, perPage)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("reindex: create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("reindex: fetch products page %d: %w", page, err)
		}

		var result struct {
			Data       []json.RawMessage `json:"data"`
			TotalCount int               `json:"total_count"`
			Page       int               `json:"page"`
			TotalPages int               `json:"total_pages"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("reindex: decode page %d: %w", page, err)
		}
		resp.Body.Close()

		if len(result.Data) == 0 {
			break
		}

		// Map raw products to IndexProductInput
		var inputs []IndexProductInput
		for _, raw := range result.Data {
			var p struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Slug        string `json:"slug"`
				Description string `json:"description"`
				BasePrice   int64  `json:"base_price"`
				Currency    string `json:"currency"`
				Status      string `json:"status"`
				Category    *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
				Brand *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"brand"`
				PrimaryImage *struct {
					URL string `json:"url"`
				} `json:"primary_image"`
			}
			if err := json.Unmarshal(raw, &p); err != nil {
				s.logger.WarnContext(ctx, "reindex: skip product", slog.String("error", err.Error()))
				continue
			}

			input := IndexProductInput{
				ID:          p.ID,
				Name:        p.Name,
				Slug:        p.Slug,
				Description: p.Description,
				BasePrice:   p.BasePrice,
				Currency:    p.Currency,
				Status:      p.Status,
			}
			if p.Category != nil {
				input.CategoryID = p.Category.ID
				input.CategoryName = p.Category.Name
			}
			if p.Brand != nil {
				input.BrandID = p.Brand.ID
				input.BrandName = p.Brand.Name
			}
			if p.PrimaryImage != nil {
				input.ImageURL = p.PrimaryImage.URL
			}
			inputs = append(inputs, input)
		}

		if err := s.BulkIndex(ctx, inputs); err != nil {
			return fmt.Errorf("reindex: bulk index page %d: %w", page, err)
		}
		totalIndexed += len(inputs)

		s.logger.InfoContext(ctx, "reindex: page indexed",
			slog.Int("page", page),
			slog.Int("count", len(inputs)),
			slog.Int("total_indexed", totalIndexed),
		)

		if page >= result.TotalPages {
			break
		}
		page++
	}

	s.logger.InfoContext(ctx, "reindex completed", slog.Int("total_indexed", totalIndexed))
	return nil
}

// Suggester is an optional interface for engines that support autocomplete suggestions.
type Suggester interface {
	Suggest(ctx context.Context, prefix string, limit int) ([]string, error)
}

// Suggest returns autocomplete suggestions for a query prefix.
func (s *SearchService) Suggest(ctx context.Context, prefix string, limit int) ([]string, error) {
	if suggester, ok := s.engine.(Suggester); ok {
		return suggester.Suggest(ctx, prefix, limit)
	}
	// Fallback: no suggestions for engines that don't support it
	return []string{}, nil
}
