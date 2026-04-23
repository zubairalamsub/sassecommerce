package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/ecommerce/search-service/internal/models"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
)

// SearchRepository defines the interface for search data access
type SearchRepository interface {
	EnsureIndex(ctx context.Context) error
	IndexProduct(ctx context.Context, product *models.ProductDocument) error
	DeleteProduct(ctx context.Context, productID string) error
	Search(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error)
	Autocomplete(ctx context.Context, req *models.AutocompleteRequest) (*models.AutocompleteResponse, error)
}

type esSearchRepository struct {
	client    *elasticsearch.Client
	indexName string
	logger    *logrus.Logger
}

// NewSearchRepository creates a new Elasticsearch-backed search repository
func NewSearchRepository(client *elasticsearch.Client, indexName string, logger *logrus.Logger) SearchRepository {
	return &esSearchRepository{
		client:    client,
		indexName: indexName,
		logger:    logger,
	}
}

func (r *esSearchRepository) EnsureIndex(ctx context.Context) error {
	res, err := r.client.Indices.Exists([]string{r.indexName})
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		r.logger.Infof("Index '%s' already exists", r.indexName)
		return nil
	}

	mapping := models.IndexMapping()
	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal index mapping: %w", err)
	}

	res, err = r.client.Indices.Create(
		r.indexName,
		r.client.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to create index: %s", string(bodyBytes))
	}

	r.logger.Infof("Created index '%s'", r.indexName)
	return nil
}

func (r *esSearchRepository) IndexProduct(ctx context.Context, product *models.ProductDocument) error {
	body, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	res, err := r.client.Index(
		r.indexName,
		bytes.NewReader(body),
		r.client.Index.WithDocumentID(product.ID),
		r.client.Index.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to index product: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to index product: %s", string(bodyBytes))
	}

	return nil
}

func (r *esSearchRepository) DeleteProduct(ctx context.Context, productID string) error {
	res, err := r.client.Delete(
		r.indexName,
		productID,
		r.client.Delete.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to delete product: %s", string(bodyBytes))
	}

	return nil
}

func (r *esSearchRepository) Search(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	query := buildSearchQuery(req)

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search error: %s", string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return parseSearchResponse(result, req.Page, req.PageSize)
}

func (r *esSearchRepository) Autocomplete(ctx context.Context, req *models.AutocompleteRequest) (*models.AutocompleteResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	query := map[string]interface{}{
		"size": limit,
		"_source": []string{"id", "name", "brand", "category_name"},
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":  req.Query,
							"fields": []string{"name.autocomplete^3", "brand.autocomplete^2", "category_name"},
							"type":   "best_fields",
						},
					},
				},
				"filter": []interface{}{
					map[string]interface{}{
						"term": map[string]string{"tenant_id": req.TenantID},
					},
					map[string]interface{}{
						"term": map[string]string{"status": "active"},
					},
				},
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal autocomplete query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithIndex(r.indexName),
		r.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute autocomplete: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("autocomplete error: %s", string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode autocomplete response: %w", err)
	}

	return parseAutocompleteResponse(result)
}

// buildSearchQuery constructs an Elasticsearch query from a SearchRequest
func buildSearchQuery(req *models.SearchRequest) map[string]interface{} {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Build must clause
	var mustClauses []interface{}
	if req.Query != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     req.Query,
				"fields":    []string{"name^3", "description", "brand^2", "tags", "seo_keywords"},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		})
	}

	// Build filter clauses
	var filterClauses []interface{}

	// Always filter by tenant
	filterClauses = append(filterClauses, map[string]interface{}{
		"term": map[string]string{"tenant_id": req.TenantID},
	})

	// Only show active products
	filterClauses = append(filterClauses, map[string]interface{}{
		"term": map[string]string{"status": "active"},
	})

	if req.CategoryID != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]string{"category_id": req.CategoryID},
		})
	}

	if req.Brand != "" {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]string{"brand.keyword": req.Brand},
		})
	}

	if req.InStock != nil && *req.InStock {
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]bool{"in_stock": true},
		})
	}

	if len(req.Tags) > 0 {
		filterClauses = append(filterClauses, map[string]interface{}{
			"terms": map[string][]string{"tags": req.Tags},
		})
	}

	// Price range
	if req.MinPrice != nil || req.MaxPrice != nil {
		priceRange := map[string]interface{}{}
		if req.MinPrice != nil {
			priceRange["gte"] = *req.MinPrice
		}
		if req.MaxPrice != nil {
			priceRange["lte"] = *req.MaxPrice
		}
		filterClauses = append(filterClauses, map[string]interface{}{
			"range": map[string]interface{}{"price": priceRange},
		})
	}

	// Build bool query
	boolQuery := map[string]interface{}{
		"filter": filterClauses,
	}
	if len(mustClauses) > 0 {
		boolQuery["must"] = mustClauses
	}

	// Build sort
	var sort []interface{}
	if req.SortBy != "" {
		sortField := req.SortBy
		sortOrder := "asc"
		if req.SortOrder != "" {
			sortOrder = strings.ToLower(req.SortOrder)
		}
		// Map user-friendly sort fields
		switch sortField {
		case "name":
			sortField = "name.keyword"
		case "newest":
			sortField = "created_at"
			sortOrder = "desc"
		}
		sort = append(sort, map[string]interface{}{
			sortField: map[string]string{"order": sortOrder},
		})
	}
	if req.Query != "" {
		sort = append(sort, "_score")
	}

	query := map[string]interface{}{
		"from":  (page - 1) * pageSize,
		"size":  pageSize,
		"query": map[string]interface{}{"bool": boolQuery},
		"aggs": map[string]interface{}{
			"categories": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "category_id",
					"size":  20,
				},
			},
			"brands": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "brand.keyword",
					"size":  20,
				},
			},
			"tags": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "tags",
					"size":  30,
				},
			},
			"price_stats": map[string]interface{}{
				"stats": map[string]string{
					"field": "price",
				},
			},
		},
	}

	if len(sort) > 0 {
		query["sort"] = sort
	}

	return query
}

