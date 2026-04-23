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

type MockMenuService struct {
	mock.Mock
}

func (m *MockMenuService) CreateMenu(ctx context.Context, req *models.CreateMenuRequest) (*models.MenuResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuResponse), args.Error(1)
}

func (m *MockMenuService) GetMenu(ctx context.Context, id string) (*models.MenuResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuResponse), args.Error(1)
}

func (m *MockMenuService) GetMenuBySlug(ctx context.Context, tenantID, slug string) (*models.MenuResponse, error) {
	args := m.Called(ctx, tenantID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuResponse), args.Error(1)
}

func (m *MockMenuService) UpdateMenu(ctx context.Context, id string, req *models.UpdateMenuRequest) (*models.MenuResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuResponse), args.Error(1)
}

func (m *MockMenuService) DeleteMenu(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMenuService) ListMenus(ctx context.Context, tenantID string) ([]models.MenuResponse, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]models.MenuResponse), args.Error(1)
}

func (m *MockMenuService) ListMenusByLocation(ctx context.Context, tenantID, location string) ([]models.MenuResponse, error) {
	args := m.Called(ctx, tenantID, location)
	return args.Get(0).([]models.MenuResponse), args.Error(1)
}

func (m *MockMenuService) CreateMenuItem(ctx context.Context, req *models.CreateMenuItemRequest) (*models.MenuItemResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuItemResponse), args.Error(1)
}

func (m *MockMenuService) UpdateMenuItem(ctx context.Context, id string, req *models.UpdateMenuItemRequest) (*models.MenuItemResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MenuItemResponse), args.Error(1)
}

func (m *MockMenuService) DeleteMenuItem(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMenuService) ReorderItems(ctx context.Context, menuID string, req *models.ReorderItemsRequest) error {
	args := m.Called(ctx, menuID, req)
	return args.Error(0)
}

func setupMenuRouter(mockService *MockMenuService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	handler := NewMenuHandler(mockService, logger)
	router := gin.New()
	RegisterMenuRoutes(router, handler)
	return router
}

// === CreateMenu Handler Tests ===

