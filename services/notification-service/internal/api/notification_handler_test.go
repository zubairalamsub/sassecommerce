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

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNotificationService implements service.NotificationService for handler tests
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendNotification(ctx context.Context, req *models.SendNotificationRequest) (*models.NotificationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationResponse), args.Error(1)
}

func (m *MockNotificationService) GetNotification(ctx context.Context, id string) (*models.NotificationResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationResponse), args.Error(1)
}

func (m *MockNotificationService) GetUserNotifications(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.NotificationResponse, int64, error) {
	args := m.Called(ctx, tenantID, userID, page, pageSize)
	return args.Get(0).([]models.NotificationResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockNotificationService) GetPreference(ctx context.Context, tenantID, userID string) (*models.UserPreferenceResponse, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreferenceResponse), args.Error(1)
}

func (m *MockNotificationService) UpdatePreference(ctx context.Context, tenantID, userID string, req *models.UpdatePreferenceRequest) (*models.UserPreferenceResponse, error) {
	args := m.Called(ctx, tenantID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserPreferenceResponse), args.Error(1)
}

func setupRouter(mockService *MockNotificationService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewNotificationHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

func createTestNotificationResponse() *models.NotificationResponse {
	now := time.Now().UTC()
	return &models.NotificationResponse{
		ID:            "notif-1",
		TenantID:      "tenant-1",
		UserID:        "user-1",
		Channel:       models.ChannelEmail,
		Type:          models.TypeOrderConfirmation,
		Status:        models.StatusSent,
		Subject:       "Order Confirmed",
		Body:          "Your order has been confirmed.",
		Recipient:     "user@example.com",
		ReferenceID:   "order-1",
		ReferenceType: "order",
		SentAt:        &now,
		CreatedAt:     now,
	}
}

// === SendNotification Handler Tests ===

func TestHandler_SendNotification_Success(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	resp := createTestNotificationResponse()
	mockService.On("SendNotification", mock.Anything, mock.AnythingOfType("*models.SendNotificationRequest")).Return(resp, nil)

	body := `{
		"tenant_id": "tenant-1",
		"user_id": "user-1",
		"channel": "email",
		"type": "order_confirmation",
		"subject": "Order Confirmed",
		"body": "Your order has been confirmed.",
		"recipient": "user@example.com"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/send", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.NotificationResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "notif-1", result.ID)
	assert.Equal(t, models.StatusSent, result.Status)
}

func TestHandler_SendNotification_BadRequest(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	body := `{"tenant_id": "tenant-1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/send", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SendNotification_ServiceError(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	mockService.On("SendNotification", mock.Anything, mock.AnythingOfType("*models.SendNotificationRequest")).
		Return(nil, errors.New("provider error"))

	body := `{
		"tenant_id": "tenant-1",
		"user_id": "user-1",
		"channel": "email",
		"type": "order_confirmation",
		"subject": "Test",
		"body": "Test",
		"recipient": "user@example.com"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notifications/send", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// === GetNotification Handler Tests ===

func TestHandler_GetNotification_Success(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	resp := createTestNotificationResponse()
	mockService.On("GetNotification", mock.Anything, "notif-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/notif-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.NotificationResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "notif-1", result.ID)
}

func TestHandler_GetNotification_NotFound(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	mockService.On("GetNotification", mock.Anything, "nonexistent").Return(nil, errors.New("notification not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetUserNotifications Handler Tests ===

func TestHandler_GetUserNotifications_Success(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	responses := []models.NotificationResponse{*createTestNotificationResponse()}
	mockService.On("GetUserNotifications", mock.Anything, "tenant-1", "user-1", 1, 20).Return(responses, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/user/user-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result ListNotificationsResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, int64(1), result.Pagination.Total)
}

func TestHandler_GetUserNotifications_MissingTenantID(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/user/user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetUserNotifications_Pagination(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	responses := []models.NotificationResponse{*createTestNotificationResponse()}
	mockService.On("GetUserNotifications", mock.Anything, "tenant-1", "user-1", 2, 10).Return(responses, int64(25), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/user/user-1?tenant_id=tenant-1&page=2&page_size=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result ListNotificationsResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 2, result.Pagination.Page)
	assert.Equal(t, 10, result.Pagination.PageSize)
	assert.Equal(t, int64(25), result.Pagination.Total)
	assert.Equal(t, int64(3), result.Pagination.TotalPages)
}

// === MarkAsRead Handler Tests ===

func TestHandler_MarkAsRead_Success(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	mockService.On("MarkAsRead", mock.Anything, "notif-1").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/notif-1/read", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_MarkAsRead_NotFound(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	mockService.On("MarkAsRead", mock.Anything, "nonexistent").Return(errors.New("notification not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/nonexistent/read", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetPreference Handler Tests ===

func TestHandler_GetPreference_Success(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	pref := &models.UserPreferenceResponse{
		ID:           "pref-1",
		TenantID:     "tenant-1",
		UserID:       "user-1",
		EmailEnabled: true,
		SMSEnabled:   false,
		PushEnabled:  true,
		OptedOut:     []models.NotificationType{},
	}
	mockService.On("GetPreference", mock.Anything, "tenant-1", "user-1").Return(pref, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/preferences/user-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.UserPreferenceResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.True(t, result.EmailEnabled)
	assert.False(t, result.SMSEnabled)
}

func TestHandler_GetPreference_MissingTenantID(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/preferences/user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === UpdatePreference Handler Tests ===

func TestHandler_UpdatePreference_Success(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	pref := &models.UserPreferenceResponse{
		ID:           "pref-1",
		TenantID:     "tenant-1",
		UserID:       "user-1",
		EmailEnabled: false,
		SMSEnabled:   true,
		PushEnabled:  true,
		OptedOut:     []models.NotificationType{},
	}
	mockService.On("UpdatePreference", mock.Anything, "tenant-1", "user-1", mock.AnythingOfType("*models.UpdatePreferenceRequest")).Return(pref, nil)

	body := `{"email_enabled": false}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/preferences/user-1?tenant_id=tenant-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.UserPreferenceResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.False(t, result.EmailEnabled)
}

func TestHandler_UpdatePreference_MissingTenantID(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	body := `{"email_enabled": false}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/preferences/user-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdatePreference_ServiceError(t *testing.T) {
	mockService := new(MockNotificationService)
	router := setupRouter(mockService)

	mockService.On("UpdatePreference", mock.Anything, "tenant-1", "user-1", mock.AnythingOfType("*models.UpdatePreferenceRequest")).
		Return(nil, errors.New("db error"))

	body := `{"email_enabled": false}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/preferences/user-1?tenant_id=tenant-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
