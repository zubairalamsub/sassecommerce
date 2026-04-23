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

	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockShippingService implements service.ShippingService for handler tests
type MockShippingService struct {
	mock.Mock
}

func (m *MockShippingService) CreateShipment(ctx context.Context, req *models.CreateShipmentRequest) (*models.ShipmentResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ShipmentResponse), args.Error(1)
}

func (m *MockShippingService) GetShipment(ctx context.Context, id string) (*models.ShipmentResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ShipmentResponse), args.Error(1)
}

func (m *MockShippingService) GetShipmentByTracking(ctx context.Context, trackingNumber string) (*models.ShipmentResponse, error) {
	args := m.Called(ctx, trackingNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ShipmentResponse), args.Error(1)
}

func (m *MockShippingService) GetShipmentByOrderID(ctx context.Context, tenantID, orderID string) (*models.ShipmentResponse, error) {
	args := m.Called(ctx, tenantID, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ShipmentResponse), args.Error(1)
}

func (m *MockShippingService) ListShipments(ctx context.Context, tenantID string, page, pageSize int, status string) ([]models.ShipmentResponse, int64, error) {
	args := m.Called(ctx, tenantID, page, pageSize, status)
	return args.Get(0).([]models.ShipmentResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockShippingService) UpdateStatus(ctx context.Context, id string, req *models.UpdateStatusRequest) (*models.ShipmentResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ShipmentResponse), args.Error(1)
}

func (m *MockShippingService) CancelShipment(ctx context.Context, id string) (*models.ShipmentResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ShipmentResponse), args.Error(1)
}

func (m *MockShippingService) CalculateRates(ctx context.Context, req *models.CalculateRateRequest) (*models.RateCalculationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RateCalculationResponse), args.Error(1)
}

