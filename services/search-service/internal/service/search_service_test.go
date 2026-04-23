package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/search-service/internal/models"
	repoMocks "github.com/ecommerce/search-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*searchService, *repoMocks.MockSearchRepository) {
	mockRepo := new(repoMocks.MockSearchRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &searchService{
		repo:   mockRepo,
		logger: logger,
	}

	return svc, mockRepo
}

func createTestProduct() *models.ProductDocument {
	return &models.ProductDocument{
		ID:          "product-1",
		TenantID:    "tenant-1",
		SKU:         "SKU-001",
		Name:        "Premium Widget",
		Description: "A high-quality widget for all your needs",
		Brand:       "WidgetCo",
		CategoryID:  "cat-1",
		Price:       49.99,
		Tags:        []string{"premium", "widget", "new"},
		Status:      "active",
		InStock:     true,
		StockQuantity: 100,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func createTestSearchResponse() *models.SearchResponse {
	return &models.SearchResponse{
		Products: []models.ProductHit{
			{
				ProductDocument: *createTestProduct(),
				Score:           1.5,
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
		Facets: &models.SearchFacets{
			Categories: []models.FacetBucket{{Key: "cat-1", Count: 1}},
			Brands:     []models.FacetBucket{{Key: "WidgetCo", Count: 1}},
		},
	}
}

// === Search Tests ===

func TestSearch_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := createTestSearchResponse()
	mockRepo.On("Search", ctx, mock.AnythingOfType("*models.SearchRequest")).Return(expected, nil)

	req := &models.SearchRequest{
		Query:    "widget",
		TenantID: "tenant-1",
		Page:     1,
		PageSize: 20,
	}

	result, err := svc.Search(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 1, len(result.Products))
	assert.Equal(t, "Premium Widget", result.Products[0].Name)
}

func TestSearch_DefaultPagination(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := createTestSearchResponse()
	mockRepo.On("Search", ctx, mock.AnythingOfType("*models.SearchRequest")).Return(expected, nil)

	req := &models.SearchRequest{
		Query:    "widget",
		TenantID: "tenant-1",
		Page:     0,  // should default to 1
		PageSize: 0,  // should default to 20
	}

	result, err := svc.Search(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, req.Page)
	assert.Equal(t, 20, req.PageSize)
}

func TestSearch_MaxPageSize(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := createTestSearchResponse()
	mockRepo.On("Search", ctx, mock.AnythingOfType("*models.SearchRequest")).Return(expected, nil)

	req := &models.SearchRequest{
		Query:    "widget",
		TenantID: "tenant-1",
		PageSize: 500, // should be capped to 100
	}

	_, err := svc.Search(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 100, req.PageSize)
}

func TestSearch_EmptyQuery(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := createTestSearchResponse()
	mockRepo.On("Search", ctx, mock.AnythingOfType("*models.SearchRequest")).Return(expected, nil)

	req := &models.SearchRequest{
		TenantID: "tenant-1",
	}

	result, err := svc.Search(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSearch_WithFilters(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := createTestSearchResponse()
	mockRepo.On("Search", ctx, mock.AnythingOfType("*models.SearchRequest")).Return(expected, nil)

	minPrice := 10.0
	maxPrice := 100.0
	inStock := true

	req := &models.SearchRequest{
		Query:      "widget",
		TenantID:   "tenant-1",
		CategoryID: "cat-1",
		Brand:      "WidgetCo",
		MinPrice:   &minPrice,
		MaxPrice:   &maxPrice,
		InStock:    &inStock,
		Tags:       []string{"premium"},
	}

	result, err := svc.Search(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSearch_Failure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Search", ctx, mock.AnythingOfType("*models.SearchRequest")).
		Return(nil, errors.New("elasticsearch error"))

	req := &models.SearchRequest{
		Query:    "widget",
		TenantID: "tenant-1",
	}

	result, err := svc.Search(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "search failed")
}

// === Autocomplete Tests ===

func TestAutocomplete_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := &models.AutocompleteResponse{
		Suggestions: []models.Suggestion{
			{Text: "Premium Widget", Type: "product", ID: "product-1"},
			{Text: "WidgetCo", Type: "brand"},
		},
	}
	mockRepo.On("Autocomplete", ctx, mock.AnythingOfType("*models.AutocompleteRequest")).Return(expected, nil)

	req := &models.AutocompleteRequest{
		Query:    "wid",
		TenantID: "tenant-1",
		Limit:    10,
	}

	result, err := svc.Autocomplete(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result.Suggestions))
	assert.Equal(t, "product", result.Suggestions[0].Type)
	assert.Equal(t, "brand", result.Suggestions[1].Type)
}

func TestAutocomplete_DefaultLimit(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := &models.AutocompleteResponse{Suggestions: []models.Suggestion{}}
	mockRepo.On("Autocomplete", ctx, mock.AnythingOfType("*models.AutocompleteRequest")).Return(expected, nil)

	req := &models.AutocompleteRequest{
		Query:    "wid",
		TenantID: "tenant-1",
		Limit:    0,
	}

	_, err := svc.Autocomplete(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 10, req.Limit)
}

func TestAutocomplete_MaxLimit(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	expected := &models.AutocompleteResponse{Suggestions: []models.Suggestion{}}
	mockRepo.On("Autocomplete", ctx, mock.AnythingOfType("*models.AutocompleteRequest")).Return(expected, nil)

	req := &models.AutocompleteRequest{
		Query:    "wid",
		TenantID: "tenant-1",
		Limit:    50,
	}

	_, err := svc.Autocomplete(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 20, req.Limit)
}

func TestAutocomplete_Failure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("Autocomplete", ctx, mock.AnythingOfType("*models.AutocompleteRequest")).
		Return(nil, errors.New("elasticsearch error"))

	req := &models.AutocompleteRequest{
		Query:    "wid",
		TenantID: "tenant-1",
	}

	result, err := svc.Autocomplete(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "autocomplete failed")
}

// === IndexProduct Tests ===

func TestIndexProduct_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	product := createTestProduct()
	mockRepo.On("IndexProduct", ctx, mock.AnythingOfType("*models.ProductDocument")).Return(nil)

	err := svc.IndexProduct(ctx, product)

	assert.NoError(t, err)
	mockRepo.AssertCalled(t, "IndexProduct", ctx, product)
}

func TestIndexProduct_MissingID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	product := createTestProduct()
	product.ID = ""

	err := svc.IndexProduct(ctx, product)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID is required")
}

func TestIndexProduct_MissingTenantID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	product := createTestProduct()
	product.TenantID = ""

	err := svc.IndexProduct(ctx, product)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tenant ID is required")
}

