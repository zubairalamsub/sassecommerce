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

	"github.com/ecommerce/promotion-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPromotionService struct {
	mock.Mock
}

func (m *MockPromotionService) CreatePromotion(ctx context.Context, req *models.CreatePromotionRequest) (*models.PromotionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PromotionResponse), args.Error(1)
}

func (m *MockPromotionService) GetPromotion(ctx context.Context, id string) (*models.PromotionResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PromotionResponse), args.Error(1)
}

func (m *MockPromotionService) GetActivePromotions(ctx context.Context, tenantID string) ([]models.PromotionResponse, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]models.PromotionResponse), args.Error(1)
}

func (m *MockPromotionService) CreateCoupon(ctx context.Context, req *models.CreateCouponRequest) (*models.CouponResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CouponResponse), args.Error(1)
}

func (m *MockPromotionService) ValidateCoupon(ctx context.Context, code string, req *models.ValidateCouponRequest) (*models.ValidateCouponResponse, error) {
	args := m.Called(ctx, code, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ValidateCouponResponse), args.Error(1)
}

func (m *MockPromotionService) ApplyCoupon(ctx context.Context, req *models.ApplyCouponRequest) (*models.ValidateCouponResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ValidateCouponResponse), args.Error(1)
}

func (m *MockPromotionService) GetLoyaltyAccount(ctx context.Context, tenantID, userID string) (*models.LoyaltyAccountResponse, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoyaltyAccountResponse), args.Error(1)
}

func (m *MockPromotionService) ProcessLoyaltyPoints(ctx context.Context, req *models.LoyaltyPointsRequest) (*models.LoyaltyAccountResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoyaltyAccountResponse), args.Error(1)
}

func setupRouter(mockService *MockPromotionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewPromotionHandler(mockService, logger)
	router := gin.New()
	RegisterRoutes(router, handler)
	return router
}

// === CreatePromotion Handler Tests ===

func TestHandler_CreatePromotion_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.PromotionResponse{
		ID: "promo-1", TenantID: "tenant-1", Name: "Summer Sale",
		DiscountType: models.DiscountPercentage, DiscountValue: 20,
		Status: models.StatusActive,
		StartDate: time.Now().UTC(), EndDate: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	mockService.On("CreatePromotion", mock.Anything, mock.AnythingOfType("*models.CreatePromotionRequest")).Return(resp, nil)

	body := `{
		"tenant_id": "tenant-1", "name": "Summer Sale",
		"discount_type": "percentage", "discount_value": 20,
		"start_date": "2026-04-01T00:00:00Z", "end_date": "2026-05-01T00:00:00Z"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/promotions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreatePromotion_BadRequest(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	body := `{"name": "Sale"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/promotions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreatePromotion_ValidationError(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	mockService.On("CreatePromotion", mock.Anything, mock.AnythingOfType("*models.CreatePromotionRequest")).
		Return(nil, errors.New("end date must be after start date"))

	body := `{
		"tenant_id": "t1", "name": "Sale",
		"discount_type": "percentage", "discount_value": 10,
		"start_date": "2026-05-01T00:00:00Z", "end_date": "2026-04-01T00:00:00Z"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/promotions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === GetPromotion Handler Tests ===

func TestHandler_GetPromotion_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.PromotionResponse{ID: "promo-1", Name: "Summer Sale"}
	mockService.On("GetPromotion", mock.Anything, "promo-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/promotions/promo-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetPromotion_NotFound(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	mockService.On("GetPromotion", mock.Anything, "bad").Return(nil, errors.New("promotion not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/promotions/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetActivePromotions Handler Tests ===

func TestHandler_GetActivePromotions_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := []models.PromotionResponse{{ID: "promo-1", Name: "Sale"}}
	mockService.On("GetActivePromotions", mock.Anything, "tenant-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/promotions/active?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []models.PromotionResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Len(t, result, 1)
}

func TestHandler_GetActivePromotions_MissingTenantID(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/promotions/active", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === CreateCoupon Handler Tests ===

func TestHandler_CreateCoupon_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.CouponResponse{ID: "coupon-1", Code: "SUMMER20"}
	mockService.On("CreateCoupon", mock.Anything, mock.AnythingOfType("*models.CreateCouponRequest")).Return(resp, nil)

	body := `{"tenant_id": "tenant-1", "promotion_id": "promo-1", "code": "SUMMER20"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/coupons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateCoupon_Conflict(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	mockService.On("CreateCoupon", mock.Anything, mock.AnythingOfType("*models.CreateCouponRequest")).
		Return(nil, errors.New("coupon code already exists"))

	body := `{"tenant_id": "tenant-1", "promotion_id": "promo-1", "code": "SUMMER20"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/coupons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// === ValidateCoupon Handler Tests ===

func TestHandler_ValidateCoupon_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.ValidateCouponResponse{Valid: true, Code: "SUMMER20", DiscountAmount: 20}
	mockService.On("ValidateCoupon", mock.Anything, "SUMMER20", mock.AnythingOfType("*models.ValidateCouponRequest")).Return(resp, nil)

	body := `{"tenant_id": "tenant-1", "user_id": "user-1", "order_total": 100}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/coupons/validate/SUMMER20", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// === ApplyCoupon Handler Tests ===

func TestHandler_ApplyCoupon_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.ValidateCouponResponse{Valid: true, Code: "SUMMER20", DiscountAmount: 20}
	mockService.On("ApplyCoupon", mock.Anything, mock.AnythingOfType("*models.ApplyCouponRequest")).Return(resp, nil)

	body := `{"tenant_id": "tenant-1", "user_id": "user-1", "order_id": "order-1", "order_total": 100, "code": "SUMMER20"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/coupons/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ApplyCoupon_BadRequest(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	body := `{"tenant_id": "t1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/coupons/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Loyalty Handler Tests ===

func TestHandler_GetLoyaltyAccount_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.LoyaltyAccountResponse{UserID: "user-1", Points: 500, TierLevel: "bronze"}
	mockService.On("GetLoyaltyAccount", mock.Anything, "tenant-1", "user-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/loyalty/user-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetLoyaltyAccount_MissingTenantID(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/loyalty/user-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessLoyaltyPoints_Success(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	resp := &models.LoyaltyAccountResponse{UserID: "user-1", Points: 600, TierLevel: "bronze"}
	mockService.On("ProcessLoyaltyPoints", mock.Anything, mock.AnythingOfType("*models.LoyaltyPointsRequest")).Return(resp, nil)

	body := `{"tenant_id": "tenant-1", "user_id": "user-1", "type": "earn", "points": 100}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/loyalty/points", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ProcessLoyaltyPoints_InsufficientPoints(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	mockService.On("ProcessLoyaltyPoints", mock.Anything, mock.AnythingOfType("*models.LoyaltyPointsRequest")).
		Return(nil, errors.New("insufficient loyalty points: have 100, need 500"))

	body := `{"tenant_id": "tenant-1", "user_id": "user-1", "type": "redeem", "points": 500}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/loyalty/points", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ProcessLoyaltyPoints_BadRequest(t *testing.T) {
	mockService := new(MockPromotionService)
	router := setupRouter(mockService)

	body := `{"tenant_id": "t1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/loyalty/points", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
