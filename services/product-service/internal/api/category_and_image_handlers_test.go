package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/ecommerce/product-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCategoryService implements service.CategoryService for testing.
type MockCategoryService struct {
	mock.Mock
}

func (m *MockCategoryService) CreateCategory(ctx context.Context, req *models.CreateCategoryRequest) (*models.CategoryResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CategoryResponse), args.Error(1)
}

func (m *MockCategoryService) GetCategoryByID(ctx context.Context, tenantID, id string) (*models.CategoryResponse, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CategoryResponse), args.Error(1)
}

func (m *MockCategoryService) GetCategoryBySlug(ctx context.Context, tenantID, slug string) (*models.CategoryResponse, error) {
	args := m.Called(ctx, tenantID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CategoryResponse), args.Error(1)
}

func (m *MockCategoryService) ListCategories(ctx context.Context, tenantID string, offset, limit int) ([]models.CategoryResponse, int64, error) {
	args := m.Called(ctx, tenantID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.CategoryResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryService) ListCategoriesByParent(ctx context.Context, tenantID string, parentID *string, offset, limit int) ([]models.CategoryResponse, int64, error) {
	args := m.Called(ctx, tenantID, parentID, offset, limit)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]models.CategoryResponse), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryService) UpdateCategory(ctx context.Context, tenantID, id string, req *models.UpdateCategoryRequest) (*models.CategoryResponse, error) {
	args := m.Called(ctx, tenantID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CategoryResponse), args.Error(1)
}

func (m *MockCategoryService) DeleteCategory(ctx context.Context, tenantID, id string) error {
	return m.Called(ctx, tenantID, id).Error(0)
}

func (m *MockCategoryService) UpdateCategoryStatus(ctx context.Context, tenantID, id string, status models.CategoryStatus) error {
	return m.Called(ctx, tenantID, id, status).Error(0)
}

// MockImageService implements service.ImageService for testing.
type MockImageService struct {
	mock.Mock
}

func (m *MockImageService) PresignUpload(ctx context.Context, tenantID, productID, contentType, filename string) (*service.PresignUploadResult, error) {
	args := m.Called(ctx, tenantID, productID, contentType, filename)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PresignUploadResult), args.Error(1)
}

func (m *MockImageService) ConfirmUpload(ctx context.Context, tenantID, productID, imageURL string) error {
	return m.Called(ctx, tenantID, productID, imageURL).Error(0)
}

func (m *MockImageService) RemoveImage(ctx context.Context, tenantID, productID, imageURL string) error {
	return m.Called(ctx, tenantID, productID, imageURL).Error(0)
}

// MockCategoryImageService implements service.CategoryImageService for testing.
type MockCategoryImageService struct {
	mock.Mock
}

func (m *MockCategoryImageService) PresignUpload(ctx context.Context, tenantID, categoryID, contentType, filename string) (*service.PresignUploadResult, error) {
	args := m.Called(ctx, tenantID, categoryID, contentType, filename)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.PresignUploadResult), args.Error(1)
}

func (m *MockCategoryImageService) ConfirmUpload(ctx context.Context, tenantID, categoryID, imageURL string) error {
	return m.Called(ctx, tenantID, categoryID, imageURL).Error(0)
}

func (m *MockCategoryImageService) RemoveImage(ctx context.Context, tenantID, categoryID string) error {
	return m.Called(ctx, tenantID, categoryID).Error(0)
}

func quietLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return logger
}