func TestIndexProduct_DefaultStatus(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	product := createTestProduct()
	product.Status = ""
	mockRepo.On("IndexProduct", ctx, mock.AnythingOfType("*models.ProductDocument")).Return(nil)

	err := svc.IndexProduct(ctx, product)

	assert.NoError(t, err)
	assert.Equal(t, "active", product.Status)
}

func TestIndexProduct_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	product := createTestProduct()
	mockRepo.On("IndexProduct", ctx, mock.AnythingOfType("*models.ProductDocument")).
		Return(errors.New("es error"))

	err := svc.IndexProduct(ctx, product)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to index product")
}

// === DeleteProduct Tests ===

func TestDeleteProduct_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("DeleteProduct", ctx, "product-1").Return(nil)

	err := svc.DeleteProduct(ctx, "product-1")

	assert.NoError(t, err)
}

func TestDeleteProduct_EmptyID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.DeleteProduct(ctx, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID is required")
}

func TestDeleteProduct_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("DeleteProduct", ctx, "product-1").Return(errors.New("es error"))

	err := svc.DeleteProduct(ctx, "product-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete product")
}

// === UpdateProductStock Tests ===

func TestUpdateProductStock_Success(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.UpdateProductStock(ctx, "product-1", 50, true)

	assert.NoError(t, err)
}

func TestUpdateProductStock_EmptyID(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	err := svc.UpdateProductStock(ctx, "", 50, true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID is required")
}
