package service

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ecommerce/product-service/internal/mocks"
	"github.com/ecommerce/product-service/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategoryServiceTestSuite struct {
	suite.Suite
	mockCategoryRepo *mocks.MockCategoryRepository
	service          CategoryService
	logger           *logrus.Logger
	ctx              context.Context
}

func (suite *CategoryServiceTestSuite) SetupTest() {
	suite.mockCategoryRepo = new(mocks.MockCategoryRepository)
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard)
	suite.service = NewCategoryService(suite.mockCategoryRepo, suite.logger)
	suite.ctx = context.Background()
}

func TestCategoryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CategoryServiceTestSuite))
}

func (suite *CategoryServiceTestSuite) TestCreateCategory_Success() {
	req := &models.CreateCategoryRequest{
		TenantID:    "tenant-1",
		Name:        "Electronics",
		Slug:        "electronics",
		Description: "Electronic products",
		SortOrder:   1,
		CreatedBy:   "user-1",
	}

	suite.mockCategoryRepo.On("SlugExists", suite.ctx, "tenant-1", "electronics").Return(false, nil)
	suite.mockCategoryRepo.On("Create", suite.ctx, mock.Anything).Return(nil)

	category, err := suite.service.CreateCategory(suite.ctx, req)

	suite.NoError(err)
	suite.NotNil(category)
	suite.Equal("Electronics", category.Name)
	suite.Equal(models.CategoryStatusActive, category.Status)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestCreateCategory_SlugExists() {
	req := &models.CreateCategoryRequest{
		TenantID:  "tenant-1",
		Name:      "Electronics",
		Slug:      "electronics",
		SortOrder: 1,
		CreatedBy: "user-1",
	}

	suite.mockCategoryRepo.On("SlugExists", suite.ctx, "tenant-1", "electronics").Return(true, nil)

	category, err := suite.service.CreateCategory(suite.ctx, req)

	suite.Error(err)
	suite.Nil(category)
	suite.Contains(err.Error(), "slug already exists")
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestCreateCategory_WithParent() {
	parentID := primitive.NewObjectID().Hex()
	req := &models.CreateCategoryRequest{
		TenantID:  "tenant-1",
		Name:      "Laptops",
		Slug:      "laptops",
		ParentID:  &parentID,
		SortOrder: 1,
		CreatedBy: "user-1",
	}

	parent := &models.Category{
		ID:       primitive.NewObjectID(),
		TenantID: "tenant-1",
		Name:     "Electronics",
	}

	suite.mockCategoryRepo.On("SlugExists", suite.ctx, "tenant-1", "laptops").Return(false, nil)
	suite.mockCategoryRepo.On("GetByID", suite.ctx, parentID).Return(parent, nil)
	suite.mockCategoryRepo.On("Create", suite.ctx, mock.Anything).Return(nil)

	category, err := suite.service.CreateCategory(suite.ctx, req)

	suite.NoError(err)
	suite.NotNil(category)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestCreateCategory_ParentNotFound() {
	parentID := "non-existent"
	req := &models.CreateCategoryRequest{
		TenantID:  "tenant-1",
		Name:      "Laptops",
		Slug:      "laptops",
		ParentID:  &parentID,
		SortOrder: 1,
		CreatedBy: "user-1",
	}

	suite.mockCategoryRepo.On("SlugExists", suite.ctx, "tenant-1", "laptops").Return(false, nil)
	suite.mockCategoryRepo.On("GetByID", suite.ctx, parentID).Return(nil, errors.New("category not found"))

	category, err := suite.service.CreateCategory(suite.ctx, req)

	suite.Error(err)
	suite.Nil(category)
	suite.Contains(err.Error(), "parent category not found")
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetCategoryByID_Success() {
	categoryID := primitive.NewObjectID()
	category := &models.Category{
		ID:       categoryID,
		TenantID: "tenant-1",
		Name:     "Electronics",
		Slug:     "electronics",
		Status:   models.CategoryStatusActive,
	}

	suite.mockCategoryRepo.On("GetByID", suite.ctx, categoryID.Hex()).Return(category, nil)

	result, err := suite.service.GetCategoryByID(suite.ctx, categoryID.Hex())

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("Electronics", result.Name)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetCategoryByID_NotFound() {
	suite.mockCategoryRepo.On("GetByID", suite.ctx, "non-existent").Return(nil, errors.New("category not found"))

	result, err := suite.service.GetCategoryByID(suite.ctx, "non-existent")

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "category not found")
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestGetCategoryBySlug_Success() {
	category := &models.Category{
		ID:       primitive.NewObjectID(),
		TenantID: "tenant-1",
		Name:     "Electronics",
		Slug:     "electronics",
		Status:   models.CategoryStatusActive,
	}

	suite.mockCategoryRepo.On("GetBySlug", suite.ctx, "tenant-1", "electronics").Return(category, nil)

	result, err := suite.service.GetCategoryBySlug(suite.ctx, "tenant-1", "electronics")

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("electronics", result.Slug)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestListCategories_Success() {
	categories := []models.Category{
		{
			ID:       primitive.NewObjectID(),
			TenantID: "tenant-1",
			Name:     "Category 1",
			Slug:     "category-1",
		},
		{
			ID:       primitive.NewObjectID(),
			TenantID: "tenant-1",
			Name:     "Category 2",
			Slug:     "category-2",
		},
	}

	suite.mockCategoryRepo.On("List", suite.ctx, "tenant-1", 0, 20).Return(categories, int64(2), nil)

	result, total, err := suite.service.ListCategories(suite.ctx, "tenant-1", 0, 20)

	suite.NoError(err)
	suite.Len(result, 2)
	suite.Equal(int64(2), total)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestListCategoriesByParent_Success() {
	parentID := primitive.NewObjectID().Hex()
	categories := []models.Category{
		{
			ID:       primitive.NewObjectID(),
			TenantID: "tenant-1",
			Name:     "Child Category",
			ParentID: &parentID,
		},
	}

	suite.mockCategoryRepo.On("ListByParent", suite.ctx, "tenant-1", &parentID, 0, 20).Return(categories, int64(1), nil)

	result, total, err := suite.service.ListCategoriesByParent(suite.ctx, "tenant-1", &parentID, 0, 20)

	suite.NoError(err)
	suite.Len(result, 1)
	suite.Equal(int64(1), total)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestUpdateCategory_Success() {
	categoryID := primitive.NewObjectID()
	existingCategory := &models.Category{
		ID:        categoryID,
		TenantID:  "tenant-1",
		Name:      "Old Name",
		Slug:      "old-slug",
		SortOrder: 1,
		Status:    models.CategoryStatusActive,
	}

	newName := "New Name"
	newSlug := "new-slug"
	req := &models.UpdateCategoryRequest{
		Name:      &newName,
		Slug:      &newSlug,
		UpdatedBy: "user-1",
	}

	suite.mockCategoryRepo.On("GetByID", suite.ctx, categoryID.Hex()).Return(existingCategory, nil)
	suite.mockCategoryRepo.On("SlugExists", suite.ctx, "tenant-1", "new-slug").Return(false, nil)
	suite.mockCategoryRepo.On("Update", suite.ctx, categoryID.Hex(), mock.Anything).Return(nil)

	result, err := suite.service.UpdateCategory(suite.ctx, categoryID.Hex(), req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("New Name", result.Name)
	suite.Equal("new-slug", result.Slug)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestUpdateCategory_SlugAlreadyExists() {
	categoryID := primitive.NewObjectID()
	existingCategory := &models.Category{
		ID:       categoryID,
		TenantID: "tenant-1",
		Name:     "Category",
		Slug:     "old-slug",
	}

	newSlug := "existing-slug"
	req := &models.UpdateCategoryRequest{
		Slug:      &newSlug,
		UpdatedBy: "user-1",
	}

	suite.mockCategoryRepo.On("GetByID", suite.ctx, categoryID.Hex()).Return(existingCategory, nil)
	suite.mockCategoryRepo.On("SlugExists", suite.ctx, "tenant-1", "existing-slug").Return(true, nil)

	result, err := suite.service.UpdateCategory(suite.ctx, categoryID.Hex(), req)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "slug already exists")
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestDeleteCategory_Success() {
	categoryID := primitive.NewObjectID()

	suite.mockCategoryRepo.On("Delete", suite.ctx, categoryID.Hex()).Return(nil)

	err := suite.service.DeleteCategory(suite.ctx, categoryID.Hex())

	suite.NoError(err)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestDeleteCategory_NotFound() {
	suite.mockCategoryRepo.On("Delete", suite.ctx, "non-existent").Return(errors.New("category not found"))

	err := suite.service.DeleteCategory(suite.ctx, "non-existent")

	suite.Error(err)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *CategoryServiceTestSuite) TestUpdateCategoryStatus_Success() {
	categoryID := primitive.NewObjectID()

	suite.mockCategoryRepo.On("UpdateStatus", suite.ctx, categoryID.Hex(), models.CategoryStatusInactive).Return(nil)

	err := suite.service.UpdateCategoryStatus(suite.ctx, categoryID.Hex(), models.CategoryStatusInactive)

	suite.NoError(err)
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}
