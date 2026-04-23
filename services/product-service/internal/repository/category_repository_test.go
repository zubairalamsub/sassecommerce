package repository

import (
	"context"
	"testing"
	"time"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CategoryRepositoryTestSuite struct {
	suite.Suite
	client     *mongo.Client
	db         *mongo.Database
	repository CategoryRepository
	ctx        context.Context
}

func (suite *CategoryRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Connect to MongoDB (use test database)
	uri := "mongodb://localhost:27017"
	clientOptions := options.Client().ApplyURI(uri).SetConnectTimeout(3 * time.Second).SetServerSelectionTimeout(3 * time.Second)
	client, err := mongo.Connect(suite.ctx, clientOptions)
	if err != nil {
		suite.T().Skipf("Skipping integration test: cannot connect to MongoDB: %v", err)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(suite.ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(suite.ctx)
		suite.T().Skipf("Skipping integration test: MongoDB not available: %v", err)
	}

	suite.client = client
	suite.db = client.Database("category_test_db")
	suite.repository = NewCategoryRepository(suite.db)
}

func (suite *CategoryRepositoryTestSuite) TearDownSuite() {
	if suite.client != nil {
		// Drop test database
		_ = suite.db.Drop(suite.ctx)
		_ = suite.client.Disconnect(suite.ctx)
	}
}

func (suite *CategoryRepositoryTestSuite) SetupTest() {
	// Clear collection before each test
	_ = suite.db.Collection("categories").Drop(suite.ctx)
}

func TestCategoryRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(CategoryRepositoryTestSuite))
}

func (suite *CategoryRepositoryTestSuite) TestCreate() {
	category := &models.Category{
		TenantID:    "tenant-1",
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic products",
		SortOrder:   1,
		Status:      models.CategoryStatusActive,
		CreatedBy:   "user-1",
	}

	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)
	suite.NotEqual(primitive.NilObjectID, category.ID)
	suite.NotZero(category.CreatedAt)
	suite.NotZero(category.UpdatedAt)
}

func (suite *CategoryRepositoryTestSuite) TestGetByID() {
	// Create a category first
	category := &models.Category{
		TenantID:  "tenant-1",
		Name:      "Computers",
		Slug:      "computers",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)

	// Get by ID
	retrieved, err := suite.repository.GetByID(suite.ctx, category.ID.Hex())
	suite.NoError(err)
	suite.Equal(category.ID, retrieved.ID)
	suite.Equal(category.Name, retrieved.Name)
	suite.Equal(category.Slug, retrieved.Slug)
}

func (suite *CategoryRepositoryTestSuite) TestGetByID_NotFound() {
	nonExistentID := primitive.NewObjectID().Hex()
	_, err := suite.repository.GetByID(suite.ctx, nonExistentID)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *CategoryRepositoryTestSuite) TestGetByID_InvalidID() {
	_, err := suite.repository.GetByID(suite.ctx, "invalid-id")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid")
}

func (suite *CategoryRepositoryTestSuite) TestGetBySlug() {
	// Create a category
	category := &models.Category{
		TenantID:  "tenant-1",
		Name:      "Laptops",
		Slug:      "laptops",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)

	// Get by slug
	retrieved, err := suite.repository.GetBySlug(suite.ctx, "tenant-1", "laptops")
	suite.NoError(err)
	suite.Equal(category.ID, retrieved.ID)
	suite.Equal(category.Slug, retrieved.Slug)
}

func (suite *CategoryRepositoryTestSuite) TestGetBySlug_NotFound() {
	_, err := suite.repository.GetBySlug(suite.ctx, "tenant-1", "non-existent")
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *CategoryRepositoryTestSuite) TestSlugExists() {
	// Create a category
	category := &models.Category{
		TenantID:  "tenant-1",
		Name:      "Phones",
		Slug:      "phones",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)

	// Check if slug exists
	exists, err := suite.repository.SlugExists(suite.ctx, "tenant-1", "phones")
	suite.NoError(err)
	suite.True(exists)

	// Check non-existent slug
	exists, err = suite.repository.SlugExists(suite.ctx, "tenant-1", "non-existent")
	suite.NoError(err)
	suite.False(exists)
}

func (suite *CategoryRepositoryTestSuite) TestList() {
	// Create multiple categories
	for i := 0; i < 5; i++ {
		category := &models.Category{
			TenantID:  "tenant-1",
			Name:      "Category",
			Slug:      primitive.NewObjectID().Hex(),
			SortOrder: i,
			Status:    models.CategoryStatusActive,
			CreatedBy: "user-1",
		}
		err := suite.repository.Create(suite.ctx, category)
		suite.NoError(err)
		time.Sleep(10 * time.Millisecond)
	}

	// List categories
	categories, total, err := suite.repository.List(suite.ctx, "tenant-1", 0, 10)
	suite.NoError(err)
	suite.Equal(int64(5), total)
	suite.Len(categories, 5)
	// Verify sorted by sort_order
	for i := 0; i < 4; i++ {
		suite.LessOrEqual(categories[i].SortOrder, categories[i+1].SortOrder)
	}
}

