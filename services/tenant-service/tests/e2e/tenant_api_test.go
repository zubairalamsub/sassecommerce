package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/tenant-service/internal/api"
	"github.com/ecommerce/tenant-service/internal/middleware"
	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/ecommerce/tenant-service/pkg/kafka/mocks"
	"github.com/ecommerce/tenant-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type E2ETestSuite struct {
	suite.Suite
	router       *gin.Engine
	db           *gorm.DB
	logger       *logrus.Logger
	createdTenantID string
}

func (suite *E2ETestSuite) SetupSuite() {
	// Setup logger
	suite.logger = logger.NewLogger("test")
	suite.logger.SetLevel(logrus.ErrorLevel)

	// Setup in-memory database with shared cache so all connections see the same data
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Limit to single connection for SQLite
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	// Run migrations
	err = db.AutoMigrate(&models.Tenant{}, &models.AuditLog{})
	assert.NoError(suite.T(), err)

	suite.db = db

	// Setup repositories
	tenantRepo := repository.NewTenantRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Setup services with mock Kafka
	mockKafka := new(mocks.MockKafkaProducer)
	mockKafka.On("Publish", mock.Anything, "tenant-events", mock.Anything, mock.Anything).Return(nil)

	tenantService := service.NewTenantService(tenantRepo, mockKafka, suite.logger)
	auditService := service.NewAuditService(auditRepo, suite.logger)

	// Setup handlers
	tenantHandler := api.NewTenantHandler(tenantService, suite.logger)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuditMiddleware(auditService, suite.logger))

	// Register routes
	v1 := router.Group("/api/v1")
	{
		tenants := v1.Group("/tenants")
		{
			tenants.POST("", tenantHandler.CreateTenant)
			tenants.GET("", tenantHandler.ListTenants)
			tenants.GET("/:id", tenantHandler.GetTenant)
			tenants.GET("/slug/:slug", tenantHandler.GetTenantBySlug)
			tenants.GET("/domain", tenantHandler.GetTenantByDomain)
			tenants.PUT("/:id", tenantHandler.UpdateTenant)
			tenants.PATCH("/:id/config", tenantHandler.UpdateTenantConfig)
			tenants.DELETE("/:id", tenantHandler.DeleteTenant)
		}
	}

	suite.router = router
}

func (suite *E2ETestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *E2ETestSuite) TestCreateTenant_Success() {
	reqBody := models.CreateTenantRequest{
		Name:  "E2E Test Store",
		Email: "e2e@example.com",
		Tier:  "free",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.TenantResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "E2E Test Store", response.Name)
	assert.Equal(suite.T(), "e2e@example.com", response.Email)
	assert.Equal(suite.T(), models.TierFree, response.Tier)
	assert.Equal(suite.T(), models.StatusPending, response.Status)
	assert.NotEmpty(suite.T(), response.ID)

	// Save tenant ID for other tests
	suite.createdTenantID = response.ID
}

