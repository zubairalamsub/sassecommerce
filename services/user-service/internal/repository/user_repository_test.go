package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ecommerce/user-service/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type UserRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo UserRepository
}

func (suite *UserRepositoryTestSuite) SetupTest() {
	// Create in-memory SQLite database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(suite.T(), err)

	// Run migrations
	err = db.AutoMigrate(&models.User{})
	assert.NoError(suite.T(), err)

	suite.db = db
	suite.repo = NewUserRepository(db)
}

func (suite *UserRepositoryTestSuite) TearDownTest() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *UserRepositoryTestSuite) TestCreate() {
	ctx := context.Background()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     uuid.New().String(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), user.CreatedAt)
}

func (suite *UserRepositoryTestSuite) TestGetByID() {
	ctx := context.Background()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     uuid.New().String(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	found, err := suite.repo.GetByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Email, found.Email)
	assert.Equal(suite.T(), user.Username, found.Username)
}

func (suite *UserRepositoryTestSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	_, err := suite.repo.GetByID(ctx, uuid.New().String())
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "user not found", err.Error())
}

func (suite *UserRepositoryTestSuite) TestGetByEmail() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	found, err := suite.repo.GetByEmail(ctx, tenantID, user.Email)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
	assert.Equal(suite.T(), user.Email, found.Email)
}

func (suite *UserRepositoryTestSuite) TestGetByUsername() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	found, err := suite.repo.GetByUsername(ctx, tenantID, user.Username)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, found.ID)
	assert.Equal(suite.T(), user.Username, found.Username)
}

func (suite *UserRepositoryTestSuite) TestList() {
	ctx := context.Background()
	tenantID := uuid.New().String()

	// Create multiple users
	for i := 0; i < 5; i++ {
		user := &models.User{
			ID:           uuid.New().String(),
			TenantID:     tenantID,
			Email:        "test" + uuid.New().String() + "@example.com",
			Username:     "testuser" + uuid.New().String(),
			PasswordHash: "hashedpassword",
			FirstName:    "Test",
			LastName:     "User",
			Status:       models.UserStatusActive,
			Role:         models.UserRoleCustomer,
		}
		err := suite.repo.Create(ctx, user)
		assert.NoError(suite.T(), err)
	}

	users, total, err := suite.repo.List(ctx, tenantID, 0, 10)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), int64(5), total)
	assert.Len(suite.T(), users, 5)
}

func (suite *UserRepositoryTestSuite) TestUpdate() {
	ctx := context.Background()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     uuid.New().String(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	user.FirstName = "Updated"
	user.LastName = "Name"
	err = suite.repo.Update(ctx, user)
	assert.NoError(suite.T(), err)

	found, err := suite.repo.GetByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated", found.FirstName)
	assert.Equal(suite.T(), "Name", found.LastName)
}

func (suite *UserRepositoryTestSuite) TestUpdateLastLogin() {
	ctx := context.Background()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     uuid.New().String(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), user.LastLoginAt)

	time.Sleep(100 * time.Millisecond)

	err = suite.repo.UpdateLastLogin(ctx, user.ID)
	assert.NoError(suite.T(), err)

	found, err := suite.repo.GetByID(ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), found.LastLoginAt)
}

func (suite *UserRepositoryTestSuite) TestDelete() {
	ctx := context.Background()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     uuid.New().String(),
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	err = suite.repo.Delete(ctx, user.ID)
	assert.NoError(suite.T(), err)

	// Should not find deleted user
	_, err = suite.repo.GetByID(ctx, user.ID)
	assert.Error(suite.T(), err)
}

func (suite *UserRepositoryTestSuite) TestEmailExists() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	exists, err := suite.repo.EmailExists(ctx, tenantID, "test@example.com")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	exists, err = suite.repo.EmailExists(ctx, tenantID, "nonexistent@example.com")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *UserRepositoryTestSuite) TestUsernameExists() {
	ctx := context.Background()
	tenantID := uuid.New().String()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "Test",
		LastName:     "User",
		Status:       models.UserStatusActive,
		Role:         models.UserRoleCustomer,
	}

	err := suite.repo.Create(ctx, user)
	assert.NoError(suite.T(), err)

	exists, err := suite.repo.UsernameExists(ctx, tenantID, "testuser")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	exists, err = suite.repo.UsernameExists(ctx, tenantID, "nonexistent")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