func (suite *CategoryRepositoryTestSuite) TestList_Pagination() {
	// Create 10 categories
	for i := 0; i < 10; i++ {
		category := &models.Category{
			TenantID:  "tenant-1",
			Name:      "Category",
			Slug:      primitive.NewObjectID().Hex(),
			SortOrder: i,
			Status:    models.CategoryStatusActive,
			CreatedBy: "user-1",
		}
		err := suite.repository.Create(suite.ctx, category)
		suite.NoError(err)
		time.Sleep(10 * time.Millisecond)
	}

	// Get first page
	categories, total, err := suite.repository.List(suite.ctx, "tenant-1", 0, 5)
	suite.NoError(err)
	suite.Equal(int64(10), total)
	suite.Len(categories, 5)

	// Get second page
	categories, total, err = suite.repository.List(suite.ctx, "tenant-1", 5, 5)
	suite.NoError(err)
	suite.Equal(int64(10), total)
	suite.Len(categories, 5)
}

func (suite *CategoryRepositoryTestSuite) TestListByParent() {
	// Create parent category
	parent := &models.Category{
		TenantID:  "tenant-1",
		Name:      "Parent",
		Slug:      "parent",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, parent)
	suite.NoError(err)

	parentID := parent.ID.Hex()

	// Create child categories
	for i := 0; i < 3; i++ {
		category := &models.Category{
			TenantID:  "tenant-1",
			Name:      "Child",
			Slug:      primitive.NewObjectID().Hex(),
			ParentID:  &parentID,
			SortOrder: i,
			Status:    models.CategoryStatusActive,
			CreatedBy: "user-1",
		}
		err := suite.repository.Create(suite.ctx, category)
		suite.NoError(err)
	}

	// Create root categories (no parent)
	for i := 0; i < 2; i++ {
		category := &models.Category{
			TenantID:  "tenant-1",
			Name:      "Root",
			Slug:      primitive.NewObjectID().Hex(),
			SortOrder: i,
			Status:    models.CategoryStatusActive,
			CreatedBy: "user-1",
		}
		err := suite.repository.Create(suite.ctx, category)
		suite.NoError(err)
	}

	// List child categories
	categories, total, err := suite.repository.ListByParent(suite.ctx, "tenant-1", &parentID, 0, 10)
	suite.NoError(err)
	suite.Equal(int64(3), total)
	suite.Len(categories, 3)

	// List root categories (nil parent)
	categories, total, err = suite.repository.ListByParent(suite.ctx, "tenant-1", nil, 0, 10)
	suite.NoError(err)
	suite.Equal(int64(3), total) // Parent + 2 root categories
	suite.Len(categories, 3)
}

func (suite *CategoryRepositoryTestSuite) TestUpdate() {
	// Create a category
	category := &models.Category{
		TenantID:  "tenant-1",
		Name:      "Original Name",
		Slug:      "original-slug",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)

	// Update the category
	category.Name = "Updated Name"
	category.Slug = "updated-slug"
	category.SortOrder = 2
	err = suite.repository.Update(suite.ctx, category.ID.Hex(), category)
	suite.NoError(err)

	// Verify update
	retrieved, err := suite.repository.GetByID(suite.ctx, category.ID.Hex())
	suite.NoError(err)
	suite.Equal("Updated Name", retrieved.Name)
	suite.Equal("updated-slug", retrieved.Slug)
	suite.Equal(2, retrieved.SortOrder)
}

func (suite *CategoryRepositoryTestSuite) TestDelete() {
	// Create a category
	category := &models.Category{
		TenantID:  "tenant-1",
		Name:      "To Delete",
		Slug:      "to-delete",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)

	// Delete the category
	err = suite.repository.Delete(suite.ctx, category.ID.Hex())
	suite.NoError(err)

	// Verify deletion (soft delete)
	_, err = suite.repository.GetByID(suite.ctx, category.ID.Hex())
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *CategoryRepositoryTestSuite) TestUpdateStatus() {
	// Create a category
	category := &models.Category{
		TenantID:  "tenant-1",
		Name:      "Test Category",
		Slug:      "test-category",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
		CreatedBy: "user-1",
	}
	err := suite.repository.Create(suite.ctx, category)
	suite.NoError(err)

	// Update status
	err = suite.repository.UpdateStatus(suite.ctx, category.ID.Hex(), models.CategoryStatusInactive)
	suite.NoError(err)

	// Verify status update
	retrieved, err := suite.repository.GetByID(suite.ctx, category.ID.Hex())
	suite.NoError(err)
	suite.Equal(models.CategoryStatusInactive, retrieved.Status)
}
