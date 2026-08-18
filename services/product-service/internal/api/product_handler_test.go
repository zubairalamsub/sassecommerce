package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductService implements service.ProductService for testing.
type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.ProductResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductResponse), args.Error(1)
}

func (m *MockProductService) GetProductByID(ctx context.Context, tenantID, id string) (*models.ProductResponse, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductResponse), args.Error(1)
}

func (m *MockProductService) GetProductBySKU(ctx context.Context, tenantID, sku string) (*models.ProductResponse, error) {
	args := m.Called(ctx, tenantID, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductResponse), args.Error(1)
}

func (m *MockProductService) ListProducts(ctx context.Context, tenantID string, offset, limit int) ([]models.ProductResponse, int64, error) {
	args := m.Called(ctx, tenantID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.ProductResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductService) ListProductsByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.ProductResponse, int64, error) {
	args := m.Called(ctx, tenantID, categoryID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.ProductResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductService) SearchProducts(ctx context.Context, tenantID, query string, offset, limit int) ([]models.ProductResponse, int64, error) {
	args := m.Called(ctx, tenantID, query, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.ProductResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockProductService) UpdateProduct(ctx context.Context, tenantID, id string, req *models.UpdateProductRequest) (*models.ProductResponse, error) {
	args := m.Called(ctx, tenantID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductResponse), args.Error(1)
}

func (m *MockProductService) DeleteProduct(ctx context.Context, tenantID, id string) error {
	args := m.Called(ctx, tenantID, id)
	return args.Error(0)
}

func (m *MockProductService) UpdateProductStatus(ctx context.Context, tenantID, id string, status models.ProductStatus) error {
	args := m.Called(ctx, tenantID, id, status)
	return args.Error(0)
}

// setupProductRouter wires the handler behind a stub auth middleware that
// populates tenant_id in the gin context from a test header, mirroring what the
// real shared Auth middleware derives from the verified JWT. A request without
// the header simulates an unauthenticated caller (empty context).
//
// Routes are registered without auth middleware so the handler bodies are
// exercised directly; the role gate is covered by TestRegisterRoutes_AuthGate.
func setupProductRouter(mockService *MockProductService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewProductHandler(mockService, logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		c.Next()
	})
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

// doRequest issues a JSON request, setting the stub tenant header when tenant is
// non-empty. A nil body sends no payload.
func doRequest(router *gin.Engine, method, path, tenant string, body interface{}) *httptest.ResponseRecorder {
	var payload io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		payload = bytes.NewReader(b)
	}
	return doRawRequest(router, method, path, tenant, payload)
}

func doRawRequest(router *gin.Engine, method, path, tenant string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tenant != "" {
		req.Header.Set("X-Test-Tenant", tenant)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &out)
	assert.NoError(t, err, "response body should be valid JSON: %s", w.Body.String())
	return out
}

func testProductResponse() *models.ProductResponse {
	return &models.ProductResponse{
		ID:            "product-1",
		TenantID:      "tenant-1",
		SKU:           "SKU-001",
		Name:          "Test Product",
		Slug:          "test-product",
		CategoryID:    "category-1",
		Price:         99.99,
		Images:        []string{},
		Tags:          []string{},
		Status:        models.ProductStatusActive,
		InStock:       true,
		StockQuantity: 5,
		CreatedBy:     "user-1",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func validCreateBody() map[string]interface{} {
	return map[string]interface{}{
		"sku":         "SKU-001",
		"name":        "Test Product",
		"category_id": "category-1",
		"price":       99.99,
		"created_by":  "user-1",
	}
}

// ---------------------------------------------------------------- CreateProduct

func TestCreateProduct_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	expected := testProductResponse()
	mockService.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.CreateProductRequest")).Return(expected, nil)

	w := doRequest(router, http.MethodPost, "/api/v1/products", "tenant-1", validCreateBody())

	assert.Equal(t, http.StatusCreated, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "product-1", body["id"])
	assert.Equal(t, "SKU-001", body["sku"])
	mockService.AssertExpectations(t)
}

// The tenant must come from the authenticated context, never the request body —
// CreateProductRequest.TenantID is json:"-" precisely so a client cannot spoof it.
func TestCreateProduct_TenantComesFromContextNotBody(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	body := validCreateBody()
	body["tenant_id"] = "attacker-tenant"
	body["TenantID"] = "attacker-tenant"

	var captured *models.CreateProductRequest
	mockService.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.CreateProductRequest")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*models.CreateProductRequest)
		}).Return(testProductResponse(), nil)

	w := doRequest(router, http.MethodPost, "/api/v1/products", "tenant-1", body)

	assert.Equal(t, http.StatusCreated, w.Code)
	if assert.NotNil(t, captured) {
		assert.Equal(t, "tenant-1", captured.TenantID, "handler must override any body-supplied tenant with the JWT tenant")
	}
	mockService.AssertExpectations(t)
}

