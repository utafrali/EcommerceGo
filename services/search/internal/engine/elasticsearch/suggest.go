package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/utafrali/EcommerceGo/services/search/internal/domain"
)

// esSuggestResponse is the structure used to decode Elasticsearch suggest responses.
type esSuggestResponse struct {
	Hits struct {
		Hits []struct {
			Source domain.SearchableProduct `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// Suggest returns autocomplete suggestions for the given prefix.
// It queries the name.autocomplete field and returns unique product names
// that match the prefix, limited to published products.
func (e *Engine) Suggest(ctx context.Context, prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"match": map[string]interface{}{
							"name.autocomplete": prefix,
						},
					},
				},
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"status": "published",
						},
					},
				},
			},
		},
		"size":    limit,
		"_source": []string{"name"},
		"sort": []interface{}{
			map[string]interface{}{"_score": "desc"},
		},
	}

	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch suggest: marshal query: %w", err)
	}

	res, err := e.client.Search(
		e.client.Search.WithIndex(IndexName),
		e.client.Search.WithBody(bytes.NewReader(data)),
		e.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch suggest: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var errResp esErrorResponse
		if decErr := json.NewDecoder(res.Body).Decode(&errResp); decErr == nil {
			return nil, fmt.Errorf("elasticsearch suggest: %s — %s", errResp.Error.Type, errResp.Error.Reason)
		}
		return nil, fmt.Errorf("elasticsearch suggest: unexpected status %s", res.Status())
	}

	var esResp esSuggestResponse
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("elasticsearch suggest: decode response: %w", err)
	}

	// Deduplicate names while preserving order.
	seen := make(map[string]struct{})
	var names []string
	for _, hit := range esResp.Hits.Hits {
		name := hit.Source.Name
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	return names, nil
}