// setupCategoryRouter mirrors setupProductRouter: the shared Auth middleware is
// stubbed by a header so handler bodies are exercised directly.
func setupCategoryRouter(mockService *MockCategoryService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewCategoryHandler(mockService, quietLogger())
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

func testCategoryResponse() *models.CategoryResponse {
	return &models.CategoryResponse{
		ID:        "category-1",
		TenantID:  "tenant-1",
		Name:      "Shirts",
		Slug:      "shirts",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func validCategoryBody() map[string]interface{} {
	return map[string]interface{}{
		"name":       "Shirts",
		"slug":       "shirts",
		"created_by": "user-1",
	}
}

// -------------------------------------------------------------- CreateCategory

func TestCreateCategory_Success(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("CreateCategory", mock.Anything, mock.AnythingOfType("*models.CreateCategoryRequest")).
		Return(testCategoryResponse(), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPost, "/api/v1/categories", "tenant-1", validCategoryBody())

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "category-1", decodeBody(t, w)["id"])
	mockService.AssertExpectations(t)
}

// The tenant on the request the service receives must come from the JWT. The
// body field is json:"-", so a spoofed tenant_id cannot even be bound.
func TestCreateCategory_TenantComesFromJWTNotBody(t *testing.T) {
	mockService := new(MockCategoryService)
	var captured *models.CreateCategoryRequest
	mockService.On("CreateCategory", mock.Anything, mock.AnythingOfType("*models.CreateCategoryRequest")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*models.CreateCategoryRequest)
		}).
		Return(testCategoryResponse(), nil)
	router := setupCategoryRouter(mockService)

	body := validCategoryBody()
	body["tenant_id"] = "tenant-attacker"

	w := doRequest(router, http.MethodPost, "/api/v1/categories", "tenant-1", body)

	assert.Equal(t, http.StatusCreated, w.Code)
	if assert.NotNil(t, captured) {
		assert.Equal(t, "tenant-1", captured.TenantID, "tenant must be the JWT's, not the body's")
	}
}

func TestCreateCategory_Unauthenticated(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPost, "/api/v1/categories", "", validCategoryBody())

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "CreateCategory", mock.Anything, mock.Anything)
}

