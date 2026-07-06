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

	"github.com/ecommerce/review-service/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockReviewService struct {
	mock.Mock
}

func (m *MockReviewService) CreateReview(ctx context.Context, tenantID, userID string, req *models.CreateReviewRequest) (*models.ReviewResponse, error) {
	args := m.Called(ctx, tenantID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewResponse), args.Error(1)
}

func (m *MockReviewService) GetReview(ctx context.Context, tenantID, id string) (*models.ReviewResponse, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewResponse), args.Error(1)
}

func (m *MockReviewService) GetProductReviews(ctx context.Context, tenantID, productID string, page, pageSize int) ([]models.ReviewResponse, int64, error) {
	args := m.Called(ctx, tenantID, productID, page, pageSize)
	return args.Get(0).([]models.ReviewResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockReviewService) GetUserReviews(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.ReviewResponse, int64, error) {
	args := m.Called(ctx, tenantID, userID, page, pageSize)
	return args.Get(0).([]models.ReviewResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockReviewService) UpdateReview(ctx context.Context, tenantID, userID, id string, req *models.UpdateReviewRequest) (*models.ReviewResponse, error) {
	args := m.Called(ctx, tenantID, userID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewResponse), args.Error(1)
}

func (m *MockReviewService) DeleteReview(ctx context.Context, tenantID, userID, id string, isStaff bool) error {
	args := m.Called(ctx, tenantID, userID, id, isStaff)
	return args.Error(0)
}

func (m *MockReviewService) ModerateReview(ctx context.Context, tenantID, id string, req *models.ModerateReviewRequest) (*models.ReviewResponse, error) {
	args := m.Called(ctx, tenantID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewResponse), args.Error(1)
}

func (m *MockReviewService) AddHelpfulVote(ctx context.Context, tenantID, id string, req *models.HelpfulVoteRequest) error {
	args := m.Called(ctx, tenantID, id, req)
	return args.Error(0)
}

func (m *MockReviewService) RespondToReview(ctx context.Context, tenantID, id string, req *models.SellerResponseRequest) (*models.ReviewResponse, error) {
	args := m.Called(ctx, tenantID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewResponse), args.Error(1)
}

func (m *MockReviewService) GetProductSummary(ctx context.Context, tenantID, productID string) (*models.ReviewSummaryResponse, error) {
	args := m.Called(ctx, tenantID, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReviewSummaryResponse), args.Error(1)
}

func setupRouter(mockService *MockReviewService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewReviewHandler(mockService, logger)
	router := gin.New()
	// Stub the shared Auth middleware: populate tenant_id/user_id/user_role in
	// context from test headers so handlers (and RequireRole) read the
	// JWT-derived identity, never client-controlled query params. Requests
	// without these headers simulate an anonymous/unauthenticated caller.
	router.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		if u := c.GetHeader("X-Test-User"); u != "" {
			c.Set("user_id", u)
		}
		if r := c.GetHeader("X-Test-Role"); r != "" {
			c.Set("user_role", r)
		}
		c.Next()
	})
	RegisterRoutes(router, handler)
	return router
}

// authed sets the test auth headers that the stub middleware turns into
// JWT-derived context values.
func authed(req *http.Request, tenant, user, role string) *http.Request {
	req.Header.Set("X-Test-Tenant", tenant)
	req.Header.Set("X-Test-User", user)
	if role != "" {
		req.Header.Set("X-Test-Role", role)
	}
	return req
}

