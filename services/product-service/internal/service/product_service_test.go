package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ecommerce/product-service/internal/mocks"
	"github.com/ecommerce/product-service/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductServiceTestSuite struct {
	suite.Suite
	mockProductRepo  *mocks.MockProductRepository
	mockCategoryRepo *mocks.MockCategoryRepository
	service          ProductService
	logger           *logrus.Logger
	ctx              context.Context
}

func (suite *ProductServiceTestSuite) SetupTest() {
	suite.mockProductRepo = new(mocks.MockProductRepository)
	suite.mockCategoryRepo = new(mocks.MockCategoryRepository)
	suite.logger = logrus.New()
	suite.logger.SetOutput(io.Discard)
	suite.service = NewProductService(suite.mockProductRepo, suite.mockCategoryRepo, suite.logger)
	suite.ctx = context.Background()
}

func TestProductServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProductServiceTestSuite))
}

func (suite *ProductServiceTestSuite) TestCreateProduct_Success() {
	req := &models.CreateProductRequest{
		TenantID:    "tenant-1",
		SKU:         "PROD-001",
		Name:        "Test Product",
		Description: "Test Description",
		CategoryID:  "cat-1",
		Price:       99.99,
		CreatedBy:   "user-1",
		Variants:    []models.ProductVariant{},
	}

	category := &models.Category{
		ID:       primitive.NewObjectID(),
		TenantID: "tenant-1",
		Name:     "Electronics",
	}

	suite.mockProductRepo.On("SKUExists", suite.ctx, "tenant-1", "PROD-001").Return(false, nil)
	suite.mockCategoryRepo.On("GetByID", suite.ctx, "cat-1").Return(category, nil)
	suite.mockProductRepo.On("Create", suite.ctx, mock.Anything).Return(nil)

	product, err := suite.service.CreateProduct(suite.ctx, req)

	suite.NoError(err)
	suite.NotNil(product)
	suite.Equal("Test Product", product.Name)
	suite.Equal(models.ProductStatusDraft, product.Status)
	suite.mockProductRepo.AssertExpectations(suite.T())
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestCreateProduct_SKUExists() {
	req := &models.CreateProductRequest{
		TenantID:   "tenant-1",
		SKU:        "PROD-001",
		Name:       "Test Product",
		CategoryID: "cat-1",
		Price:      99.99,
		CreatedBy:  "user-1",
	}

	suite.mockProductRepo.On("SKUExists", suite.ctx, "tenant-1", "PROD-001").Return(true, nil)

	product, err := suite.service.CreateProduct(suite.ctx, req)

	suite.Error(err)
	suite.Nil(product)
	suite.Contains(err.Error(), "SKU already exists")
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestCreateProduct_CategoryNotFound() {
	req := &models.CreateProductRequest{
		TenantID:   "tenant-1",
		SKU:        "PROD-001",
		Name:       "Test Product",
		CategoryID: "non-existent",
		Price:      99.99,
		CreatedBy:  "user-1",
	}

	suite.mockProductRepo.On("SKUExists", suite.ctx, "tenant-1", "PROD-001").Return(false, nil)
	suite.mockCategoryRepo.On("GetByID", suite.ctx, "non-existent").Return(nil, errors.New("category not found"))

	product, err := suite.service.CreateProduct(suite.ctx, req)

	suite.Error(err)
	suite.Nil(product)
	suite.Contains(err.Error(), "category not found")
	suite.mockProductRepo.AssertExpectations(suite.T())
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetProductByID_Success() {
	productID := primitive.NewObjectID()
	product := &models.Product{
		ID:       productID,
		TenantID: "tenant-1",
		SKU:      "PROD-001",
		Name:     "Test Product",
		Price:    99.99,
		Status:   models.ProductStatusActive,
	}

	suite.mockProductRepo.On("GetByID", suite.ctx, productID.Hex()).Return(product, nil)

	result, err := suite.service.GetProductByID(suite.ctx, productID.Hex())

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("Test Product", result.Name)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetProductByID_NotFound() {
	suite.mockProductRepo.On("GetByID", suite.ctx, "non-existent").Return(nil, errors.New("product not found"))

	result, err := suite.service.GetProductByID(suite.ctx, "non-existent")

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "product not found")
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestGetProductBySKU_Success() {
	product := &models.Product{
		ID:       primitive.NewObjectID(),
		TenantID: "tenant-1",
		SKU:      "PROD-001",
		Name:     "Test Product",
		Price:    99.99,
		Status:   models.ProductStatusActive,
	}

	suite.mockProductRepo.On("GetBySKU", suite.ctx, "tenant-1", "PROD-001").Return(product, nil)

	result, err := suite.service.GetProductBySKU(suite.ctx, "tenant-1", "PROD-001")

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("PROD-001", result.SKU)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestListProducts_Success() {
	products := []models.Product{
		{
			ID:       primitive.NewObjectID(),
			TenantID: "tenant-1",
			Name:     "Product 1",
			Price:    99.99,
		},
		{
			ID:       primitive.NewObjectID(),
			TenantID: "tenant-1",
			Name:     "Product 2",
			Price:    49.99,
		},
	}

	suite.mockProductRepo.On("List", suite.ctx, "tenant-1", 0, 20).Return(products, int64(2), nil)

	result, total, err := suite.service.ListProducts(suite.ctx, "tenant-1", 0, 20)

	suite.NoError(err)
	suite.Len(result, 2)
	suite.Equal(int64(2), total)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestListProductsByCategory_Success() {
	products := []models.Product{
		{
			ID:         primitive.NewObjectID(),
			TenantID:   "tenant-1",
			CategoryID: "cat-1",
			Name:       "Product 1",
			Price:      99.99,
		},
	}

	suite.mockProductRepo.On("ListByCategory", suite.ctx, "tenant-1", "cat-1", 0, 20).Return(products, int64(1), nil)

	result, total, err := suite.service.ListProductsByCategory(suite.ctx, "tenant-1", "cat-1", 0, 20)

	suite.NoError(err)
	suite.Len(result, 1)
	suite.Equal(int64(1), total)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestSearchProducts_Success() {
	products := []models.Product{
		{
			ID:       primitive.NewObjectID(),
			TenantID: "tenant-1",
			Name:     "Blue Laptop",
			Price:    999.99,
		},
	}

	suite.mockProductRepo.On("Search", suite.ctx, "tenant-1", "laptop", 0, 20).Return(products, int64(1), nil)

	result, total, err := suite.service.SearchProducts(suite.ctx, "tenant-1", "laptop", 0, 20)

	suite.NoError(err)
	suite.Len(result, 1)
	suite.Equal(int64(1), total)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestUpdateProduct_Success() {
	productID := primitive.NewObjectID()
	existingProduct := &models.Product{
		ID:         productID,
		TenantID:   "tenant-1",
		SKU:        "PROD-001",
		Name:       "Old Name",
		CategoryID: "cat-1",
		Price:      99.99,
		Status:     models.ProductStatusDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	newName := "New Name"
	newPrice := 149.99
	req := &models.UpdateProductRequest{
		Name:      &newName,
		Price:     &newPrice,
		UpdatedBy: "user-1",
	}

	suite.mockProductRepo.On("GetByID", suite.ctx, productID.Hex()).Return(existingProduct, nil)
	suite.mockProductRepo.On("Update", suite.ctx, productID.Hex(), mock.Anything).Return(nil)

	result, err := suite.service.UpdateProduct(suite.ctx, productID.Hex(), req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("New Name", result.Name)
	suite.Equal(149.99, result.Price)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestUpdateProduct_CategoryChange() {
	productID := primitive.NewObjectID()
	existingProduct := &models.Product{
		ID:         productID,
		TenantID:   "tenant-1",
		SKU:        "PROD-001",
		Name:       "Product",
		CategoryID: "cat-1",
		Price:      99.99,
		Status:     models.ProductStatusDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	newCategory := "cat-2"
	req := &models.UpdateProductRequest{
		CategoryID: &newCategory,
		UpdatedBy:  "user-1",
	}

	category := &models.Category{
		ID:       primitive.NewObjectID(),
		TenantID: "tenant-1",
		Name:     "New Category",
	}

	suite.mockProductRepo.On("GetByID", suite.ctx, productID.Hex()).Return(existingProduct, nil)
	suite.mockCategoryRepo.On("GetByID", suite.ctx, "cat-2").Return(category, nil)
	suite.mockProductRepo.On("Update", suite.ctx, productID.Hex(), mock.Anything).Return(nil)

	result, err := suite.service.UpdateProduct(suite.ctx, productID.Hex(), req)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("cat-2", result.CategoryID)
	suite.mockProductRepo.AssertExpectations(suite.T())
	suite.mockCategoryRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestDeleteProduct_Success() {
	productID := primitive.NewObjectID()

	suite.mockProductRepo.On("Delete", suite.ctx, productID.Hex()).Return(nil)

	err := suite.service.DeleteProduct(suite.ctx, productID.Hex())

	suite.NoError(err)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestDeleteProduct_NotFound() {
	suite.mockProductRepo.On("Delete", suite.ctx, "non-existent").Return(errors.New("product not found"))

	err := suite.service.DeleteProduct(suite.ctx, "non-existent")

	suite.Error(err)
	suite.mockProductRepo.AssertExpectations(suite.T())
}

func (suite *ProductServiceTestSuite) TestUpdateProductStatus_Success() {
	productID := primitive.NewObjectID()

	suite.mockProductRepo.On("UpdateStatus", suite.ctx, productID.Hex(), models.ProductStatusActive).Return(nil)

	err := suite.service.UpdateProductStatus(suite.ctx, productID.Hex(), models.ProductStatusActive)

	suite.NoError(err)
	suite.mockProductRepo.AssertExpectations(suite.T())
}
