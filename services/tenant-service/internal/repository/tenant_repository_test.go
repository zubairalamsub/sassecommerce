package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TenantRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo TenantRepository
}

func (suite *TenantRepositoryTestSuite) SetupTest() {
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Auto migrate
	err = db.AutoMigrate(&models.Tenant{})
	assert.NoError(suite.T(), err)

	suite.db = db
	suite.repo = NewTenantRepository(db)
}

func (suite *TenantRepositoryTestSuite) TearDownTest() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *TenantRepositoryTestSuite) TestCreate() {
	ctx := context.Background()

	tenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   "test-store-123",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}

	err := suite.repo.Create(ctx, tenant)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), tenant.CreatedAt)
}

func (suite *TenantRepositoryTestSuite) TestGetByID() {
	ctx := context.Background()

	// Create a tenant
	tenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   "test-store-123",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.repo.Create(ctx, tenant)
	assert.NoError(suite.T(), err)

	// Retrieve the tenant
	retrieved, err := suite.repo.GetByID(ctx, tenant.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), tenant.ID, retrieved.ID)
	assert.Equal(suite.T(), tenant.Name, retrieved.Name)
	assert.Equal(suite.T(), tenant.Email, retrieved.Email)
}

func (suite *TenantRepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	_, err := suite.repo.GetByID(ctx, uuid.New().String())
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")
}

func (suite *TenantRepositoryTestSuite) TestGetBySlug() {
	ctx := context.Background()

	tenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   "test-store-unique",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.repo.Create(ctx, tenant)
	assert.NoError(suite.T(), err)

	retrieved, err := suite.repo.GetBySlug(ctx, "test-store-unique")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), tenant.Slug, retrieved.Slug)
}

func (suite *TenantRepositoryTestSuite) TestGetByDomain() {
	ctx := context.Background()

	tenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   "test-store-123",
		Email:  "test@example.com",
		Domain: "example.mystore.com",
		Status: models.StatusActive,
		Tier:   models.TierProfessional,
	}
	err := suite.repo.Create(ctx, tenant)
	assert.NoError(suite.T(), err)

	retrieved, err := suite.repo.GetByDomain(ctx, "example.mystore.com")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), tenant.Domain, retrieved.Domain)
}

func (suite *TenantRepositoryTestSuite) TestList() {
	ctx := context.Background()

	// Create multiple tenants
	for i := 0; i < 5; i++ {
		tenant := &models.Tenant{
			ID:     uuid.New().String(),
			Name:   "Test Store " + string(rune(i)),
			Slug:   "test-store-" + uuid.New().String()[:8],
			Email:  "test" + string(rune(i)) + "@example.com",
			Status: models.StatusActive,
			Tier:   models.TierFree,
		}
		err := suite.repo.Create(ctx, tenant)
		assert.NoError(suite.T(), err)
	}

	tenants, total, err := suite.repo.List(ctx, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), total)
	assert.Len(suite.T(), tenants, 5)
}

func (suite *TenantRepositoryTestSuite) TestList_Pagination() {
	ctx := context.Background()

	// Create multiple tenants
	for i := 0; i < 15; i++ {
		tenant := &models.Tenant{
			ID:     uuid.New().String(),
			Name:   "Test Store " + string(rune(i)),
			Slug:   "test-store-" + uuid.New().String()[:8],
			Email:  "test" + string(rune(i)) + "@example.com",
			Status: models.StatusActive,
			Tier:   models.TierFree,
		}
		err := suite.repo.Create(ctx, tenant)
		assert.NoError(suite.T(), err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// First page
	tenants, total, err := suite.repo.List(ctx, 1, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(15), total)
	assert.Len(suite.T(), tenants, 10)

	// Second page
	tenants, total, err = suite.repo.List(ctx, 2, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(15), total)
	assert.Len(suite.T(), tenants, 5)
}

func (suite *TenantRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()

	tenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   "test-store-123",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.repo.Create(ctx, tenant)
	assert.NoError(suite.T(), err)

	// Update tenant
	tenant.Name = "Updated Store Name"
	tenant.Status = models.StatusSuspended
	err = suite.repo.Update(ctx, tenant)
	assert.NoError(suite.T(), err)

	// Retrieve and verify
	updated, err := suite.repo.GetByID(ctx, tenant.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Store Name", updated.Name)
	assert.Equal(suite.T(), models.StatusSuspended, updated.Status)
}

func (suite *TenantRepositoryTestSuite) TestDelete() {
	ctx := context.Background()

	tenant := &models.Tenant{
		ID:     uuid.New().String(),
		Name:   "Test Store",
		Slug:   "test-store-123",
		Email:  "test@example.com",
		Status: models.StatusActive,
		Tier:   models.TierFree,
	}
	err := suite.repo.Create(ctx, tenant)
	assert.NoError(suite.T(), err)

	// Delete tenant
	err = suite.repo.Delete(ctx, tenant.ID)
	assert.NoError(suite.T(), err)

	// Verify it's deleted (soft delete)
	_, err = suite.repo.GetByID(ctx, tenant.ID)
	assert.Error(suite.T(), err)
}

func (suite *TenantRepositoryTestSuite) TestCount() {
	ctx := context.Background()

	// Create multiple tenants
	for i := 0; i < 7; i++ {
		tenant := &models.Tenant{
			ID:     uuid.New().String(),
			Name:   "Test Store " + string(rune(i)),
			Slug:   "test-store-" + uuid.New().String()[:8],
			Email:  "test" + string(rune(i)) + "@example.com",
			Status: models.StatusActive,
			Tier:   models.TierFree,
		}
		err := suite.repo.Create(ctx, tenant)
		assert.NoError(suite.T(), err)
	}

	count, err := suite.repo.Count(ctx)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(7), count)
}

func TestTenantRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(TenantRepositoryTestSuite))
}
