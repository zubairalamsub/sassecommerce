package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/vendor-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockVendorService struct {
	mock.Mock
}

func (m *MockVendorService) RegisterVendor(ctx context.Context, req *models.RegisterVendorRequest) (*models.VendorResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VendorResponse), args.Error(1)
}

func (m *MockVendorService) GetVendor(ctx context.Context, id string) (*models.VendorResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VendorResponse), args.Error(1)
}

func (m *MockVendorService) ListVendors(ctx context.Context, tenantID, status string, page, pageSize int) ([]models.VendorResponse, int64, error) {
	args := m.Called(ctx, tenantID, status, page, pageSize)
	return args.Get(0).([]models.VendorResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockVendorService) UpdateVendor(ctx context.Context, id string, req *models.UpdateVendorRequest) (*models.VendorResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VendorResponse), args.Error(1)
}

func (m *MockVendorService) UpdateVendorStatus(ctx context.Context, id string, req *models.UpdateVendorStatusRequest) (*models.VendorResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VendorResponse), args.Error(1)
}

func (m *MockVendorService) GetVendorOrders(ctx context.Context, vendorID string, page, pageSize int) ([]models.VendorOrderResponse, int64, error) {
	args := m.Called(ctx, vendorID, page, pageSize)
	return args.Get(0).([]models.VendorOrderResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockVendorService) GetVendorAnalytics(ctx context.Context, vendorID string) (*models.VendorAnalyticsResponse, error) {
	args := m.Called(ctx, vendorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VendorAnalyticsResponse), args.Error(1)
}

func (m *MockVendorService) RecordOrder(ctx context.Context, vendorID, tenantID, orderID string, amount float64) error {
	args := m.Called(ctx, vendorID, tenantID, orderID, amount)
	return args.Error(0)
}

func setupRouter(mockService *MockVendorService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewVendorHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

// === RegisterVendor Handler Tests ===

func TestHandler_RegisterVendor_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	resp := &models.VendorResponse{ID: "vendor-1", Name: "Acme", Status: models.StatusPending}
	mockService.On("RegisterVendor", mock.Anything, mock.AnythingOfType("*models.RegisterVendorRequest")).Return(resp, nil)

	body := `{"tenant_id": "tenant-1", "name": "Acme", "email": "vendor@acme.com"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vendors/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_RegisterVendor_BadRequest(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	body := `{"name": "Acme"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vendors/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RegisterVendor_Conflict(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	mockService.On("RegisterVendor", mock.Anything, mock.AnythingOfType("*models.RegisterVendorRequest")).
		Return(nil, errors.New("vendor with this email already exists"))

	body := `{"tenant_id": "t1", "name": "Acme", "email": "vendor@acme.com"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/vendors/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// === GetVendor Handler Tests ===

func TestHandler_GetVendor_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	resp := &models.VendorResponse{ID: "vendor-1", Name: "Acme"}
	mockService.On("GetVendor", mock.Anything, "vendor-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/vendor-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetVendor_NotFound(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	mockService.On("GetVendor", mock.Anything, "bad").Return(nil, errors.New("vendor not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ListVendors Handler Tests ===

func TestHandler_ListVendors_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	resp := []models.VendorResponse{{ID: "vendor-1", Name: "Acme"}}
	mockService.On("ListVendors", mock.Anything, "tenant-1", "", 1, 20).Return(resp, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, float64(1), result["total"])
}

func TestHandler_ListVendors_MissingTenantID(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === UpdateVendor Handler Tests ===

func TestHandler_UpdateVendor_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	resp := &models.VendorResponse{ID: "vendor-1", Name: "Updated Acme"}
	mockService.On("UpdateVendor", mock.Anything, "vendor-1", mock.AnythingOfType("*models.UpdateVendorRequest")).Return(resp, nil)

	body := `{"name": "Updated Acme"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/vendors/vendor-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateVendor_NotFound(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	mockService.On("UpdateVendor", mock.Anything, "bad", mock.AnythingOfType("*models.UpdateVendorRequest")).
		Return(nil, errors.New("vendor not found"))

	body := `{"name": "Test"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/vendors/bad", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === UpdateVendorStatus Handler Tests ===

func TestHandler_UpdateVendorStatus_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	resp := &models.VendorResponse{ID: "vendor-1", Status: models.StatusApproved}
	mockService.On("UpdateVendorStatus", mock.Anything, "vendor-1", mock.AnythingOfType("*models.UpdateVendorStatusRequest")).Return(resp, nil)

	body := `{"status": "approved"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/vendors/vendor-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateVendorStatus_InvalidTransition(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	mockService.On("UpdateVendorStatus", mock.Anything, "vendor-1", mock.AnythingOfType("*models.UpdateVendorStatusRequest")).
		Return(nil, errors.New("invalid status transition from pending to suspended"))

	body := `{"status": "suspended"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/vendors/vendor-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === GetVendorOrders Handler Tests ===

func TestHandler_GetVendorOrders_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	orders := []models.VendorOrderResponse{{ID: "vo-1", OrderID: "order-1", Amount: 100}}
	mockService.On("GetVendorOrders", mock.Anything, "vendor-1", 1, 20).Return(orders, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/vendor-1/orders", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === GetVendorAnalytics Handler Tests ===

func TestHandler_GetVendorAnalytics_Success(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	analytics := &models.VendorAnalyticsResponse{
		VendorID: "vendor-1", TotalRevenue: 5000, NetEarnings: 4500,
	}
	mockService.On("GetVendorAnalytics", mock.Anything, "vendor-1").Return(analytics, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/vendor-1/analytics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.VendorAnalyticsResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 5000.0, result.TotalRevenue)
}

func TestHandler_GetVendorAnalytics_NotFound(t *testing.T) {
	mockService := new(MockVendorService)
	router := setupRouter(mockService)

	mockService.On("GetVendorAnalytics", mock.Anything, "bad").Return(nil, errors.New("vendor not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vendors/bad/analytics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
