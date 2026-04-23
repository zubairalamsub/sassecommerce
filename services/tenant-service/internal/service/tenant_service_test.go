package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ecommerce/tenant-service/internal/models"
	repoMocks "github.com/ecommerce/tenant-service/internal/repository/mocks"
	kafkaMocks "github.com/ecommerce/tenant-service/pkg/kafka/mocks"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TenantServiceTestSuite struct {
	suite.Suite
	mockRepo     *repoMocks.MockTenantRepository
	mockKafka    *kafkaMocks.MockKafkaProducer
	service      TenantService
	logger       *logrus.Logger
}

func (suite *TenantServiceTestSuite) SetupTest() {
	suite.mockRepo = new(repoMocks.MockTenantRepository)
	suite.mockKafka = new(kafkaMocks.MockKafkaProducer)
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	suite.service = NewTenantService(suite.mockRepo, suite.mockKafka, suite.logger)
}

func (suite *TenantServiceTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
	suite.mockKafka.AssertExpectations(suite.T())
}

func (suite *TenantServiceTestSuite) TestCreateTenant_Success() {
	ctx := context.Background()

	req := &models.CreateTenantRequest{
		Name:  "Test Store",
		Email: "test@example.com",
		Tier:  "free",
	}

	// Mock GetBySlug to return nil (no existing tenant)
	suite.mockRepo.On("GetBySlug", ctx, mock.AnythingOfType("string")).Return(nil, errors.New("not found"))

	// Mock Create
	suite.mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Tenant")).Return(nil)

	// Mock Kafka publish
	suite.mockKafka.On("Publish", ctx, "tenant-events", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

	result, err := suite.service.CreateTenant(ctx, req)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "Test Store", result.Name)
	assert.Equal(suite.T(), "test@example.com", result.Email)
	assert.Equal(suite.T(), models.TierFree, result.Tier)
	assert.Equal(suite.T(), models.StatusPending, result.Status)
}

func (suite *TenantServiceTestSuite) TestCreateTenant_DuplicateSlug() {
	ctx := context.Background()

	req := &models.CreateTenantRequest{
		Name:  "Test Store",
		Email: "test@example.com",
		Tier:  "free",
	}

	existingTenant := &models.Tenant{
		ID:   uuid.New().String(),
		Name: "Existing Store",
		Slug: "test-store-123",
	}

	// Mock GetBySlug to return existing tenant
	suite.mockRepo.On("GetBySlug", ctx, mock.AnythingOfType("string")).Return(existingTenant, nil)

	result, err := suite.service.CreateTenant(ctx, req)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "already exists")
}

func (suite *TenantServiceTestSuite) TestGetTenant_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	expectedTenant := &models.Tenant{
		ID:     tenantID,
		Name:   "Test Store",
		Slug:   "test-store",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}

	suite.mockRepo.On("GetByID", ctx, tenantID).Return(expectedTenant, nil)

	result, err := suite.service.GetTenant(ctx, tenantID)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), tenantID, result.ID)
	assert.Equal(suite.T(), "Test Store", result.Name)
}

func (suite *TenantServiceTestSuite) TestGetTenant_NotFound() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	suite.mockRepo.On("GetByID", ctx, tenantID).Return(nil, errors.New("tenant not found"))

	result, err := suite.service.GetTenant(ctx, tenantID)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}

func (suite *TenantServiceTestSuite) TestGetTenantBySlug_Success() {
	ctx := context.Background()
	slug := "test-store"

	expectedTenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   slug,
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}

	suite.mockRepo.On("GetBySlug", ctx, slug).Return(expectedTenant, nil)

	result, err := suite.service.GetTenantBySlug(ctx, slug)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), slug, result.Slug)
}

func (suite *TenantServiceTestSuite) TestListTenants_Success() {
	ctx := context.Background()

	expectedTenants := []models.Tenant{
		{
			ID:     uuid.New().String(),
			Name:   "Store 1",
			Slug:   "store-1",
			Email:  "store1@example.com",
			Status: models.StatusActive,
			Tier:   models.TierFree,
		},
		{
			ID:     uuid.New().String(),
			Name:   "Store 2",
			Slug:   "store-2",
			Email:  "store2@example.com",
			Status: models.StatusActive,
			Tier:   models.TierProfessional,
		},
	}

	suite.mockRepo.On("List", ctx, 1, 20).Return(expectedTenants, int64(2), nil)

	result, total, err := suite.service.ListTenants(ctx, 1, 20)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), int64(2), total)
}

func (suite *TenantServiceTestSuite) TestUpdateTenant_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	existingTenant := &models.Tenant{
		ID:     tenantID,
		Name:   "Old Name",
		Slug:   "test-store",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}

	newName := "New Name"
	newStatus := string(models.StatusSuspended)
	updateReq := &models.UpdateTenantRequest{
		Name:   &newName,
		Status: &newStatus,
	}

	suite.mockRepo.On("GetByID", ctx, tenantID).Return(existingTenant, nil)
	suite.mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Tenant")).Return(nil)
	suite.mockKafka.On("Publish", ctx, "tenant-events", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

	result, err := suite.service.UpdateTenant(ctx, tenantID, updateReq)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.Equal(suite.T(), "New Name", result.Name)
	assert.Equal(suite.T(), models.StatusSuspended, result.Status)
}

func (suite *TenantServiceTestSuite) TestUpdateTenant_NotFound() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	updateReq := &models.UpdateTenantRequest{}

	suite.mockRepo.On("GetByID", ctx, tenantID).Return(nil, errors.New("tenant not found"))

	result, err := suite.service.UpdateTenant(ctx, tenantID, updateReq)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
}

func (suite *TenantServiceTestSuite) TestDeleteTenant_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	suite.mockRepo.On("Delete", ctx, tenantID).Return(nil)
	suite.mockKafka.On("Publish", ctx, "tenant-events", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(nil)

	err := suite.service.DeleteTenant(ctx, tenantID)

	assert.NoError(suite.T(), err)
}

func (suite *TenantServiceTestSuite) TestUpdateTenantConfig_Success() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	existingTenant := &models.Tenant{
		ID:     tenantID,
		Name:   "Test Store",
		Slug:   "test-store",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierProfessional,
	}

	newConfig := &models.TenantConfig{
		General: models.GeneralConfig{
			Timezone: "Asia/Dhaka",
			Currency: "BDT",
		},
		Branding: models.BrandingConfig{
			PrimaryColor: "#FF0000",
		},
	}

	suite.mockRepo.On("GetByID", ctx, tenantID).Return(existingTenant, nil)
	suite.mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Tenant")).Return(nil)

	err := suite.service.UpdateTenantConfig(ctx, tenantID, newConfig)

	assert.NoError(suite.T(), err)
}

func TestTenantServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TenantServiceTestSuite))
}
