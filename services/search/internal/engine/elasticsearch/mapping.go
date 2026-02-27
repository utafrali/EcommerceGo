package elasticsearch

// IndexName is the Elasticsearch index used for product documents.
const IndexName = "products"

// IndexMapping is the full JSON mapping for the products index,
// including custom analyzers for autocomplete support.
const IndexMapping = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "analyzer": {
        "autocomplete_analyzer": {
          "type": "custom",
          "tokenizer": "autocomplete_tokenizer",
          "filter": ["lowercase"]
        },
        "autocomplete_search": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase"]
        }
      },
      "tokenizer": {
        "autocomplete_tokenizer": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20,
          "token_chars": ["letter", "digit"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "name":          { "type": "text", "analyzer": "standard", "fields": { "autocomplete": { "type": "text", "analyzer": "autocomplete_analyzer", "search_analyzer": "autocomplete_search" } } },
      "slug":          { "type": "keyword" },
      "description":   { "type": "text", "analyzer": "standard" },
      "category_id":   { "type": "keyword" },
      "category_name": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "brand_id":      { "type": "keyword" },
      "brand_name":    { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
      "base_price":    { "type": "long" },
      "currency":      { "type": "keyword" },
      "status":        { "type": "keyword" },
      "image_url":     { "type": "keyword", "index": false },
      "tags":          { "type": "keyword" },
      "attributes":    { "type": "object", "enabled": false },
      "created_at":    { "type": "date" },
      "updated_at":    { "type": "date" }
    }
  }
}`