func TestCreateProduct_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodPost, "/api/v1/products", "", validCreateBody())

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
}

func TestCreateProduct_InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body interface{}
	}{
		{"malformed json", nil},
		{"missing sku", map[string]interface{}{"name": "n", "category_id": "c", "price": 1.0, "created_by": "u"}},
		{"missing name", map[string]interface{}{"sku": "s", "category_id": "c", "price": 1.0, "created_by": "u"}},
		{"missing category_id", map[string]interface{}{"sku": "s", "name": "n", "price": 1.0, "created_by": "u"}},
		{"missing created_by", map[string]interface{}{"sku": "s", "name": "n", "category_id": "c", "price": 1.0}},
		{"zero price", map[string]interface{}{"sku": "s", "name": "n", "category_id": "c", "price": 0, "created_by": "u"}},
		{"negative price", map[string]interface{}{"sku": "s", "name": "n", "category_id": "c", "price": -5, "created_by": "u"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockProductService)
			router := setupProductRouter(mockService)

			var w *httptest.ResponseRecorder
			if tc.body == nil {
				w = doRawRequest(router, http.MethodPost, "/api/v1/products", "tenant-1", bytes.NewReader([]byte("{not json")))
			} else {
				w = doRequest(router, http.MethodPost, "/api/v1/products", "tenant-1", tc.body)
			}

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "Invalid request body", decodeBody(t, w)["error"])
			mockService.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateProduct_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("CreateProduct", mock.Anything, mock.Anything).Return(nil, errors.New("sku already exists"))

	w := doRequest(router, http.MethodPost, "/api/v1/products", "tenant-1", validCreateBody())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "sku already exists", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

// ------------------------------------------------------------------- GetProduct

func TestGetProduct_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("GetProductByID", mock.Anything, "tenant-1", "product-1").Return(testProductResponse(), nil)

	w := doRequest(router, http.MethodGet, "/api/v1/products/product-1", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "product-1", decodeBody(t, w)["id"])
	mockService.AssertExpectations(t)
}