func (suite *E2ETestSuite) TestCreateTenant_InvalidRequest() {
	reqBody := map[string]interface{}{
		"name": "Test",
		// Missing required fields
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

func (suite *E2ETestSuite) TestGetTenant_Success() {
	// First create a tenant
	tenant := &models.Tenant{
		Name:   "Test Get Store",
		Slug:   "test-get-store-123",
		Email:  "testget@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.db.Create(tenant).Error
	assert.NoError(suite.T(), err)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenants/%s", tenant.ID), nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.TenantResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), tenant.ID, response.ID)
	assert.Equal(suite.T(), tenant.Name, response.Name)
}

func (suite *E2ETestSuite) TestGetTenant_NotFound() {
	req, _ := http.NewRequest("GET", "/api/v1/tenants/nonexistent-id", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *E2ETestSuite) TestGetTenantBySlug_Success() {
	// Create a tenant
	tenant := &models.Tenant{
		Name:   "Test Slug Store",
		Slug:   "test-slug-unique-123",
		Email:  "testslug@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.db.Create(tenant).Error
	assert.NoError(suite.T(), err)

	req, _ := http.NewRequest("GET", "/api/v1/tenants/slug/test-slug-unique-123", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.TenantResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-slug-unique-123", response.Slug)
}

func (suite *E2ETestSuite) TestGetTenantByDomain_Success() {
	// Create a tenant with custom domain
	tenant := &models.Tenant{
		Name:   "Test Domain Store",
		Slug:   "test-domain-store-123",
		Email:  "testdomain@example.com",
		Domain: "testdomain.mystore.com",
		Status: models.StatusActive,
		Tier:   models.TierProfessional,
	}
	err := suite.db.Create(tenant).Error
	assert.NoError(suite.T(), err)

	req, _ := http.NewRequest("GET", "/api/v1/tenants/domain?domain=testdomain.mystore.com", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.TenantResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testdomain.mystore.com", response.Domain)
}

func (suite *E2ETestSuite) TestListTenants_Success() {
	// Create multiple tenants
	for i := 0; i < 5; i++ {
		tenant := &models.Tenant{
			Name:   fmt.Sprintf("List Test Store %d", i),
			Slug:   fmt.Sprintf("list-test-store-%d", i),
			Email:  fmt.Sprintf("listtest%d@example.com", i),
			Status: models.StatusActive,
			Tier:   models.TierFree,
		}
		err := suite.db.Create(tenant).Error
		assert.NoError(suite.T(), err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/tenants?page=1&page_size=10", nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response struct {
		Data       []models.TenantResponse `json:"data"`
		Pagination struct {
			Page       int   `json:"page"`
			PageSize   int   `json:"page_size"`
			Total      int64 `json:"total"`
			TotalPages int64 `json:"total_pages"`
		} `json:"pagination"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.GreaterOrEqual(suite.T(), len(response.Data), 5)
	assert.GreaterOrEqual(suite.T(), response.Pagination.Total, int64(5))
}

func (suite *E2ETestSuite) TestUpdateTenant_Success() {
	// Create a tenant
	tenant := &models.Tenant{
		Name:   "Update Test Store",
		Slug:   "update-test-store-123",
		Email:  "updatetest@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.db.Create(tenant).Error
	assert.NoError(suite.T(), err)

	newName := "Updated Store Name"
	updateReq := models.UpdateTenantRequest{
		Name: &newName,
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenants/%s", tenant.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.TenantResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Store Name", response.Name)
}

func (suite *E2ETestSuite) TestUpdateTenantConfig_Success() {
	// Create a tenant
	tenant := &models.Tenant{
		Name:   "Config Test Store",
		Slug:   "config-test-store-123",
		Email:  "configtest@example.com",
		Status: models.StatusActive,
		Tier:   models.TierProfessional,
	}
	err := suite.db.Create(tenant).Error
	assert.NoError(suite.T(), err)

	configReq := models.TenantConfig{
		General: models.GeneralConfig{
			Timezone: "Asia/Dhaka",
			Currency: "BDT",
			Language: "bn",
		},
		Branding: models.BrandingConfig{
			PrimaryColor:   "#FF5733",
			SecondaryColor: "#33FF57",
		},
		Features: models.FeatureConfig{
			MultiCurrency:     true,
			AIRecommendations: true,
		},
	}

	body, _ := json.Marshal(configReq)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/v1/tenants/%s/config", tenant.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func (suite *E2ETestSuite) TestDeleteTenant_Success() {
	// Create a tenant
	tenant := &models.Tenant{
		Name:   "Delete Test Store",
		Slug:   "delete-test-store-123",
		Email:  "deletetest@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.db.Create(tenant).Error
	assert.NoError(suite.T(), err)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/tenants/%s", tenant.ID), nil)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// Verify tenant is deleted
	var deletedTenant models.Tenant
	err = suite.db.Unscoped().First(&deletedTenant, "id = ?", tenant.ID).Error
	assert.Error(suite.T(), err) // Record should not be found after deletion
}

func (suite *E2ETestSuite) TestListTenantsWithPagination() {
	// Create 25 tenants
	for i := 0; i < 25; i++ {
		tenant := &models.Tenant{
			Name:   fmt.Sprintf("Pagination Test Store %d", i),
			Slug:   fmt.Sprintf("pagination-test-store-%d", i),
			Email:  fmt.Sprintf("paginationtest%d@example.com", i),
			Status: models.StatusActive,
			Tier:   models.TierFree,
		}
		err := suite.db.Create(tenant).Error
		assert.NoError(suite.T(), err)
	}

	// Test first page
	req, _ := http.NewRequest("GET", "/api/v1/tenants?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response1 struct {
		Data       []models.TenantResponse `json:"data"`
		Pagination struct {
			Page       int   `json:"page"`
			PageSize   int   `json:"page_size"`
			Total      int64 `json:"total"`
			TotalPages int64 `json:"total_pages"`
		} `json:"pagination"`
	}
	json.Unmarshal(w.Body.Bytes(), &response1)
	assert.LessOrEqual(suite.T(), len(response1.Data), 10)

	// Test second page
	req, _ = http.NewRequest("GET", "/api/v1/tenants?page=2&page_size=10", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
