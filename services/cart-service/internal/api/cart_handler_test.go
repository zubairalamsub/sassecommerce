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

	"github.com/ecommerce/cart-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCartService implements service.CartService for testing
type MockCartService struct {
	mock.Mock
}

func (m *MockCartService) AddItem(ctx context.Context, req *models.AddItemRequest) (*models.CartResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CartResponse), args.Error(1)
}

func (m *MockCartService) GetCart(ctx context.Context, tenantID, userID string) (*models.CartResponse, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CartResponse), args.Error(1)
}

func (m *MockCartService) UpdateItem(ctx context.Context, tenantID, userID, itemID string, req *models.UpdateItemRequest) (*models.CartResponse, error) {
	args := m.Called(ctx, tenantID, userID, itemID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CartResponse), args.Error(1)
}

func (m *MockCartService) RemoveItem(ctx context.Context, tenantID, userID, itemID string) (*models.CartResponse, error) {
	args := m.Called(ctx, tenantID, userID, itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CartResponse), args.Error(1)
}

func (m *MockCartService) ClearCart(ctx context.Context, tenantID, userID string) error {
	args := m.Called(ctx, tenantID, userID)
	return args.Error(0)
}

func (m *MockCartService) UpdateProductPrice(ctx context.Context, productID string, newPrice float64) error {
	args := m.Called(ctx, productID, newPrice)
	return args.Error(0)
}

func (m *MockCartService) RemoveProduct(ctx context.Context, productID string) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

func setupRouter(mockService *MockCartService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewCartHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

func createTestCartResponse() *models.CartResponse {
	return &models.CartResponse{
		TenantID: "tenant-1",
		UserID:   "user-1",
		Items: []models.CartItemResponse{
			{
				ID:        "item-1",
				ProductID: "product-1",
				Name:      "Widget A",
				Price:     29.99,
				Quantity:  2,
				Subtotal:  59.98,
				AddedAt:   time.Now().UTC().Format(time.RFC3339),
			},
		},
		TotalItems:  2,
		TotalAmount: 59.98,
		UpdatedAt:   time.Now().UTC(),
	}
}

// === AddItem Handler Tests ===

func TestHandler_AddItem_Success(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	resp := createTestCartResponse()
	mockService.On("AddItem", mock.Anything, mock.AnythingOfType("*models.AddItemRequest")).Return(resp, nil)

	body := `{
		"tenant_id": "tenant-1",
		"user_id": "user-1",
		"product_id": "product-1",
		"name": "Widget A",
		"price": 29.99,
		"quantity": 2
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/cart/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.CartResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 1, len(result.Items))
}

func TestHandler_AddItem_BadRequest(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	body := `{"tenant_id": "t1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/cart/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_AddItem_ServiceError(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	mockService.On("AddItem", mock.Anything, mock.AnythingOfType("*models.AddItemRequest")).
		Return(nil, errors.New("redis error"))

	body := `{
		"tenant_id": "tenant-1", "user_id": "user-1",
		"product_id": "product-1", "name": "Widget", "price": 10.0, "quantity": 1
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/cart/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === GetCart Handler Tests ===

func TestHandler_GetCart_Success(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	resp := createTestCartResponse()
	mockService.On("GetCart", mock.Anything, "tenant-1", "user-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/cart?tenant_id=tenant-1&user_id=user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.CartResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "tenant-1", result.TenantID)
	assert.Equal(t, 2, result.TotalItems)
}

func TestHandler_GetCart_MissingParams(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/cart", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCart_MissingUserID(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/cart?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCart_ServiceError(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	mockService.On("GetCart", mock.Anything, "tenant-1", "user-1").Return(nil, errors.New("redis error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/cart?tenant_id=tenant-1&user_id=user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === UpdateItem Handler Tests ===

func TestHandler_UpdateItem_Success(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	resp := createTestCartResponse()
	resp.Items[0].Quantity = 5
	mockService.On("UpdateItem", mock.Anything, "tenant-1", "user-1", "item-1", mock.AnythingOfType("*models.UpdateItemRequest")).Return(resp, nil)

	body := `{"quantity": 5}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/item-1?tenant_id=tenant-1&user_id=user-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateItem_MissingParams(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	body := `{"quantity": 5}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/item-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateItem_BadRequest(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	body := `{"quantity": 0}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/item-1?tenant_id=tenant-1&user_id=user-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateItem_NotFound(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	mockService.On("UpdateItem", mock.Anything, "tenant-1", "user-1", "bad", mock.AnythingOfType("*models.UpdateItemRequest")).
		Return(nil, errors.New("item not found in cart"))

	body := `{"quantity": 5}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/cart/items/bad?tenant_id=tenant-1&user_id=user-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === RemoveItem Handler Tests ===

func TestHandler_RemoveItem_Success(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	resp := createTestCartResponse()
	resp.Items = []models.CartItemResponse{}
	resp.TotalItems = 0
	resp.TotalAmount = 0
	mockService.On("RemoveItem", mock.Anything, "tenant-1", "user-1", "item-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/cart/items/item-1?tenant_id=tenant-1&user_id=user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RemoveItem_MissingParams(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/cart/items/item-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RemoveItem_NotFound(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	mockService.On("RemoveItem", mock.Anything, "tenant-1", "user-1", "bad").
		Return(nil, errors.New("item not found in cart"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/cart/items/bad?tenant_id=tenant-1&user_id=user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ClearCart Handler Tests ===

func TestHandler_ClearCart_Success(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	mockService.On("ClearCart", mock.Anything, "tenant-1", "user-1").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/cart?tenant_id=tenant-1&user_id=user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ClearCart_MissingParams(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/cart", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ClearCart_ServiceError(t *testing.T) {
	mockService := new(MockCartService)
	router := setupRouter(mockService)

	mockService.On("ClearCart", mock.Anything, "tenant-1", "user-1").Return(errors.New("redis error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/cart?tenant_id=tenant-1&user_id=user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
