package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTemplateService is the test double used by template_handler_test.go.
// We only stub the methods the handler tests actually invoke; the rest of the
// service.TemplateService surface returns the zero value of its return type
// (this is fine because gin will short-circuit unused routes).
type MockTemplateService struct {
	mock.Mock
}

func (m *MockTemplateService) List(ctx context.Context, tenantID string) ([]models.NotificationTemplate, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NotificationTemplate), args.Error(1)
}

func (m *MockTemplateService) Get(ctx context.Context, tenantID, id string) (*models.NotificationTemplate, error) {
	args := m.Called(ctx, tenantID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationTemplate), args.Error(1)
}

func (m *MockTemplateService) Create(ctx context.Context, tenantID string, req *models.CreateTemplateRequest) (*models.NotificationTemplate, error) {
	args := m.Called(ctx, tenantID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationTemplate), args.Error(1)
}

func (m *MockTemplateService) Update(ctx context.Context, tenantID, id string, req *models.UpdateTemplateRequest) (*models.NotificationTemplate, error) {
	args := m.Called(ctx, tenantID, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NotificationTemplate), args.Error(1)
}

func (m *MockTemplateService) Delete(ctx context.Context, tenantID, id string) error {
	return m.Called(ctx, tenantID, id).Error(0)
}

func (m *MockTemplateService) Preview(ctx context.Context, tenantID, id string, sampleVars map[string]interface{}) (*models.RenderedTemplate, error) {
	args := m.Called(ctx, tenantID, id, sampleVars)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RenderedTemplate), args.Error(1)
}

func (m *MockTemplateService) TestSend(ctx context.Context, tenantID, id, email string, sampleVars map[string]interface{}) error {
	return m.Called(ctx, tenantID, id, email, sampleVars).Error(0)
}

func (m *MockTemplateService) InstallDefaults(ctx context.Context, tenantID string, force bool) (*service.InstallDefaultsResult, error) {
	args := m.Called(ctx, tenantID, force)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.InstallDefaultsResult), args.Error(1)
}

func setupTemplateRouter(svc *MockTemplateService) *gin.Engine {
	return setupTemplateRouterWithTenant(svc, "tenant-1")
}

// setupTemplateRouterWithTenant builds the template router with a middleware
// that injects the given tenant into the gin context, standing in for the JWT
// auth middleware. Pass an empty tenantID to simulate an unauthenticated
// request (no tenant claim).
func setupTemplateRouterWithTenant(svc *MockTemplateService, tenantID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	h := NewTemplateHandler(svc, logger)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	})
	RegisterTemplateRoutes(router, h)
	return router
}

// TestInstallDefaults_Success exercises the happy path: a tenant with no
// existing templates installs the full starter pack. We assert the response
// shape and counters that the admin UI relies on for its toast.
func TestInstallDefaults_Success(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	expected := &service.InstallDefaultsResult{
		Created:   11,
		Updated:   0,
		Skipped:   0,
		Templates: []models.NotificationTemplate{{ID: "t-1", TenantID: "tenant-1"}},
	}
	mockSvc.On("InstallDefaults", mock.Anything, "tenant-1", false).Return(expected, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notification-templates/install-defaults", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", "tenant-1")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var got service.InstallDefaultsResult
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, 11, got.Created)
	assert.Equal(t, 0, got.Skipped)
}

func TestInstallDefaults_ForceFlag(t *testing.T) {
	mockSvc := new(MockTemplateService)
	router := setupTemplateRouter(mockSvc)

	mockSvc.On("InstallDefaults", mock.Anything, "tenant-1", true).
		Return(&service.InstallDefaultsResult{Created: 5, Updated: 6}, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notification-templates/install-defaults", bytes.NewBufferString(`{"force":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", "tenant-1")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockSvc.AssertExpectations(t)
}

func TestInstallDefaults_Unauthenticated(t *testing.T) {
	mockSvc := new(MockTemplateService)
	// No tenant claim in context → request is unauthenticated. Note the
	// X-Tenant-Id header is deliberately not honored anymore.
	router := setupTemplateRouterWithTenant(mockSvc, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/notification-templates/install-defaults", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", "tenant-1")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
