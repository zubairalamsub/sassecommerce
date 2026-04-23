package models

import "time"

// ProductDocument represents a product indexed in Elasticsearch
type ProductDocument struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	SKU            string            `json:"sku"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Brand          string            `json:"brand"`
	CategoryID     string            `json:"category_id"`
	CategoryName   string            `json:"category_name,omitempty"`
	Price          float64           `json:"price"`
	CompareAtPrice float64           `json:"compare_at_price,omitempty"`
	Images         []string          `json:"images,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Status         string            `json:"status"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	Variants       []VariantDocument `json:"variants,omitempty"`
	InStock        bool              `json:"in_stock"`
	StockQuantity  int               `json:"stock_quantity"`
	Weight         float64           `json:"weight,omitempty"`
	SEOKeywords    []string          `json:"seo_keywords,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// VariantDocument represents a product variant in the search index
type VariantDocument struct {
	ID      string            `json:"id"`
	SKU     string            `json:"sku"`
	Name    string            `json:"name"`
	Price   float64           `json:"price"`
	Options map[string]string `json:"options,omitempty"`
}

// --- Search Request/Response DTOs ---

type SearchRequest struct {
	Query      string            `form:"q"`
	TenantID   string            `form:"tenant_id" binding:"required"`
	CategoryID string            `form:"category_id"`
	Brand      string            `form:"brand"`
	MinPrice   *float64          `form:"min_price"`
	MaxPrice   *float64          `form:"max_price"`
	Tags       []string          `form:"tags"`
	InStock    *bool             `form:"in_stock"`
	Attributes map[string]string `form:"-"`
	SortBy     string            `form:"sort_by"`
	SortOrder  string            `form:"sort_order"`
	Page       int               `form:"page"`
	PageSize   int               `form:"page_size"`
}

type SearchResponse struct {
	Products   []ProductHit       `json:"products"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
	Facets     *SearchFacets      `json:"facets,omitempty"`
}

type ProductHit struct {
	ProductDocument
	Score float64 `json:"_score,omitempty"`
}

type SearchFacets struct {
	Categories []FacetBucket `json:"categories,omitempty"`
	Brands     []FacetBucket `json:"brands,omitempty"`
	Tags       []FacetBucket `json:"tags,omitempty"`
	PriceRange *PriceRange   `json:"price_range,omitempty"`
}

type FacetBucket struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// AutocompleteRequest for search suggestions
type AutocompleteRequest struct {
	Query    string `form:"q" binding:"required"`
	TenantID string `form:"tenant_id" binding:"required"`
	Limit    int    `form:"limit"`
}

type AutocompleteResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
}

type Suggestion struct {
	Text      string  `json:"text"`
	Type      string  `json:"type"` // product, brand, category
	ID        string  `json:"id,omitempty"`
	Score     float64 `json:"_score,omitempty"`
}

// --- Kafka Event Models ---

type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Data      map[string]interface{} `json:"data"`
}

func (e *EventEnvelope) GetPayload() map[string]interface{} {
	if e.Payload != nil {
		return e.Payload
	}
	return e.Data
}

// IndexMapping returns the Elasticsearch index mapping for products
func IndexMapping() map[string]interface{} {
	return map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"autocomplete_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "autocomplete_tokenizer",
						"filter":    []string{"lowercase"},
					},
					"search_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter":    []string{"lowercase"},
					},
				},
				"tokenizer": map[string]interface{}{
					"autocomplete_tokenizer": map[string]interface{}{
						"type":        "edge_ngram",
						"min_gram":    2,
						"max_gram":    20,
						"token_chars": []string{"letter", "digit"},
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id":               map[string]string{"type": "keyword"},
				"tenant_id":        map[string]string{"type": "keyword"},
				"sku":              map[string]string{"type": "keyword"},
				"name": map[string]interface{}{
					"type":            "text",
					"analyzer":        "standard",
					"fields": map[string]interface{}{
						"autocomplete": map[string]interface{}{
							"type":            "text",
							"analyzer":        "autocomplete_analyzer",
							"search_analyzer": "search_analyzer",
						},
						"keyword": map[string]string{
							"type": "keyword",
						},
					},
				},
				"description":      map[string]string{"type": "text"},
				"brand": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]string{"type": "keyword"},
						"autocomplete": map[string]interface{}{
							"type":            "text",
							"analyzer":        "autocomplete_analyzer",
							"search_analyzer": "search_analyzer",
						},
					},
				},
				"category_id":      map[string]string{"type": "keyword"},
				"category_name": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"keyword": map[string]string{"type": "keyword"},
					},
				},
				"price":            map[string]string{"type": "float"},
				"compare_at_price": map[string]string{"type": "float"},
				"images":           map[string]string{"type": "keyword"},
				"tags":             map[string]string{"type": "keyword"},
				"status":           map[string]string{"type": "keyword"},
				"attributes": map[string]interface{}{
					"type": "object",
					"dynamic": true,
				},
				"in_stock":         map[string]string{"type": "boolean"},
				"stock_quantity":   map[string]string{"type": "integer"},
				"weight":           map[string]string{"type": "float"},
				"seo_keywords":     map[string]string{"type": "keyword"},
				"created_at":       map[string]string{"type": "date"},
				"updated_at":       map[string]string{"type": "date"},
			},
		},
	}
}
