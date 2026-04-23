package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/analytics-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAnalyticsService struct {
	mock.Mock
}

func (m *MockAnalyticsService) GetSalesReport(ctx context.Context, req *models.SalesReportRequest) (*models.SalesReportResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SalesReportResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetCustomerInsights(ctx context.Context, req *models.CustomerInsightsRequest) (*models.CustomerInsightsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CustomerInsightsResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetProductPerformance(ctx context.Context, req *models.ProductPerformanceRequest) (*models.ProductPerformanceResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProductPerformanceResponse), args.Error(1)
}

func (m *MockAnalyticsService) CreateReport(ctx context.Context, req *models.CreateReportRequest) (*models.CustomReportResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CustomReportResponse), args.Error(1)
}

func (m *MockAnalyticsService) GetReport(ctx context.Context, id string) (*models.CustomReportResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CustomReportResponse), args.Error(1)
}

func (m *MockAnalyticsService) ListReports(ctx context.Context, tenantID string, page, pageSize int) ([]models.CustomReportResponse, int64, error) {
	args := m.Called(ctx, tenantID, page, pageSize)
	return args.Get(0).([]models.CustomReportResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockAnalyticsService) RecordSale(ctx context.Context, tenantID, orderID, userID, vendorID string, amount float64, channel string) error {
	args := m.Called(ctx, tenantID, orderID, userID, vendorID, amount, channel)
	return args.Error(0)
}

func (m *MockAnalyticsService) RecordCustomerActivity(ctx context.Context, tenantID, userID, eventType, orderID string, amount float64) error {
	args := m.Called(ctx, tenantID, userID, eventType, orderID, amount)
	return args.Error(0)
}

func (m *MockAnalyticsService) RecordProductActivity(ctx context.Context, tenantID, productID, eventType string, quantity int, revenue float64) error {
	args := m.Called(ctx, tenantID, productID, eventType, quantity, revenue)
	return args.Error(0)
}

func setupRouter(mockService *MockAnalyticsService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewAnalyticsHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

// === GetSalesReport Handler Tests ===

func TestHandler_GetSalesReport_Success(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	report := &models.SalesReportResponse{TenantID: "tenant-1", TotalRevenue: 50000, TotalOrders: 100}
	mockService.On("GetSalesReport", mock.Anything, mock.AnythingOfType("*models.SalesReportRequest")).Return(report, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/sales?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.SalesReportResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 50000.0, result.TotalRevenue)
}

func TestHandler_GetSalesReport_MissingTenantID(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/sales", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSalesReport_ServiceFailure(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	mockService.On("GetSalesReport", mock.Anything, mock.AnythingOfType("*models.SalesReportRequest")).Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/sales?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === GetCustomerInsights Handler Tests ===

func TestHandler_GetCustomerInsights_Success(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	insights := &models.CustomerInsightsResponse{TenantID: "tenant-1", TotalCustomers: 500}
	mockService.On("GetCustomerInsights", mock.Anything, mock.AnythingOfType("*models.CustomerInsightsRequest")).Return(insights, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/customers?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetCustomerInsights_MissingTenantID(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/customers", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === GetProductPerformance Handler Tests ===

func TestHandler_GetProductPerformance_Success(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	perf := &models.ProductPerformanceResponse{TenantID: "tenant-1", TotalProducts: 200}
	mockService.On("GetProductPerformance", mock.Anything, mock.AnythingOfType("*models.ProductPerformanceRequest")).Return(perf, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/products?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetProductPerformance_MissingTenantID(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/products", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === CreateReport Handler Tests ===

func TestHandler_CreateReport_Success(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	resp := &models.CustomReportResponse{ID: "r-1", Name: "Monthly Sales", Status: "completed"}
	mockService.On("CreateReport", mock.Anything, mock.AnythingOfType("*models.CreateReportRequest")).Return(resp, nil)

	body := `{"tenant_id":"tenant-1","name":"Monthly Sales","report_type":"sales","date_from":"2026-03-01","date_to":"2026-03-31"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/analytics/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateReport_BadRequest(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	body := `{"name":"Test"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/analytics/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateReport_InvalidType(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	mockService.On("CreateReport", mock.Anything, mock.AnythingOfType("*models.CreateReportRequest")).
		Return(nil, errors.New("invalid report_type, must be one of: sales, customers, products"))

	body := `{"tenant_id":"tenant-1","name":"Test","report_type":"bad","date_from":"2026-03-01","date_to":"2026-03-31"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/analytics/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateReport_DateValidation(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	mockService.On("CreateReport", mock.Anything, mock.AnythingOfType("*models.CreateReportRequest")).
		Return(nil, errors.New("date_to must be after date_from"))

	body := `{"tenant_id":"tenant-1","name":"Test","report_type":"sales","date_from":"2026-04-01","date_to":"2026-03-01"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/analytics/reports", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === GetReport Handler Tests ===

func TestHandler_GetReport_Success(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	resp := &models.CustomReportResponse{ID: "r-1", Name: "Monthly Sales", Status: "completed"}
	mockService.On("GetReport", mock.Anything, "r-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/reports/r-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetReport_NotFound(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	mockService.On("GetReport", mock.Anything, "bad").Return(nil, errors.New("report not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/reports/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ListReports Handler Tests ===

func TestHandler_ListReports_Success(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	reports := []models.CustomReportResponse{{ID: "r-1", Name: "Report 1"}}
	mockService.On("ListReports", mock.Anything, "tenant-1", 1, 20).Return(reports, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/reports?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, float64(1), result["total"])
}

func TestHandler_ListReports_MissingTenantID(t *testing.T) {
	mockService := new(MockAnalyticsService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/reports", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
