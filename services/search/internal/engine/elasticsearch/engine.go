package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
)

// Engine is an Elasticsearch-backed implementation of the SearchEngine interface.
type Engine struct {
	client    *elasticsearch.Client
	indexName string
	logger    *slog.Logger
}

// esSearchResponse is the structure used to decode Elasticsearch search responses.
type esSearchResponse struct {
	Took int `json:"took"`
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source domain.SearchableProduct `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// esBulkResponse is the structure used to decode Elasticsearch bulk responses.
type esBulkResponse struct {
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"index"`
	} `json:"items"`
}

// esErrorResponse is used to decode Elasticsearch error responses.
type esErrorResponse struct {
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
	Status int `json:"status"`
}

// New creates a new Elasticsearch engine connected to the given URL.
// It ensures the products index exists, creating it if necessary.
// If indexName is empty, DefaultIndexName ("ecommerce_products") is used.
func New(esURL string, indexName string, logger *slog.Logger) (*Engine, error) {
	if indexName == "" {
		indexName = DefaultIndexName
	}

	cfg := elasticsearch.Config{
		Addresses: []string{esURL},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: failed to create client: %w", err)
	}

	e := &Engine{
		client:    client,
		indexName: indexName,
		logger:    logger,
	}

	if err := e.ensureIndex(); err != nil {
		return nil, fmt.Errorf("elasticsearch: failed to ensure index: %w", err)
	}

	return e, nil
}

// Ping checks whether the Elasticsearch cluster is reachable.
func (e *Engine) Ping(ctx context.Context) error {
	res, err := e.client.Ping(e.client.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elasticsearch ping: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping: unexpected status %s", res.Status())
	}
	return nil
}

// ensureIndex checks whether the products index exists and creates it if not.
func (e *Engine) ensureIndex() error {
	res, err := e.client.Indices.Exists([]string{e.indexName})
	if err != nil {
		return fmt.Errorf("check index exists: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	// Status 200 means the index exists.
	if res.StatusCode == 200 {
		e.logger.Info("elasticsearch index already exists", "index", e.indexName)
		return nil
	}

	// Create the index with the mapping.
	mapping := buildIndexMapping()
	res, err = e.client.Indices.Create(
		e.indexName,
		e.client.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return fmt.Errorf("create index: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return fmt.Errorf("create index: unexpected status %s", res.Status())
	}

	e.logger.Info("elasticsearch index created", "index", e.indexName)
	return nil
}

// Index adds or updates a single product in the Elasticsearch index.
func (e *Engine) Index(ctx context.Context, product *domain.SearchableProduct) error {
	data, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("elasticsearch index: marshal product: %w", err)
	}

	res, err := e.client.Index(
		e.indexName,
		bytes.NewReader(data),
		e.client.Index.WithDocumentID(product.ID),
		e.client.Index.WithRefresh("true"),
		e.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch index: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return fmt.Errorf("elasticsearch index: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return fmt.Errorf("elasticsearch index: unexpected status %s", res.Status())
	}

	e.logger.Debug("indexed product", "id", product.ID, "name", product.Name)
	return nil
}

// Delete removes a product from the Elasticsearch index by its ID.
// It does not return an error if the document does not exist (404 is ignored).
func (e *Engine) Delete(ctx context.Context, id string) error {
	res, err := e.client.Delete(
		e.indexName,
		id,
		e.client.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch delete: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	// Ignore 404 — the document might not exist.
	if res.IsError() && res.StatusCode != 404 {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return fmt.Errorf("elasticsearch delete: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return fmt.Errorf("elasticsearch delete: unexpected status %s", res.Status())
	}

	e.logger.Debug("deleted product", "id", id)
	return nil
}

// Search executes a search query against Elasticsearch and returns matching products.
func (e *Engine) Search(ctx context.Context, query *domain.SearchQuery) (*domain.SearchResult, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	perPage := query.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	esQuery := e.buildSearchQuery(query, page, perPage)

	data, err := json.Marshal(esQuery)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search: marshal query: %w", err)
	}

	res, err := e.client.Search(
		e.client.Search.WithIndex(e.indexName),
		e.client.Search.WithBody(bytes.NewReader(data)),
		e.client.Search.WithContext(ctx),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return nil, fmt.Errorf("elasticsearch search: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return nil, fmt.Errorf("elasticsearch search: unexpected status %s", res.Status())
	}

	var esResp esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("elasticsearch search: decode response: %w", err)
	}

	products := make([]domain.SearchableProduct, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		products = append(products, hit.Source)
	}

	return &domain.SearchResult{
		Products: products,
		Total:    esResp.Hits.Total.Value,
		Page:     page,
		PerPage:  perPage,
		TookMs:   int64(esResp.Took),
	}, nil
}

// buildSearchQuery constructs the Elasticsearch query DSL as a map.
func (e *Engine) buildSearchQuery(query *domain.SearchQuery, page, perPage int) map[string]interface{} {
	// Build the must clause.
	var mustClause interface{}
	if query.Query != "" {
		mustClause = map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":         query.Query,
				"fields":        []string{"name^3", "name.autocomplete^2", "description", "category_name", "brand_name"},
				"type":          "best_fields",
				"fuzziness":     "AUTO",
				"prefix_length": 1,
			},
		}
	} else {
		mustClause = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// Build the filter clauses.
	filters := e.buildFilters(query)

	// Build the bool query.
	boolQuery := map[string]interface{}{
		"must": []interface{}{mustClause},
	}
	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	// Build the sort clause.
	sortClause := e.buildSort(query.SortBy)

	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": boolQuery,
		},
		"from":             (page - 1) * perPage,
		"size":             perPage,
		"track_total_hits": true,
	}

	if sortClause != nil {
		esQuery["sort"] = sortClause
	}

	return esQuery
}

// buildFilters constructs the filter clauses based on the search query.
func (e *Engine) buildFilters(query *domain.SearchQuery) []interface{} {
	var filters []interface{}

	if query.CategoryID != nil {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"category_id": *query.CategoryID,
			},
		})
	}

	if query.BrandID != nil {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"brand_id": *query.BrandID,
			},
		})
	}

	if query.Status != nil {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"status": *query.Status,
			},
		})
	}

	if query.MinPrice != nil || query.MaxPrice != nil {
		rangeFilter := map[string]interface{}{}
		if query.MinPrice != nil {
			rangeFilter["gte"] = *query.MinPrice
		}
		if query.MaxPrice != nil {
			rangeFilter["lte"] = *query.MaxPrice
		}
		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"base_price": rangeFilter,
			},
		})
	}

	return filters
}