func setupRouter(mockService *MockShippingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewShippingHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

func createTestResponse() *models.ShipmentResponse {
	now := time.Now().UTC()
	estimated := now.AddDate(0, 0, 5)
	return &models.ShipmentResponse{
		ID:             "shipment-1",
		TenantID:       "tenant-1",
		OrderID:        "order-1",
		Carrier:        "pathao",
		TrackingNumber: "PA1234567890",
		ServiceType:    "standard",
		LabelURL:       "https://labels.simulated.dev/pathao/PA1234567890.pdf",
		Status:         models.StatusLabelCreated,
		WeightOz:       2.0,
		ShippingCost:   80.0,
		Currency:       "BDT",
		FromAddress: models.AddressResponse{
			Name: "Warehouse", Street: "BSCIC, Gazipur", City: "Gazipur", State: "Dhaka", PostalCode: "1700", Country: "BD",
		},
		ToAddress: models.AddressResponse{
			Name: "Rahim Uddin", Street: "Dhanmondi 27", City: "Dhaka", State: "Dhaka", PostalCode: "1209", Country: "BD",
		},
		EstimatedDelivery: &estimated,
		SignatureRequired: true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// === CreateShipment Handler Tests ===

func TestHandler_CreateShipment_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	resp := createTestResponse()
	mockService.On("CreateShipment", mock.Anything, mock.AnythingOfType("*models.CreateShipmentRequest")).Return(resp, nil)

	body := `{
		"tenant_id": "tenant-1",
		"order_id": "order-1",
		"carrier": "pathao",
		"service_type": "standard",
		"weight_oz": 2.0,
		"from_address": {"name":"Warehouse","street":"BSCIC, Gazipur","city":"Gazipur","state":"Dhaka","postal_code":"1700","country":"BD"},
		"to_address": {"name":"Rahim Uddin","street":"Dhanmondi 27","city":"Dhaka","state":"Dhaka","postal_code":"1209","country":"BD"}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.ShipmentResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, "PA1234567890", result.TrackingNumber)
}

func TestHandler_CreateShipment_BadRequest(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	body := `{"invalid": "json"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateShipment_ServiceError(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("CreateShipment", mock.Anything, mock.AnythingOfType("*models.CreateShipmentRequest")).
		Return(nil, errors.New("carrier unavailable"))

	body := `{
		"tenant_id": "tenant-1",
		"order_id": "order-1",
		"carrier": "pathao",
		"from_address": {"name":"W","street":"1","city":"Gazipur","state":"Dhaka","postal_code":"1700","country":"BD"},
		"to_address": {"name":"R","street":"4","city":"Dhaka","state":"Dhaka","postal_code":"1205","country":"BD"}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// === GetShipment Handler Tests ===

func TestHandler_GetShipment_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	resp := createTestResponse()
	mockService.On("GetShipment", mock.Anything, "shipment-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/shipment-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.ShipmentResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "shipment-1", result.ID)
}

func TestHandler_GetShipment_NotFound(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("GetShipment", mock.Anything, "nonexistent").Return(nil, errors.New("shipment not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetByTracking Handler Tests ===

func TestHandler_GetShipmentByTracking_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	resp := createTestResponse()
	mockService.On("GetShipmentByTracking", mock.Anything, "PA1234567890").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/tracking/PA1234567890", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetShipmentByTracking_NotFound(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("GetShipmentByTracking", mock.Anything, "INVALID").Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/tracking/INVALID", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetByOrderID Handler Tests ===

func TestHandler_GetShipmentByOrderID_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	resp := createTestResponse()
	mockService.On("GetShipmentByOrderID", mock.Anything, "tenant-1", "order-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/order/order-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetShipmentByOrderID_MissingTenantID(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/order/order-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetShipmentByOrderID_NotFound(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("GetShipmentByOrderID", mock.Anything, "tenant-1", "bad").Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments/order/bad?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ListShipments Handler Tests ===

func TestHandler_ListShipments_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	responses := []models.ShipmentResponse{*createTestResponse()}
	mockService.On("ListShipments", mock.Anything, "tenant-1", 1, 20, "").Return(responses, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result ListShipmentsResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, int64(1), result.Pagination.Total)
}

func TestHandler_ListShipments_MissingTenantID(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListShipments_WithStatusFilter(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	responses := []models.ShipmentResponse{*createTestResponse()}
	mockService.On("ListShipments", mock.Anything, "tenant-1", 1, 20, "in_transit").Return(responses, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments?tenant_id=tenant-1&status=in_transit", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ListShipments_Pagination(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	responses := []models.ShipmentResponse{*createTestResponse()}
	mockService.On("ListShipments", mock.Anything, "tenant-1", 2, 10, "").Return(responses, int64(15), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/shipments?tenant_id=tenant-1&page=2&page_size=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result ListShipmentsResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 2, result.Pagination.Page)
	assert.Equal(t, 10, result.Pagination.PageSize)
	assert.Equal(t, int64(15), result.Pagination.Total)
	assert.Equal(t, int64(2), result.Pagination.TotalPages)
}

// === UpdateStatus Handler Tests ===

func TestHandler_UpdateStatus_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	resp := createTestResponse()
	resp.Status = models.StatusInTransit
	mockService.On("UpdateStatus", mock.Anything, "shipment-1", mock.AnythingOfType("*models.UpdateStatusRequest")).Return(resp, nil)

	body := `{"status": "in_transit", "location": "Gazipur, Dhaka", "description": "In transit"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/shipments/shipment-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateStatus_BadRequest(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	body := `{"invalid": true}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/shipments/shipment-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateStatus_InvalidTransition(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("UpdateStatus", mock.Anything, "shipment-1", mock.AnythingOfType("*models.UpdateStatusRequest")).
		Return(nil, errors.New("invalid status transition"))

	body := `{"status": "delivered"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/shipments/shipment-1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// === CancelShipment Handler Tests ===

func TestHandler_CancelShipment_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	resp := createTestResponse()
	resp.Status = models.StatusCancelled
	mockService.On("CancelShipment", mock.Anything, "shipment-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/shipment-1/cancel", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.ShipmentResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, models.StatusCancelled, result.Status)
}

func TestHandler_CancelShipment_Error(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("CancelShipment", mock.Anything, "shipment-1").
		Return(nil, errors.New("can only be cancelled when pending"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/shipments/shipment-1/cancel", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// === CalculateRates Handler Tests ===

func TestHandler_CalculateRates_Success(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	ratesResp := &models.RateCalculationResponse{
		Rates: []models.CarrierRateResponse{
			{Carrier: "pathao", ServiceType: "standard", Rate: 60.0, Currency: "BDT", EstimatedDays: 5},
			{Carrier: "steadfast", ServiceType: "standard", Rate: 70.0, Currency: "BDT", EstimatedDays: 5},
		},
	}
	mockService.On("CalculateRates", mock.Anything, mock.AnythingOfType("*models.CalculateRateRequest")).Return(ratesResp, nil)

	body := `{
		"tenant_id": "tenant-1",
		"from_address": {"name":"W","street":"BSCIC","city":"Gazipur","state":"Dhaka","postal_code":"1700","country":"BD"},
		"to_address": {"name":"C","street":"Dhanmondi","city":"Dhaka","state":"Dhaka","postal_code":"1209","country":"BD"},
		"weight_oz": 1.0
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.RateCalculationResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Len(t, result.Rates, 2)
}

func TestHandler_CalculateRates_BadRequest(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	body := `{"invalid": true}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CalculateRates_ServiceError(t *testing.T) {
	mockService := new(MockShippingService)
	router := setupRouter(mockService)

	mockService.On("CalculateRates", mock.Anything, mock.AnythingOfType("*models.CalculateRateRequest")).
		Return(nil, errors.New("service error"))

	body := `{
		"tenant_id": "tenant-1",
		"from_address": {"name":"W","street":"BSCIC","city":"Gazipur","state":"Dhaka","postal_code":"1700","country":"BD"},
		"to_address": {"name":"C","street":"Dhanmondi","city":"Dhaka","state":"Dhaka","postal_code":"1209","country":"BD"},
		"weight_oz": 1.0
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/rates", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