func createTestReviewResponse() *models.ReviewResponse {
	return &models.ReviewResponse{
		ID:               "review-1",
		TenantID:         "tenant-1",
		ProductID:        "product-1",
		UserID:           "user-1",
		Rating:           5,
		Title:            "Great product!",
		Comment:          "Highly recommended.",
		Status:           models.StatusApproved,
		VerifiedPurchase: true,
		HelpfulCount:     3,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}

// === CreateReview Handler Tests ===

func TestHandler_CreateReview_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	resp := createTestReviewResponse()
	// Tenant and user are taken from the JWT context, never the body.
	mockService.On("CreateReview", mock.Anything, "tenant-1", "user-1", mock.AnythingOfType("*models.CreateReviewRequest")).Return(resp, nil)

	body := `{
		"product_id": "product-1",
		"rating": 5,
		"title": "Great product!",
		"comment": "Highly recommended."
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateReview_IgnoresBodyIdentity(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	resp := createTestReviewResponse()
	// Even if an attacker sends tenant_id/user_id in the body, the handler must
	// pass only the JWT-derived identity to the service.
	mockService.On("CreateReview", mock.Anything, "tenant-1", "user-1", mock.AnythingOfType("*models.CreateReviewRequest")).Return(resp, nil)

	body := `{
		"tenant_id": "attacker-tenant",
		"user_id": "victim-user",
		"product_id": "product-1",
		"rating": 5,
		"title": "Great product!",
		"comment": "Highly recommended."
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

func TestHandler_CreateReview_Unauthenticated(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	body := `{
		"product_id": "product-1", "rating": 5, "title": "Test", "comment": "Test"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "CreateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHandler_CreateReview_BadRequest(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	body := `{"title": "no product id"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateReview_Conflict(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	mockService.On("CreateReview", mock.Anything, "tenant-1", "user-1", mock.AnythingOfType("*models.CreateReviewRequest")).
		Return(nil, errors.New("user has already reviewed this product"))

	body := `{
		"product_id": "product-1", "rating": 5, "title": "Test", "comment": "Test"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// === GetReview Handler Tests ===

func TestHandler_GetReview_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	resp := createTestReviewResponse()
	mockService.On("GetReview", mock.Anything, "tenant-1", "review-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/review-1", nil)
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetReview_CrossTenantNotFound(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	// A caller in tenant-2 requesting a tenant-1 review: the tenant-scoped
	// lookup misses and the service returns not found.
	mockService.On("GetReview", mock.Anything, "tenant-2", "review-1").Return(nil, errors.New("review not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/review-1", nil)
	authed(req, "tenant-2", "user-9", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetReview_NotFound(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	mockService.On("GetReview", mock.Anything, "tenant-1", "bad").Return(nil, errors.New("review not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/bad", nil)
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetProductReviews Handler Tests ===

func TestHandler_GetProductReviews_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	reviews := []models.ReviewResponse{*createTestReviewResponse()}
	mockService.On("GetProductReviews", mock.Anything, "tenant-1", "product-1", 1, 20).Return(reviews, int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/product/product-1?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result ListReviewsResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Len(t, result.Data, 1)
}

func TestHandler_GetProductReviews_MissingTenantID(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/product/product-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === UpdateReview Handler Tests ===

func TestHandler_UpdateReview_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	resp := createTestReviewResponse()
	resp.Title = "Updated"
	// Ownership is derived from the JWT user id, not a query param.
	mockService.On("UpdateReview", mock.Anything, "tenant-1", "user-1", "review-1", mock.AnythingOfType("*models.UpdateReviewRequest")).Return(resp, nil)

	body := `{"title": "Updated"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/reviews/review-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateReview_Unauthenticated(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	body := `{"title": "Updated"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/reviews/review-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "UpdateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHandler_UpdateReview_Forbidden(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	// The JWT user is not the review owner; the service enforces ownership.
	mockService.On("UpdateReview", mock.Anything, "tenant-1", "other", "review-1", mock.AnythingOfType("*models.UpdateReviewRequest")).
		Return(nil, errors.New("unauthorized: you can only update your own reviews"))

	body := `{"title": "Hacked"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/reviews/review-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "other", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// === DeleteReview Handler Tests ===

func TestHandler_DeleteReview_Owner(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	// A customer deletes their own review: isStaff = false.
	mockService.On("DeleteReview", mock.Anything, "tenant-1", "user-1", "review-1", false).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/reviews/review-1", nil)
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_DeleteReview_Staff(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	// A moderator deletes someone else's review: isStaff = true is threaded
	// through from the JWT role.
	mockService.On("DeleteReview", mock.Anything, "tenant-1", "mod-1", "review-1", true).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/reviews/review-1", nil)
	authed(req, "tenant-1", "mod-1", "moderator")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_DeleteReview_Unauthenticated(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/reviews/review-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "DeleteReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHandler_DeleteReview_NotFound(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	mockService.On("DeleteReview", mock.Anything, "tenant-1", "user-1", "bad", false).Return(errors.New("review not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/reviews/bad", nil)
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ModerateReview Handler Tests ===

func TestHandler_ModerateReview_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	resp := createTestReviewResponse()
	resp.Status = models.StatusRejected
	mockService.On("ModerateReview", mock.Anything, "tenant-1", "review-1", mock.AnythingOfType("*models.ModerateReviewRequest")).Return(resp, nil)

	body := `{"status": "rejected", "reject_reason": "Spam"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews/review-1/moderate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "mod-1", "moderator")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ModerateReview_ForbiddenForCustomer(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	body := `{"status": "rejected", "reject_reason": "Spam"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews/review-1/moderate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertNotCalled(t, "ModerateReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// === AddHelpfulVote Handler Tests ===

func TestHandler_AddHelpfulVote_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	mockService.On("AddHelpfulVote", mock.Anything, "tenant-1", "review-1", mock.AnythingOfType("*models.HelpfulVoteRequest")).Return(nil)

	body := `{"user_id": "voter-1", "helpful": true}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews/review-1/helpful", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "voter-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_AddHelpfulVote_BadRequest(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	body := `{"helpful": true}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews/review-1/helpful", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "voter-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === RespondToReview Handler Tests ===

func TestHandler_RespondToReview_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	resp := createTestReviewResponse()
	resp.SellerResponse = "Thanks!"
	mockService.On("RespondToReview", mock.Anything, "tenant-1", "review-1", mock.AnythingOfType("*models.SellerResponseRequest")).Return(resp, nil)

	body := `{"response": "Thanks!"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews/review-1/respond", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "admin-1", "admin")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RespondToReview_ForbiddenForCustomer(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	body := `{"response": "Thanks!"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/reviews/review-1/respond", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authed(req, "tenant-1", "user-1", "customer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockService.AssertNotCalled(t, "RespondToReview", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// === GetProductSummary Handler Tests ===

func TestHandler_GetProductSummary_Success(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	summary := &models.ReviewSummaryResponse{
		ProductID:     "product-1",
		AverageRating: 4.5,
		TotalReviews:  10,
		Distribution:  map[string]int{"1": 0, "2": 1, "3": 1, "4": 3, "5": 5},
	}
	mockService.On("GetProductSummary", mock.Anything, "tenant-1", "product-1").Return(summary, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/product/product-1/summary?tenant_id=tenant-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result models.ReviewSummaryResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, 4.5, result.AverageRating)
	assert.Equal(t, 10, result.TotalReviews)
}

func TestHandler_GetProductSummary_MissingTenantID(t *testing.T) {
	mockService := new(MockReviewService)
	router := setupRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/reviews/product/product-1/summary", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