// buildSort constructs the sort clause based on the sort option.
func (e *Engine) buildSort(sortBy string) []interface{} {
	switch sortBy {
	case domain.SortPriceAsc:
		return []interface{}{
			map[string]interface{}{"base_price": "asc"},
		}
	case domain.SortPriceDesc:
		return []interface{}{
			map[string]interface{}{"base_price": "desc"},
		}
	case domain.SortNewest:
		return []interface{}{
			map[string]interface{}{"created_at": "desc"},
		}
	default:
		// SortRelevance: use default ES scoring (no explicit sort).
		return []interface{}{
			map[string]interface{}{"_score": "desc"},
		}
	}
}

// DeleteIndex removes the entire Elasticsearch index.
// It is intended for testing and administrative operations only.
// A 404 response is treated as success (index already absent).
func (e *Engine) DeleteIndex(ctx context.Context) error {
	res, err := e.client.Indices.Delete(
		[]string{e.indexName},
		e.client.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch delete index: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() && res.StatusCode != 404 {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return fmt.Errorf("elasticsearch delete index: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return fmt.Errorf("elasticsearch delete index: unexpected status %s", res.Status())
	}

	e.logger.Info("elasticsearch index deleted", "index", e.indexName)
	return nil
}

// BulkIndex adds or updates multiple products in the Elasticsearch index
// using the bulk NDJSON API.
func (e *Engine) BulkIndex(ctx context.Context, products []domain.SearchableProduct) error {
	if len(products) == 0 {
		return nil
	}

	var buf bytes.Buffer

	for i := range products {
		// Action line.
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": e.indexName,
				"_id":    products[i].ID,
			},
		}
		if err := json.NewEncoder(&buf).Encode(action); err != nil {
			return fmt.Errorf("elasticsearch bulk index: encode action: %w", err)
		}

		// Document line.
		if err := json.NewEncoder(&buf).Encode(products[i]); err != nil {
			return fmt.Errorf("elasticsearch bulk index: encode document: %w", err)
		}
	}

	res, err := e.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		e.client.Bulk.WithIndex(e.indexName),
		e.client.Bulk.WithRefresh("true"),
		e.client.Bulk.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("elasticsearch bulk index: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.IsError() {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return fmt.Errorf("elasticsearch bulk index: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return fmt.Errorf("elasticsearch bulk index: unexpected status %s", res.Status())
	}

	// Parse the bulk response to check for per-item errors.
	var bulkResp esBulkResponse
	if err := json.NewDecoder(res.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("elasticsearch bulk index: decode response: %w", err)
	}

	if bulkResp.Errors {
		var errMsgs []string
		for _, item := range bulkResp.Items {
			if item.Index.Error.Type != "" {
				errMsgs = append(errMsgs, fmt.Sprintf("id=%s: %s — %s", item.Index.ID, item.Index.Error.Type, item.Index.Error.Reason))
			}
		}
		return fmt.Errorf("elasticsearch bulk index: partial errors: %s", strings.Join(errMsgs, "; "))
	}

	e.logger.Info("bulk indexed products", "count", len(products))
	return nil
}