func parseSearchResponse(result map[string]interface{}, page, pageSize int) (*models.SearchResponse, error) {
	hits := result["hits"].(map[string]interface{})
	totalObj := hits["total"].(map[string]interface{})
	total := int64(totalObj["value"].(float64))

	hitsList := hits["hits"].([]interface{})
	products := make([]models.ProductHit, 0, len(hitsList))

	for _, h := range hitsList {
		hit := h.(map[string]interface{})
		source := hit["_source"].(map[string]interface{})
		score := 0.0
		if s, ok := hit["_score"]; ok && s != nil {
			score = s.(float64)
		}

		sourceBytes, err := json.Marshal(source)
		if err != nil {
			continue
		}

		var doc models.ProductDocument
		if err := json.Unmarshal(sourceBytes, &doc); err != nil {
			continue
		}

		products = append(products, models.ProductHit{
			ProductDocument: doc,
			Score:           score,
		})
	}

	// Parse facets
	var facets *models.SearchFacets
	if aggs, ok := result["aggregations"].(map[string]interface{}); ok {
		facets = &models.SearchFacets{}

		if cats, ok := aggs["categories"].(map[string]interface{}); ok {
			facets.Categories = parseBuckets(cats)
		}
		if brands, ok := aggs["brands"].(map[string]interface{}); ok {
			facets.Brands = parseBuckets(brands)
		}
		if tags, ok := aggs["tags"].(map[string]interface{}); ok {
			facets.Tags = parseBuckets(tags)
		}
		if priceStats, ok := aggs["price_stats"].(map[string]interface{}); ok {
			if min, ok := priceStats["min"].(float64); ok {
				facets.PriceRange = &models.PriceRange{
					Min: min,
					Max: priceStats["max"].(float64),
					Avg: priceStats["avg"].(float64),
				}
			}
		}
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &models.SearchResponse{
		Products:   products,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Facets:     facets,
	}, nil
}

func parseBuckets(agg map[string]interface{}) []models.FacetBucket {
	buckets, ok := agg["buckets"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.FacetBucket, 0, len(buckets))
	for _, b := range buckets {
		bucket := b.(map[string]interface{})
		key := fmt.Sprintf("%v", bucket["key"])
		count := int64(bucket["doc_count"].(float64))
		result = append(result, models.FacetBucket{
			Key:   key,
			Count: count,
		})
	}
	return result
}

func parseAutocompleteResponse(result map[string]interface{}) (*models.AutocompleteResponse, error) {
	hits := result["hits"].(map[string]interface{})
	hitsList := hits["hits"].([]interface{})

	seen := make(map[string]bool)
	suggestions := make([]models.Suggestion, 0)

	for _, h := range hitsList {
		hit := h.(map[string]interface{})
		source := hit["_source"].(map[string]interface{})
		score := 0.0
		if s, ok := hit["_score"]; ok && s != nil {
			score = s.(float64)
		}

		// Add product name suggestion
		if name, ok := source["name"].(string); ok && !seen[name] {
			seen[name] = true
			id := ""
			if v, ok := source["id"].(string); ok {
				id = v
			}
			suggestions = append(suggestions, models.Suggestion{
				Text:  name,
				Type:  "product",
				ID:    id,
				Score: score,
			})
		}

		// Add brand suggestion
		if brand, ok := source["brand"].(string); ok && brand != "" && !seen["brand:"+brand] {
			seen["brand:"+brand] = true
			suggestions = append(suggestions, models.Suggestion{
				Text:  brand,
				Type:  "brand",
				Score: score * 0.8,
			})
		}
	}

	return &models.AutocompleteResponse{Suggestions: suggestions}, nil
}
