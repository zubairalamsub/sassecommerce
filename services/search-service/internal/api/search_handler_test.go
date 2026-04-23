package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecommerce/search-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSearchService implements service.SearchService for testing
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) Search(ctx context.Context, req *models.SearchRequest) (*models.SearchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SearchResponse), args.Error(1)
}

func (m *MockSearchService) Autocomplete(ctx context.Context, req *models.AutocompleteRequest) (*models.AutocompleteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AutocompleteResponse), args.Error(1)
}

func (m *MockSearchService) IndexProduct(ctx context.Context, product *models.ProductDocument) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockSearchService) DeleteProduct(ctx context.Context, productID string) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

func (m *MockSearchService) UpdateProductStock(ctx context.Context, productID string, quantity int, inStock bool) error {
	args := m.Called(ctx, productID, quantity, inStock)
	return args.Error(0)
}

func setupRouter(mockService *MockSearchService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewSearchHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

func createTestSearchResponse() *models.SearchResponse {
	return &models.SearchResponse{
		Products: []models.ProductHit{
			{
				ProductDocument: models.ProductDocument{
					ID:       "product-1",
					TenantID: "tenant-1",
					Name:     "Premium Widget",
					Price:    49.99,
					Status:   "active",
					InStock:  true,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				},
				Score: 1.5,
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}
}

// === SearchProducts Handler Tests ===

func TestHandler_SearchProducts_Success(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	resp := createTestSearchResponse()
	mockService.On("Search", mock.Anything, mock.AnythingOfType("*models.SearchRequest")).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/products?tenant_id=tenant-1&q=widget", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.SearchResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, 1, len(result.Products))
}

func TestHandler_SearchProducts_MissingTenantID(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/products?q=widget", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SearchProducts_EmptyQuery(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	resp := createTestSearchResponse()
	mockService.On("Search", mock.Anything, mock.AnythingOfType("*models.SearchRequest")).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/products?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SearchProducts_WithFilters(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	resp := createTestSearchResponse()
	mockService.On("Search", mock.Anything, mock.AnythingOfType("*models.SearchRequest")).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/products?tenant_id=tenant-1&q=widget&category_id=cat-1&brand=WidgetCo&min_price=10&max_price=100&in_stock=true&sort_by=price&sort_order=asc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SearchProducts_WithTags(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	resp := createTestSearchResponse()
	mockService.On("Search", mock.Anything, mock.AnythingOfType("*models.SearchRequest")).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/products?tenant_id=tenant-1&tags=premium,new", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SearchProducts_ServiceError(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	mockService.On("Search", mock.Anything, mock.AnythingOfType("*models.SearchRequest")).
		Return(nil, errors.New("es error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/products?tenant_id=tenant-1&q=widget", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === Autocomplete Handler Tests ===

func TestHandler_Autocomplete_Success(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	resp := &models.AutocompleteResponse{
		Suggestions: []models.Suggestion{
			{Text: "Premium Widget", Type: "product", ID: "product-1"},
		},
	}
	mockService.On("Autocomplete", mock.Anything, mock.AnythingOfType("*models.AutocompleteRequest")).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/autocomplete?q=wid&tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.AutocompleteResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 1, len(result.Suggestions))
}

func TestHandler_Autocomplete_MissingQuery(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/autocomplete?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Autocomplete_MissingTenantID(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/autocomplete?q=wid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Autocomplete_ServiceError(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	mockService.On("Autocomplete", mock.Anything, mock.AnythingOfType("*models.AutocompleteRequest")).
		Return(nil, errors.New("es error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/autocomplete?q=wid&tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === ReindexProduct Handler Tests ===

func TestHandler_ReindexProduct_Success(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	mockService.On("IndexProduct", mock.Anything, mock.AnythingOfType("*models.ProductDocument")).Return(nil)

	body := `{
		"id": "product-1",
		"tenant_id": "tenant-1",
		"name": "Premium Widget",
		"price": 49.99,
		"status": "active"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search/reindex", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ReindexProduct_MissingID(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	body := `{"tenant_id": "tenant-1", "name": "Widget"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search/reindex", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReindexProduct_MissingTenantID(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	body := `{"id": "product-1", "name": "Widget"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search/reindex", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReindexProduct_InvalidJSON(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search/reindex", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReindexProduct_ServiceError(t *testing.T) {
	mockService := new(MockSearchService)
	router := setupRouter(mockService)

	mockService.On("IndexProduct", mock.Anything, mock.AnythingOfType("*models.ProductDocument")).
		Return(errors.New("es error"))

	body := `{"id": "product-1", "tenant_id": "tenant-1", "name": "Widget"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search/reindex", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === splitTags Tests ===

func TestSplitTags(t *testing.T) {
	tags := splitTags("premium,new,sale")
	assert.Equal(t, []string{"premium", "new", "sale"}, tags)
}

func TestSplitTags_Single(t *testing.T) {
	tags := splitTags("premium")
	assert.Equal(t, []string{"premium"}, tags)
}

func TestSplitTags_Empty(t *testing.T) {
	tags := splitTags("")
	assert.Equal(t, 0, len(tags))
}