func TestMenuHandler_CreateMenu_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	resp := &models.MenuResponse{ID: "m-1", Name: "Main Nav", Slug: "main-nav", Location: "header", IsActive: true}
	mockService.On("CreateMenu", mock.Anything, mock.AnythingOfType("*models.CreateMenuRequest")).Return(resp, nil)

	body := `{"tenant_id":"t-1","name":"Main Nav","slug":"main-nav","location":"header"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/menus", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.MenuResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Main Nav", result.Name)
}

func TestMenuHandler_CreateMenu_BadRequest(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	body := `{"name":"test"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/menus", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMenuHandler_CreateMenu_DuplicateSlug(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("CreateMenu", mock.Anything, mock.AnythingOfType("*models.CreateMenuRequest")).
		Return(nil, errors.New("menu with slug 'main-nav' already exists for this tenant"))

	body := `{"tenant_id":"t-1","name":"Main Nav","slug":"main-nav","location":"header"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/menus", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === GetMenu Handler Tests ===

func TestMenuHandler_GetMenu_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	resp := &models.MenuResponse{ID: "m-1", Name: "Main Nav", Items: []models.MenuItemResponse{}}
	mockService.On("GetMenu", mock.Anything, "m-1").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus/m-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_GetMenu_NotFound(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("GetMenu", mock.Anything, "bad").Return(nil, errors.New("menu not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === GetMenuBySlug Handler Tests ===

func TestMenuHandler_GetMenuBySlug_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	resp := &models.MenuResponse{ID: "m-1", Name: "Main Nav", Slug: "main-nav"}
	mockService.On("GetMenuBySlug", mock.Anything, "t-1", "main-nav").Return(resp, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus/slug/main-nav?tenant_id=t-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_GetMenuBySlug_MissingTenant(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus/slug/main-nav", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === UpdateMenu Handler Tests ===

func TestMenuHandler_UpdateMenu_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	resp := &models.MenuResponse{ID: "m-1", Name: "Updated Nav"}
	mockService.On("UpdateMenu", mock.Anything, "m-1", mock.AnythingOfType("*models.UpdateMenuRequest")).Return(resp, nil)

	body := `{"name":"Updated Nav"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/menus/m-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_UpdateMenu_NotFound(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("UpdateMenu", mock.Anything, "bad", mock.AnythingOfType("*models.UpdateMenuRequest")).
		Return(nil, errors.New("menu not found"))

	body := `{"name":"test"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/menus/bad", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === DeleteMenu Handler Tests ===

func TestMenuHandler_DeleteMenu_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("DeleteMenu", mock.Anything, "m-1").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/menus/m-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_DeleteMenu_NotFound(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("DeleteMenu", mock.Anything, "bad").Return(errors.New("menu not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/menus/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ListMenus Handler Tests ===

func TestMenuHandler_ListMenus_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	menus := []models.MenuResponse{
		{ID: "m-1", Name: "Header", Location: "header"},
		{ID: "m-2", Name: "Footer", Location: "footer"},
	}
	mockService.On("ListMenus", mock.Anything, "t-1").Return(menus, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus?tenant_id=t-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, float64(2), result["count"])
}

func TestMenuHandler_ListMenus_MissingTenant(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === ListMenusByLocation Handler Tests ===

func TestMenuHandler_ListMenusByLocation_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	menus := []models.MenuResponse{
		{ID: "m-1", Name: "Main Nav", Location: "header"},
	}
	mockService.On("ListMenusByLocation", mock.Anything, "t-1", "header").Return(menus, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus/location/header?tenant_id=t-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_ListMenusByLocation_MissingTenant(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/menus/location/header", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === CreateMenuItem Handler Tests ===

func TestMenuHandler_CreateMenuItem_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	resp := &models.MenuItemResponse{ID: "i-1", MenuID: "m-1", Label: "Home", URL: "/", Target: "_self"}
	mockService.On("CreateMenuItem", mock.Anything, mock.AnythingOfType("*models.CreateMenuItemRequest")).Return(resp, nil)

	body := `{"label":"Home","url":"/"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/menu-items/m-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result models.MenuItemResponse
	json.Unmarshal(w.Body.Bytes(), &result)
	assert.Equal(t, "Home", result.Label)
}

func TestMenuHandler_CreateMenuItem_BadRequest(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	body := `{"url":"/"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/menu-items/m-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === UpdateMenuItem Handler Tests ===

func TestMenuHandler_UpdateMenuItem_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	resp := &models.MenuItemResponse{ID: "i-1", Label: "Updated", URL: "/new"}
	mockService.On("UpdateMenuItem", mock.Anything, "i-1", mock.AnythingOfType("*models.UpdateMenuItemRequest")).Return(resp, nil)

	body := `{"label":"Updated","url":"/new"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/menu-items/i-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_UpdateMenuItem_NotFound(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("UpdateMenuItem", mock.Anything, "bad", mock.AnythingOfType("*models.UpdateMenuItemRequest")).
		Return(nil, errors.New("menu item not found"))

	body := `{"label":"test"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/menu-items/bad", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === DeleteMenuItem Handler Tests ===

func TestMenuHandler_DeleteMenuItem_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("DeleteMenuItem", mock.Anything, "i-1").Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/menu-items/i-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_DeleteMenuItem_NotFound(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("DeleteMenuItem", mock.Anything, "bad").Return(errors.New("menu item not found"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/menu-items/bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// === ReorderItems Handler Tests ===

func TestMenuHandler_ReorderItems_Success(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("ReorderItems", mock.Anything, "m-1", mock.AnythingOfType("*models.ReorderItemsRequest")).Return(nil)

	body := `{"items":[{"id":"i-1","position":1},{"id":"i-2","position":0}]}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/menus-reorder/m-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMenuHandler_ReorderItems_MenuNotFound(t *testing.T) {
	mockService := new(MockMenuService)
	router := setupMenuRouter(mockService)

	mockService.On("ReorderItems", mock.Anything, "bad", mock.AnythingOfType("*models.ReorderItemsRequest")).
		Return(errors.New("menu not found"))

	body := `{"items":[{"id":"i-1","position":0}]}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/menus-reorder/bad", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
