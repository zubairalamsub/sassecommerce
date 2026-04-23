package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetConfig(ctx context.Context, namespace, key, environment, tenantID string) (*models.ConfigEntryResponse, error) {
	args := m.Called(ctx, namespace, key, environment, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigEntryResponse), args.Error(1)
}

func (m *MockConfigService) SetConfig(ctx context.Context, req *models.SetConfigRequest) (*models.ConfigEntryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConfigEntryResponse), args.Error(1)
}

func (m *MockConfigService) DeleteConfig(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockConfigService) ListByNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntryResponse, error) {
	args := m.Called(ctx, namespace, environment, tenantID)
	return args.Get(0).([]models.ConfigEntryResponse), args.Error(1)
}

func (m *MockConfigService) ListNamespaces(ctx context.Context) (*models.NamespaceListResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NamespaceListResponse), args.Error(1)
}

func (m *MockConfigService) SearchConfigs(ctx context.Context, query, namespace, environment string, page, pageSize int) ([]models.ConfigEntryResponse, int64, error) {
	args := m.Called(ctx, query, namespace, environment, page, pageSize)
	return args.Get(0).([]models.ConfigEntryResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockConfigService) BulkGet(ctx context.Context, req *models.BulkGetRequest, environment, tenantID string) ([]models.ConfigEntryResponse, error) {
	args := m.Called(ctx, req, environment, tenantID)
	return args.Get(0).([]models.ConfigEntryResponse), args.Error(1)
}

func (m *MockConfigService) BulkSet(ctx context.Context, req *models.BulkSetRequest) ([]models.ConfigEntryResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]models.ConfigEntryResponse), args.Error(1)
}

func (m *MockConfigService) GetAuditLog(ctx context.Context, namespace, key string, page, pageSize int) ([]models.ConfigAuditResponse, int64, error) {
	args := m.Called(ctx, namespace, key, page, pageSize)
	return args.Get(0).([]models.ConfigAuditResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockConfigService) GetConfigHistory(ctx context.Context, configID string, page, pageSize int) ([]models.ConfigAuditResponse, int64, error) {
	args := m.Called(ctx, configID, page, pageSize)
	return args.Get(0).([]models.ConfigAuditResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockConfigService) ExportNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntryResponse, error) {
	args := m.Called(ctx, namespace, environment, tenantID)
	return args.Get(0).([]models.ConfigEntryResponse), args.Error(1)
}

func setupRouter(mockService *MockConfigService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewConfigHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

// === GetConfig Handler Tests ===

func TestHandler_GetConfig_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	resp := &models.ConfigEntryResponse{ID: "c-1", Namespace: "global", Key: "page_size", Value: "20"}
	mockService.On("GetConfig", mock.Anything, "global", "page_size", "all", "").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/get?namespace=global&key=page_size", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.ConfigEntryResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "20", result.Value)
}

func TestHandler_GetConfig_MissingParams(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/get?namespace=global", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetConfig_NotFound(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	mockService.On("GetConfig", mock.Anything, "bad", "key", "all", "").Return(nil, errors.New("config not found: bad.key"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/get?namespace=bad&key=key", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === SetConfig Handler Tests ===

func TestHandler_SetConfig_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	resp := &models.ConfigEntryResponse{ID: "c-1", Namespace: "test", Key: "key1", Value: "value1", Version: 1}
	mockService.On("SetConfig", mock.Anything, mock.AnythingOfType("*models.SetConfigRequest")).Return(resp, nil)

	body := `{"namespace":"test","key":"key1","value":"value1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/config/set", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SetConfig_BadRequest(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	body := `{"namespace":"test"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/config/set", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetConfig_InvalidType(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	mockService.On("SetConfig", mock.Anything, mock.AnythingOfType("*models.SetConfigRequest")).
		Return(nil, errors.New("invalid value_type"))

	body := `{"namespace":"test","key":"k1","value":"v1","value_type":"invalid"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/config/set", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === DeleteConfig Handler Tests ===

func TestHandler_DeleteConfig_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	mockService.On("DeleteConfig", mock.Anything, "c-1").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/config/c-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_DeleteConfig_NotFound(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	mockService.On("DeleteConfig", mock.Anything, "bad").Return(errors.New("config not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/config/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ListByNamespace Handler Tests ===

func TestHandler_ListByNamespace_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	configs := []models.ConfigEntryResponse{
		{ID: "c-1", Namespace: "kafka", Key: "topics.order_events", Value: "order-events"},
	}
	mockService.On("ListByNamespace", mock.Anything, "kafka", "", "").Return(configs, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/namespace/kafka", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, float64(1), result["count"])
}

// === ListNamespaces Handler Tests ===

func TestHandler_ListNamespaces_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	resp := &models.NamespaceListResponse{
		Namespaces: []models.NamespaceSummary{
			{Namespace: "global", Count: 15},
			{Namespace: "kafka", Count: 20},
		},
	}
	mockService.On("ListNamespaces", mock.Anything).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/namespaces", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === SearchConfigs Handler Tests ===

func TestHandler_SearchConfigs_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	configs := []models.ConfigEntryResponse{
		{ID: "c-1", Key: "carrier.fedex.base_rate", Value: "7.99"},
	}
	mockService.On("SearchConfigs", mock.Anything, "fedex", "", "", 1, 50).Return(configs, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/search?q=fedex", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, float64(1), result["total"])
}

// === BulkGet Handler Tests ===

func TestHandler_BulkGet_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	configs := []models.ConfigEntryResponse{
		{ID: "c-1", Key: "page_size", Value: "20"},
	}
	mockService.On("BulkGet", mock.Anything, mock.AnythingOfType("*models.BulkGetRequest"), "all", "").Return(configs, nil)

	body := `{"keys":[{"namespace":"global","key":"page_size"}]}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/config/bulk/get", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === BulkSet Handler Tests ===

func TestHandler_BulkSet_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	configs := []models.ConfigEntryResponse{
		{ID: "c-1", Key: "k1", Value: "v1"},
	}
	mockService.On("BulkSet", mock.Anything, mock.AnythingOfType("*models.BulkSetRequest")).Return(configs, nil)

	body := `{"entries":[{"namespace":"test","key":"k1","value":"v1"}],"updated_by":"admin"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/config/bulk/set", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === ExportNamespace Handler Tests ===

func TestHandler_ExportNamespace_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	configs := []models.ConfigEntryResponse{
		{ID: "c-1", Key: "carrier.fedex.base_rate", Value: "7.99"},
		{ID: "c-2", Key: "carrier.ups.base_rate", Value: "7.49"},
	}
	mockService.On("ExportNamespace", mock.Anything, "business.shipping", "", "").Return(configs, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/export/business.shipping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "business.shipping", result["namespace"])
	assert.Equal(t, float64(2), result["count"])
}

// === GetAuditLog Handler Tests ===

func TestHandler_GetAuditLog_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	logs := []models.ConfigAuditResponse{
		{ID: "a-1", Action: "create", NewValue: "10"},
	}
	mockService.On("GetAuditLog", mock.Anything, "test", "", 1, 50).Return(logs, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/audit?namespace=test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === GetConfigHistory Handler Tests ===

func TestHandler_GetConfigHistory_Success(t *testing.T) {
	mockService := new(MockConfigService)
	router := setupRouter(mockService)

	logs := []models.ConfigAuditResponse{
		{ID: "a-1", Action: "update", OldValue: "10", NewValue: "15"},
	}
	mockService.On("GetConfigHistory", mock.Anything, "c-1", 1, 50).Return(logs, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/config/audit/c-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