// The tenant is scoped from the context, so one tenant cannot read another's
// product even with a valid product id.
func TestGetProduct_ScopedToContextTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("GetProductByID", mock.Anything, "tenant-2", "product-1").Return(nil, errors.New("product not found"))

	w := doRequest(router, http.MethodGet, "/api/v1/products/product-1", "tenant-2", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestGetProduct_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/products/product-1", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "GetProductByID", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetProduct_NotFound(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("GetProductByID", mock.Anything, "tenant-1", "missing").Return(nil, errors.New("product not found"))

	w := doRequest(router, http.MethodGet, "/api/v1/products/missing", "tenant-1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Product not found", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

// -------------------------------------------------------------- GetProductBySKU

func TestGetProductBySKU_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("GetProductBySKU", mock.Anything, "tenant-1", "SKU-001").Return(testProductResponse(), nil)

	w := doRequest(router, http.MethodGet, "/api/v1/products/sku/SKU-001", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "SKU-001", decodeBody(t, w)["sku"])
	mockService.AssertExpectations(t)
}

func TestGetProductBySKU_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/products/sku/SKU-001", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "GetProductBySKU", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetProductBySKU_NotFound(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("GetProductBySKU", mock.Anything, "tenant-1", "nope").Return(nil, errors.New("product not found"))

	w := doRequest(router, http.MethodGet, "/api/v1/products/sku/nope", "tenant-1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Product not found", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

// ----------------------------------------------------------------- ListProducts

func TestListProducts_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	products := []models.ProductResponse{*testProductResponse()}
	mockService.On("ListProducts", mock.Anything, "tenant-1", 0, 20).Return(products, int64(1), nil)

	w := doRequest(router, http.MethodGet, "/api/v1/products", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, float64(1), body["total"])
	assert.Equal(t, float64(0), body["offset"])
	assert.Equal(t, float64(20), body["limit"])
	assert.Len(t, body["data"], 1)
	mockService.AssertExpectations(t)
}

// The handler clamps limit into (0, 100] and falls back to 20 for non-positive
// or unparseable values.
func TestListProducts_PaginationClamping(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedOffset int
		expectedLimit  int
	}{
		{"defaults", "", 0, 20},
		{"explicit values", "?offset=40&limit=10", 40, 10},
		{"limit above max is clamped", "?limit=500", 0, 100},
		{"limit at max is kept", "?limit=100", 0, 100},
		{"zero limit falls back", "?limit=0", 0, 20},
		{"negative limit falls back", "?limit=-5", 0, 20},
		{"unparseable limit falls back", "?limit=abc", 0, 20},
		{"unparseable offset becomes zero", "?offset=abc", 0, 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockProductService)
			router := setupProductRouter(mockService)

			mockService.On("ListProducts", mock.Anything, "tenant-1", tc.expectedOffset, tc.expectedLimit).
				Return([]models.ProductResponse{}, int64(0), nil)

			w := doRequest(router, http.MethodGet, "/api/v1/products"+tc.query, "tenant-1", nil)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestListProducts_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/products", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "ListProducts", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestListProducts_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("ListProducts", mock.Anything, "tenant-1", 0, 20).Return(nil, int64(0), errors.New("mongo down"))

	w := doRequest(router, http.MethodGet, "/api/v1/products", "tenant-1", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// The raw driver error must not leak to the client.
	assert.Equal(t, "Failed to retrieve products", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

// ------------------------------------------------------- ListProductsByCategory

func TestListProductsByCategory_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	products := []models.ProductResponse{*testProductResponse()}
	mockService.On("ListProductsByCategory", mock.Anything, "tenant-1", "category-1", 0, 20).Return(products, int64(1), nil)

	w := doRequest(router, http.MethodGet, "/api/v1/products/category/category-1", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, float64(1), body["total"])
	assert.Len(t, body["data"], 1)
	mockService.AssertExpectations(t)
}

func TestListProductsByCategory_PaginationClamping(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedOffset int
		expectedLimit  int
	}{
		{"defaults", "", 0, 20},
		{"explicit values", "?offset=10&limit=5", 10, 5},
		{"limit above max is clamped", "?limit=9999", 0, 100},
		{"zero limit falls back", "?limit=0", 0, 20},
		{"negative limit falls back", "?limit=-1", 0, 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockProductService)
			router := setupProductRouter(mockService)

			mockService.On("ListProductsByCategory", mock.Anything, "tenant-1", "category-1", tc.expectedOffset, tc.expectedLimit).
				Return([]models.ProductResponse{}, int64(0), nil)

			w := doRequest(router, http.MethodGet, "/api/v1/products/category/category-1"+tc.query, "tenant-1", nil)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestListProductsByCategory_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/products/category/category-1", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "ListProductsByCategory", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestListProductsByCategory_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("ListProductsByCategory", mock.Anything, "tenant-1", "category-1", 0, 20).
		Return(nil, int64(0), errors.New("boom"))

	w := doRequest(router, http.MethodGet, "/api/v1/products/category/category-1", "tenant-1", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to retrieve products", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

// --------------------------------------------------------------- SearchProducts

func TestSearchProducts_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	products := []models.ProductResponse{*testProductResponse()}
	mockService.On("SearchProducts", mock.Anything, "tenant-1", "shirt", 0, 20).Return(products, int64(1), nil)

	w := doRequest(router, http.MethodGet, "/api/v1/products/search?q=shirt", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "shirt", body["query"])
	assert.Equal(t, float64(1), body["total"])
	mockService.AssertExpectations(t)
}

func TestSearchProducts_PaginationClamping(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedOffset int
		expectedLimit  int
	}{
		{"defaults", "", 0, 20},
		{"explicit values", "&offset=5&limit=15", 5, 15},
		{"limit above max is clamped", "&limit=9999", 0, 100},
		{"zero limit falls back", "&limit=0", 0, 20},
		{"negative limit falls back", "&limit=-1", 0, 20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockProductService)
			router := setupProductRouter(mockService)

			mockService.On("SearchProducts", mock.Anything, "tenant-1", "shirt", tc.expectedOffset, tc.expectedLimit).
				Return([]models.ProductResponse{}, int64(0), nil)

			w := doRequest(router, http.MethodGet, "/api/v1/products/search?q=shirt"+tc.query, "tenant-1", nil)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSearchProducts_MissingQuery(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/products/search", "tenant-1", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Search query is required", decodeBody(t, w)["error"])
	mockService.AssertNotCalled(t, "SearchProducts", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSearchProducts_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/products/search?q=shirt", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "SearchProducts", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSearchProducts_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("SearchProducts", mock.Anything, "tenant-1", "shirt", 0, 20).Return(nil, int64(0), errors.New("index unavailable"))

	w := doRequest(router, http.MethodGet, "/api/v1/products/search?q=shirt", "tenant-1", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to search products", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

// ---------------------------------------------------------------- UpdateProduct

func validUpdateBody() map[string]interface{} {
	return map[string]interface{}{"name": "Renamed", "updated_by": "user-1"}
}

func TestUpdateProduct_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("UpdateProduct", mock.Anything, "tenant-1", "product-1", mock.AnythingOfType("*models.UpdateProductRequest")).
		Return(testProductResponse(), nil)

	w := doRequest(router, http.MethodPut, "/api/v1/products/product-1", "tenant-1", validUpdateBody())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "product-1", decodeBody(t, w)["id"])
	mockService.AssertExpectations(t)
}

func TestUpdateProduct_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodPut, "/api/v1/products/product-1", "", validUpdateBody())

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProduct_InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"malformed json", "{not json"},
		{"missing updated_by", "{\"name\":\"Renamed\"}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockProductService)
			router := setupProductRouter(mockService)

			w := doRawRequest(router, http.MethodPut, "/api/v1/products/product-1", "tenant-1", bytes.NewReader([]byte(tc.raw)))

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "Invalid request body", decodeBody(t, w)["error"])
			mockService.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateProduct_NotFound(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("UpdateProduct", mock.Anything, "tenant-1", "missing", mock.Anything).
		Return(nil, errors.New("product not found"))

	w := doRequest(router, http.MethodPut, "/api/v1/products/missing", "tenant-1", validUpdateBody())

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "product not found", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

func TestUpdateProduct_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("UpdateProduct", mock.Anything, "tenant-1", "product-1", mock.Anything).
		Return(nil, errors.New("write conflict"))

	w := doRequest(router, http.MethodPut, "/api/v1/products/product-1", "tenant-1", validUpdateBody())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ---------------------------------------------------------------- DeleteProduct

func TestDeleteProduct_Success(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("DeleteProduct", mock.Anything, "tenant-1", "product-1").Return(nil)

	w := doRequest(router, http.MethodDelete, "/api/v1/products/product-1", "tenant-1", nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestDeleteProduct_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/api/v1/products/product-1", "", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "DeleteProduct", mock.Anything, mock.Anything, mock.Anything)
}

func TestDeleteProduct_NotFound(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("DeleteProduct", mock.Anything, "tenant-1", "missing").Return(errors.New("failed to delete product"))

	w := doRequest(router, http.MethodDelete, "/api/v1/products/missing", "tenant-1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Product not found", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

func TestDeleteProduct_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("DeleteProduct", mock.Anything, "tenant-1", "product-1").Return(errors.New("mongo down"))

	w := doRequest(router, http.MethodDelete, "/api/v1/products/product-1", "tenant-1", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// ---------------------------------------------------------- UpdateProductStatus

func TestUpdateProductStatus_Success(t *testing.T) {
	for _, status := range []models.ProductStatus{
		models.ProductStatusDraft,
		models.ProductStatusActive,
		models.ProductStatusInactive,
		models.ProductStatusArchived,
	} {
		t.Run(string(status), func(t *testing.T) {
			mockService := new(MockProductService)
			router := setupProductRouter(mockService)

			mockService.On("UpdateProductStatus", mock.Anything, "tenant-1", "product-1", status).Return(nil)

			w := doRequest(router, http.MethodPatch, "/api/v1/products/product-1/status", "tenant-1",
				map[string]interface{}{"status": string(status)})

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "Product status updated successfully", decodeBody(t, w)["message"])
			mockService.AssertExpectations(t)
		})
	}
}

func TestUpdateProductStatus_InvalidStatus(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodPatch, "/api/v1/products/product-1/status", "tenant-1",
		map[string]interface{}{"status": "deleted"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid status value", decodeBody(t, w)["error"])
	mockService.AssertNotCalled(t, "UpdateProductStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProductStatus_MissingStatus(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodPatch, "/api/v1/products/product-1/status", "tenant-1", map[string]interface{}{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid request body", decodeBody(t, w)["error"])
	mockService.AssertNotCalled(t, "UpdateProductStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProductStatus_NoTenant(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	w := doRequest(router, http.MethodPatch, "/api/v1/products/product-1/status", "",
		map[string]interface{}{"status": "active"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "UpdateProductStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateProductStatus_NotFound(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("UpdateProductStatus", mock.Anything, "tenant-1", "missing", models.ProductStatusActive).
		Return(errors.New("failed to update product status"))

	w := doRequest(router, http.MethodPatch, "/api/v1/products/missing/status", "tenant-1",
		map[string]interface{}{"status": "active"})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Product not found", decodeBody(t, w)["error"])
	mockService.AssertExpectations(t)
}

func TestUpdateProductStatus_ServiceError(t *testing.T) {
	mockService := new(MockProductService)
	router := setupProductRouter(mockService)

	mockService.On("UpdateProductStatus", mock.Anything, "tenant-1", "product-1", models.ProductStatusActive).
		Return(errors.New("mongo down"))

	w := doRequest(router, http.MethodPatch, "/api/v1/products/product-1/status", "tenant-1",
		map[string]interface{}{"status": "active"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockService.AssertExpectations(t)
}

// --------------------------------------------------------------- RegisterRoutes

// Read routes stay public while every write route sits behind the supplied auth
// middleware, so an unauthenticated write is rejected before reaching the handler.
func TestRegisterRoutes_AuthGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	mockService := new(MockProductService)
	mockService.On("ListProducts", mock.Anything, "tenant-1", 0, 20).Return([]models.ProductResponse{}, int64(0), nil)

	handler := NewProductHandler(mockService, logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "tenant-1")
		c.Next()
	})
	// Deny-all stand-in for the real Auth middleware.
	denyAll := func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
	handler.RegisterRoutes(router.Group("/api/v1"), denyAll)

	writes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/products"},
		{http.MethodPut, "/api/v1/products/product-1"},
		{http.MethodDelete, "/api/v1/products/product-1"},
		{http.MethodPatch, "/api/v1/products/product-1/status"},
	}
	for _, wr := range writes {
		t.Run(wr.method+" is gated", func(t *testing.T) {
			w := doRequest(router, wr.method, wr.path, "", validCreateBody())
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}

	t.Run("GET stays public", func(t *testing.T) {
		w := doRequest(router, http.MethodGet, "/api/v1/products", "", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	mockService.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
	mockService.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockService.AssertNotCalled(t, "DeleteProduct", mock.Anything, mock.Anything, mock.Anything)
}
