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

	"github.com/ecommerce/recommendation-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRecommendationService struct {
	mock.Mock
}

func (m *MockRecommendationService) GetUserRecommendations(ctx context.Context, tenantID, userID string, limit int) (*models.RecommendationResponse, error) {
	args := m.Called(ctx, tenantID, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RecommendationResponse), args.Error(1)
}

func (m *MockRecommendationService) GetProductRecommendations(ctx context.Context, tenantID, productID string, limit int) (*models.RecommendationResponse, error) {
	args := m.Called(ctx, tenantID, productID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RecommendationResponse), args.Error(1)
}

func (m *MockRecommendationService) TrainModel(ctx context.Context, tenantID string) (*models.TrainingJobResponse, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TrainingJobResponse), args.Error(1)
}

func (m *MockRecommendationService) GetTrainingJob(ctx context.Context, id string) (*models.TrainingJobResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TrainingJobResponse), args.Error(1)
}

func (m *MockRecommendationService) RecordInteraction(ctx context.Context, tenantID, userID, productID, interactionType string) error {
	args := m.Called(ctx, tenantID, userID, productID, interactionType)
	return args.Error(0)
}

func setupRouter(mockService *MockRecommendationService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewRecommendationHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

// === GetUserRecommendations Handler Tests ===

func TestHandler_GetUserRecommendations_Success(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	resp := &models.RecommendationResponse{
		UserID: "user-1",
		Recommendations: []models.ProductRecommendation{
			{ProductID: "p-1", Score: 10, Reason: "collaborative_filtering"},
		},
		Strategy:    "collaborative_filtering",
		GeneratedAt: time.Now().UTC(),
	}
	mockService.On("GetUserRecommendations", mock.Anything, "tenant-1", "user-1", 10).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/user/user-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.RecommendationResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "user-1", result.UserID)
	assert.Len(t, result.Recommendations, 1)
}

func TestHandler_GetUserRecommendations_MissingTenantID(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/user/user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetUserRecommendations_ServiceFailure(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	mockService.On("GetUserRecommendations", mock.Anything, "tenant-1", "user-1", 10).Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/user/user-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetUserRecommendations_CustomLimit(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	resp := &models.RecommendationResponse{
		UserID:          "user-1",
		Recommendations: []models.ProductRecommendation{},
		Strategy:        "popular",
		GeneratedAt:     time.Now().UTC(),
	}
	mockService.On("GetUserRecommendations", mock.Anything, "tenant-1", "user-1", 5).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/user/user-1?tenant_id=tenant-1&limit=5", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === GetProductRecommendations Handler Tests ===

func TestHandler_GetProductRecommendations_Success(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	resp := &models.RecommendationResponse{
		ProductID: "p-1",
		Recommendations: []models.ProductRecommendation{
			{ProductID: "p-2", Score: 0.9, Reason: "co_purchase"},
		},
		Strategy:    "content_based",
		GeneratedAt: time.Now().UTC(),
	}
	mockService.On("GetProductRecommendations", mock.Anything, "tenant-1", "p-1", 10).Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/product/p-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.RecommendationResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "p-1", result.ProductID)
}

func TestHandler_GetProductRecommendations_MissingTenantID(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/product/p-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetProductRecommendations_ServiceFailure(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	mockService.On("GetProductRecommendations", mock.Anything, "tenant-1", "p-1", 10).Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/product/p-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === TrainModel Handler Tests ===

func TestHandler_TrainModel_Success(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	resp := &models.TrainingJobResponse{
		ID:       "job-1",
		TenantID: "tenant-1",
		Status:   "completed",
	}
	mockService.On("TrainModel", mock.Anything, "tenant-1").Return(resp, nil)

	body := `{"tenant_id": "tenant-1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/recommendations/train", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestHandler_TrainModel_BadRequest(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	body := `{}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/recommendations/train", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TrainModel_ServiceFailure(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	mockService.On("TrainModel", mock.Anything, "tenant-1").Return(nil, errors.New("training failed"))

	body := `{"tenant_id": "tenant-1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/recommendations/train", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// === GetTrainingJob Handler Tests ===

func TestHandler_GetTrainingJob_Success(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	resp := &models.TrainingJobResponse{
		ID:       "job-1",
		TenantID: "tenant-1",
		Status:   "completed",
	}
	mockService.On("GetTrainingJob", mock.Anything, "job-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/train/job-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetTrainingJob_NotFound(t *testing.T) {
	mockService := new(MockRecommendationService)
	router := setupRouter(mockService)

	mockService.On("GetTrainingJob", mock.Anything, "bad").Return(nil, errors.New("training job not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/recommendations/train/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