func TestCreateCategory_InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{name: "missing name", body: map[string]interface{}{"slug": "shirts", "created_by": "user-1"}},
		{name: "missing slug", body: map[string]interface{}{"name": "Shirts", "created_by": "user-1"}},
		{name: "missing created_by", body: map[string]interface{}{"name": "Shirts", "slug": "shirts"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCategoryService)
			router := setupCategoryRouter(mockService)

			w := doRequest(router, http.MethodPost, "/api/v1/categories", "tenant-1", tt.body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			mockService.AssertNotCalled(t, "CreateCategory", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateCategory_ServiceFailure(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("CreateCategory", mock.Anything, mock.Anything).Return(nil, errors.New("slug already taken"))
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPost, "/api/v1/categories", "tenant-1", validCategoryBody())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ----------------------------------------------------------------- GetCategory

func TestGetCategory_Success(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("GetCategoryByID", mock.Anything, "tenant-1", "category-1").Return(testCategoryResponse(), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/category-1", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "category-1", decodeBody(t, w)["id"])
	mockService.AssertExpectations(t)
}

func TestGetCategory_NotFound(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("GetCategoryByID", mock.Anything, "tenant-1", "ghost").Return(nil, errors.New("category not found"))
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/ghost", "tenant-1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCategory_NoTenant(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/category-1", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "GetCategoryByID", mock.Anything, mock.Anything, mock.Anything)
}

// ----------------------------------------------------------- GetCategoryBySlug

func TestGetCategoryBySlug_Success(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("GetCategoryBySlug", mock.Anything, "tenant-1", "shirts").Return(testCategoryResponse(), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/slug/shirts", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "shirts", decodeBody(t, w)["slug"])
	mockService.AssertExpectations(t)
}

func TestGetCategoryBySlug_NotFound(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("GetCategoryBySlug", mock.Anything, "tenant-1", "ghost").Return(nil, errors.New("not found"))
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/slug/ghost", "tenant-1", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCategoryBySlug_NoTenant(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/slug/shirts", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --------------------------------------------------------------- ListCategories

func TestListCategories_Success(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("ListCategories", mock.Anything, "tenant-1", 0, 20).
		Return([]models.CategoryResponse{*testCategoryResponse()}, int64(1), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, float64(1), body["total"])
	assert.Equal(t, float64(20), body["limit"])
	assert.Equal(t, float64(0), body["offset"])
	mockService.AssertExpectations(t)
}

// The page size is clamped so a caller cannot ask for an unbounded scan, and a
// nonsensical value falls back to the default rather than erroring.
func TestListCategories_LimitClamping(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantLimit int
	}{
		{name: "over the cap is clamped", query: "?limit=5000", wantLimit: 100},
		{name: "zero falls back to default", query: "?limit=0", wantLimit: 20},
		{name: "negative falls back to default", query: "?limit=-5", wantLimit: 20},
		{name: "unparsable falls back to default", query: "?limit=many", wantLimit: 20},
		{name: "at the cap is kept", query: "?limit=100", wantLimit: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCategoryService)
			mockService.On("ListCategories", mock.Anything, "tenant-1", 0, tt.wantLimit).
				Return([]models.CategoryResponse{}, int64(0), nil)
			router := setupCategoryRouter(mockService)

			w := doRequest(router, http.MethodGet, "/api/v1/categories"+tt.query, "tenant-1", nil)

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestListCategories_NoTenant(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListCategories_ServiceFailure(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("ListCategories", mock.Anything, "tenant-1", 0, 20).Return(nil, int64(0), errors.New("db down"))
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories", "tenant-1", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// -------------------------------------------------------- ListCategoriesByParent

func TestListCategoriesByParent_RootCategories(t *testing.T) {
	mockService := new(MockCategoryService)
	// No parent_id means root level, which the service expects as a nil pointer.
	mockService.On("ListCategoriesByParent", mock.Anything, "tenant-1", (*string)(nil), 0, 20).
		Return([]models.CategoryResponse{*testCategoryResponse()}, int64(1), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/by-parent", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestListCategoriesByParent_WithParent(t *testing.T) {
	mockService := new(MockCategoryService)
	var captured *string
	mockService.On("ListCategoriesByParent", mock.Anything, "tenant-1", mock.Anything, 0, 20).
		Run(func(args mock.Arguments) {
			if p, ok := args.Get(2).(*string); ok {
				captured = p
			}
		}).
		Return([]models.CategoryResponse{}, int64(0), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/by-parent?parent_id=category-parent", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, captured) {
		assert.Equal(t, "category-parent", *captured)
	}
}

func TestListCategoriesByParent_NoTenant(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/by-parent", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListCategoriesByParent_ServiceFailure(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("ListCategoriesByParent", mock.Anything, "tenant-1", mock.Anything, 0, 20).
		Return(nil, int64(0), errors.New("db down"))
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/by-parent", "tenant-1", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListCategoriesByParent_LimitClamping(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("ListCategoriesByParent", mock.Anything, "tenant-1", (*string)(nil), 0, 100).
		Return([]models.CategoryResponse{}, int64(0), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodGet, "/api/v1/categories/by-parent?limit=5000", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

// -------------------------------------------------------------- UpdateCategory

func TestUpdateCategory_Success(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("UpdateCategory", mock.Anything, "tenant-1", "category-1", mock.AnythingOfType("*models.UpdateCategoryRequest")).
		Return(testCategoryResponse(), nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPut, "/api/v1/categories/category-1", "tenant-1",
		map[string]interface{}{"name": "Tops", "updated_by": "user-1"})

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestUpdateCategory_Unauthenticated(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPut, "/api/v1/categories/category-1", "",
		map[string]interface{}{"updated_by": "user-1"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "UpdateCategory", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateCategory_InvalidBody(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	// updated_by is required.
	w := doRequest(router, http.MethodPut, "/api/v1/categories/category-1", "tenant-1",
		map[string]interface{}{"name": "Tops"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCategory_NotFoundVersusFailure(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "missing category maps to 404", err: errors.New("category not found"), wantCode: http.StatusNotFound},
		{name: "anything else maps to 500", err: errors.New("db down"), wantCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCategoryService)
			mockService.On("UpdateCategory", mock.Anything, "tenant-1", "category-1", mock.Anything).Return(nil, tt.err)
			router := setupCategoryRouter(mockService)

			w := doRequest(router, http.MethodPut, "/api/v1/categories/category-1", "tenant-1",
				map[string]interface{}{"updated_by": "user-1"})

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

// -------------------------------------------------------------- DeleteCategory

func TestDeleteCategory_Success(t *testing.T) {
	mockService := new(MockCategoryService)
	mockService.On("DeleteCategory", mock.Anything, "tenant-1", "category-1").Return(nil)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/api/v1/categories/category-1", "tenant-1", nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestDeleteCategory_Unauthenticated(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/api/v1/categories/category-1", "", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertNotCalled(t, "DeleteCategory", mock.Anything, mock.Anything, mock.Anything)
}

func TestDeleteCategory_NotFoundVersusFailure(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "delete failure maps to 404", err: errors.New("failed to delete category"), wantCode: http.StatusNotFound},
		{name: "anything else maps to 500", err: errors.New("db down"), wantCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCategoryService)
			mockService.On("DeleteCategory", mock.Anything, "tenant-1", "category-1").Return(tt.err)
			router := setupCategoryRouter(mockService)

			w := doRequest(router, http.MethodDelete, "/api/v1/categories/category-1", "tenant-1", nil)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

// -------------------------------------------------------- UpdateCategoryStatus

func TestUpdateCategoryStatus_Success(t *testing.T) {
	for _, status := range []models.CategoryStatus{models.CategoryStatusActive, models.CategoryStatusInactive} {
		t.Run(string(status), func(t *testing.T) {
			mockService := new(MockCategoryService)
			mockService.On("UpdateCategoryStatus", mock.Anything, "tenant-1", "category-1", status).Return(nil)
			router := setupCategoryRouter(mockService)

			w := doRequest(router, http.MethodPatch, "/api/v1/categories/category-1/status", "tenant-1",
				map[string]interface{}{"status": string(status)})

			assert.Equal(t, http.StatusOK, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// Only the two known statuses are accepted; anything else is rejected before
// it can reach the service and land in the database.
func TestUpdateCategoryStatus_RejectsUnknownStatus(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPatch, "/api/v1/categories/category-1/status", "tenant-1",
		map[string]interface{}{"status": "deleted"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid status value", decodeBody(t, w)["error"])
	mockService.AssertNotCalled(t, "UpdateCategoryStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateCategoryStatus_MissingStatus(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPatch, "/api/v1/categories/category-1/status", "tenant-1",
		map[string]interface{}{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateCategoryStatus_Unauthenticated(t *testing.T) {
	mockService := new(MockCategoryService)
	router := setupCategoryRouter(mockService)

	w := doRequest(router, http.MethodPatch, "/api/v1/categories/category-1/status", "",
		map[string]interface{}{"status": "active"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateCategoryStatus_NotFoundVersusFailure(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "update failure maps to 404", err: errors.New("failed to update category status"), wantCode: http.StatusNotFound},
		{name: "anything else maps to 500", err: errors.New("db down"), wantCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockCategoryService)
			mockService.On("UpdateCategoryStatus", mock.Anything, "tenant-1", "category-1", models.CategoryStatusActive).Return(tt.err)
			router := setupCategoryRouter(mockService)

			w := doRequest(router, http.MethodPatch, "/api/v1/categories/category-1/status", "tenant-1",
				map[string]interface{}{"status": "active"})

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

// Write routes must sit behind the supplied auth middleware; read routes must
// stay public so the storefront can browse without a token.
func TestCategoryRegisterRoutes_AuthGate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockCategoryService)
	mockService.On("ListCategories", mock.Anything, "tenant-1", 0, 20).
		Return([]models.CategoryResponse{}, int64(0), nil)

	handler := NewCategoryHandler(mockService, quietLogger())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		c.Next()
	})
	// An auth middleware that always rejects.
	handler.RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	})

	readResp := doRequest(router, http.MethodGet, "/api/v1/categories", "tenant-1", nil)
	assert.Equal(t, http.StatusOK, readResp.Code, "read routes should stay public")

	writeResp := doRequest(router, http.MethodPost, "/api/v1/categories", "tenant-1", validCategoryBody())
	assert.Equal(t, http.StatusUnauthorized, writeResp.Code, "write routes must be behind auth")
	mockService.AssertNotCalled(t, "CreateCategory", mock.Anything, mock.Anything)
}

// ----------------------------------------------------------- product images

func setupImageRouter(mockService *MockImageService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewImageHandler(mockService, quietLogger())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		c.Next()
	})
	handler.RegisterRoutes(router.Group("/products"))
	return router
}

func testPresignResult() *service.PresignUploadResult {
	return &service.PresignUploadResult{
		UploadURL:    "https://storage.test/put",
		UploadMethod: http.MethodPut,
		Headers:      map[string]string{"Content-Type": "image/png"},
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		ObjectKey:    "products/product-1/hero.png",
		ImageURL:     "https://cdn.test/products/product-1/hero.png",
	}
}

func TestPresignImageUpload_Success(t *testing.T) {
	mockService := new(MockImageService)
	mockService.On("PresignUpload", mock.Anything, "tenant-1", "product-1", "image/png", "hero.png").
		Return(testPresignResult(), nil)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images/presign", "tenant-1",
		map[string]interface{}{"content_type": "image/png", "filename": "hero.png"})

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "https://storage.test/put", body["upload_url"])
	assert.Equal(t, "https://cdn.test/products/product-1/hero.png", body["image_url"])
	mockService.AssertExpectations(t)
}

func TestPresignImageUpload_NoTenant(t *testing.T) {
	mockService := new(MockImageService)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images/presign", "",
		map[string]interface{}{"content_type": "image/png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "PresignUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPresignImageUpload_MissingContentType(t *testing.T) {
	mockService := new(MockImageService)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images/presign", "tenant-1",
		map[string]interface{}{"filename": "hero.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "PresignUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// A rejected content type is the service's call (it owns the allowlist); the
// handler must surface it as a 400 rather than a 500.
func TestPresignImageUpload_ServiceRejection(t *testing.T) {
	mockService := new(MockImageService)
	mockService.On("PresignUpload", mock.Anything, "tenant-1", "product-1", "image/svg+xml", "").
		Return(nil, errors.New("unsupported content type"))
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images/presign", "tenant-1",
		map[string]interface{}{"content_type": "image/svg+xml"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "unsupported content type", decodeBody(t, w)["error"])
}

func TestConfirmImageUpload_Success(t *testing.T) {
	mockService := new(MockImageService)
	mockService.On("ConfirmUpload", mock.Anything, "tenant-1", "product-1", "https://cdn.test/a.png").Return(nil)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images", "tenant-1",
		map[string]interface{}{"image_url": "https://cdn.test/a.png"})

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, true, body["success"])
	assert.Equal(t, "https://cdn.test/a.png", body["image_url"])
	mockService.AssertExpectations(t)
}

func TestConfirmImageUpload_NoTenant(t *testing.T) {
	mockService := new(MockImageService)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images", "",
		map[string]interface{}{"image_url": "https://cdn.test/a.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestConfirmImageUpload_MissingImageURL(t *testing.T) {
	mockService := new(MockImageService)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images", "tenant-1", map[string]interface{}{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "ConfirmUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Confirming an object that was never uploaded must fail, otherwise a broken
// image URL gets persisted on the product.
func TestConfirmImageUpload_ObjectMissingFromStorage(t *testing.T) {
	mockService := new(MockImageService)
	mockService.On("ConfirmUpload", mock.Anything, "tenant-1", "product-1", "https://cdn.test/a.png").
		Return(errors.New("object not found in storage"))
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/products/product-1/images", "tenant-1",
		map[string]interface{}{"image_url": "https://cdn.test/a.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveImage_Success(t *testing.T) {
	mockService := new(MockImageService)
	mockService.On("RemoveImage", mock.Anything, "tenant-1", "product-1", "https://cdn.test/a.png").Return(nil)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/products/product-1/images", "tenant-1",
		map[string]interface{}{"image_url": "https://cdn.test/a.png"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, decodeBody(t, w)["success"])
	mockService.AssertExpectations(t)
}

func TestRemoveImage_NoTenant(t *testing.T) {
	mockService := new(MockImageService)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/products/product-1/images", "",
		map[string]interface{}{"image_url": "https://cdn.test/a.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveImage_MissingImageURL(t *testing.T) {
	mockService := new(MockImageService)
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/products/product-1/images", "tenant-1", map[string]interface{}{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveImage_ServiceFailure(t *testing.T) {
	mockService := new(MockImageService)
	mockService.On("RemoveImage", mock.Anything, "tenant-1", "product-1", "https://cdn.test/a.png").
		Return(errors.New("storage unavailable"))
	router := setupImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/products/product-1/images", "tenant-1",
		map[string]interface{}{"image_url": "https://cdn.test/a.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------- category images

func setupCategoryImageRouter(mockService *MockCategoryImageService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewCategoryImageHandler(mockService, quietLogger())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if t := c.GetHeader("X-Test-Tenant"); t != "" {
			c.Set("tenant_id", t)
		}
		c.Next()
	})
	handler.RegisterRoutes(router.Group("/categories"))
	return router
}

func TestCategoryPresignImageUpload_Success(t *testing.T) {
	mockService := new(MockCategoryImageService)
	mockService.On("PresignUpload", mock.Anything, "tenant-1", "category-1", "image/webp", "hero.webp").
		Return(testPresignResult(), nil)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image/presign", "tenant-1",
		map[string]interface{}{"content_type": "image/webp", "filename": "hero.webp"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://storage.test/put", decodeBody(t, w)["upload_url"])
	mockService.AssertExpectations(t)
}

func TestCategoryPresignImageUpload_NoTenant(t *testing.T) {
	mockService := new(MockCategoryImageService)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image/presign", "",
		map[string]interface{}{"content_type": "image/webp"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "PresignUpload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCategoryPresignImageUpload_MissingContentType(t *testing.T) {
	mockService := new(MockCategoryImageService)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image/presign", "tenant-1",
		map[string]interface{}{"filename": "hero.webp"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryPresignImageUpload_ServiceRejection(t *testing.T) {
	mockService := new(MockCategoryImageService)
	mockService.On("PresignUpload", mock.Anything, "tenant-1", "category-1", "image/svg+xml", "").
		Return(nil, errors.New("unsupported content type"))
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image/presign", "tenant-1",
		map[string]interface{}{"content_type": "image/svg+xml"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryConfirmImageUpload_Success(t *testing.T) {
	mockService := new(MockCategoryImageService)
	mockService.On("ConfirmUpload", mock.Anything, "tenant-1", "category-1", "https://cdn.test/c.png").Return(nil)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image", "tenant-1",
		map[string]interface{}{"image_url": "https://cdn.test/c.png"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://cdn.test/c.png", decodeBody(t, w)["image_url"])
	mockService.AssertExpectations(t)
}

func TestCategoryConfirmImageUpload_NoTenant(t *testing.T) {
	mockService := new(MockCategoryImageService)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image", "",
		map[string]interface{}{"image_url": "https://cdn.test/c.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryConfirmImageUpload_MissingImageURL(t *testing.T) {
	mockService := new(MockCategoryImageService)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image", "tenant-1", map[string]interface{}{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoryConfirmImageUpload_ServiceFailure(t *testing.T) {
	mockService := new(MockCategoryImageService)
	mockService.On("ConfirmUpload", mock.Anything, "tenant-1", "category-1", "https://cdn.test/c.png").
		Return(errors.New("object not found in storage"))
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodPost, "/categories/category-1/image", "tenant-1",
		map[string]interface{}{"image_url": "https://cdn.test/c.png"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Unlike the product variant, removing a category image takes no body — the
// category has a single hero image.
func TestCategoryRemoveImage_Success(t *testing.T) {
	mockService := new(MockCategoryImageService)
	mockService.On("RemoveImage", mock.Anything, "tenant-1", "category-1").Return(nil)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/categories/category-1/image", "tenant-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, decodeBody(t, w)["success"])
	mockService.AssertExpectations(t)
}

func TestCategoryRemoveImage_NoTenant(t *testing.T) {
	mockService := new(MockCategoryImageService)
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/categories/category-1/image", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockService.AssertNotCalled(t, "RemoveImage", mock.Anything, mock.Anything, mock.Anything)
}

func TestCategoryRemoveImage_ServiceFailure(t *testing.T) {
	mockService := new(MockCategoryImageService)
	mockService.On("RemoveImage", mock.Anything, "tenant-1", "category-1").Return(errors.New("storage unavailable"))
	router := setupCategoryImageRouter(mockService)

	w := doRequest(router, http.MethodDelete, "/categories/category-1/image", "tenant-1", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
