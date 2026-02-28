package elasticsearch

// DefaultIndexName is the default Elasticsearch index used for product documents.
const DefaultIndexName = "ecommerce_products"

// buildIndexMapping returns the full JSON mapping for the products index,
// including custom analyzers for autocomplete and Turkish language support.
func buildIndexMapping() string {
	return `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "analyzer": {
        "turkish_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "turkish_stop", "turkish_stemmer"]
        },
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
      },
      "filter": {
        "turkish_stop": {
          "type": "stop",
          "stopwords": "_turkish_"
        },
        "turkish_stemmer": {
          "type": "stemmer",
          "language": "turkish"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id":           { "type": "keyword" },
      "name":         { "type": "text", "analyzer": "turkish_analyzer", "fields": { "keyword": { "type": "keyword", "ignore_above": 256 }, "autocomplete": { "type": "text", "analyzer": "autocomplete_analyzer", "search_analyzer": "autocomplete_search" } } },
      "slug":         { "type": "keyword" },
      "description":  { "type": "text", "analyzer": "turkish_analyzer" },
      "category_id":  { "type": "keyword" },
      "category_name":{ "type": "text", "analyzer": "turkish_analyzer", "fields": { "keyword": { "type": "keyword" } } },
      "brand_id":     { "type": "keyword" },
      "brand_name":   { "type": "text", "analyzer": "turkish_analyzer", "fields": { "keyword": { "type": "keyword" } } },
      "base_price":   { "type": "long" },
      "currency":     { "type": "keyword" },
      "status":       { "type": "keyword" },
      "image_url":    { "type": "keyword", "index": false },
      "tags":         { "type": "keyword" },
      "attributes":   { "type": "object", "enabled": false },
      "created_at":   { "type": "date" },
      "updated_at":   { "type": "date" }
    }
  }
}`
}
