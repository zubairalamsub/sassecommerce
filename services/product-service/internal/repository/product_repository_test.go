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

type ProductRepositoryTestSuite struct {
	suite.Suite
	client     *mongo.Client
	db         *mongo.Database
	repository ProductRepository
	ctx        context.Context
}

func (suite *ProductRepositoryTestSuite) SetupSuite() {
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
	suite.db = client.Database("product_test_db")
	suite.repository = NewProductRepository(suite.db)
}

func (suite *ProductRepositoryTestSuite) TearDownSuite() {
	if suite.client != nil {
		// Drop test database
		_ = suite.db.Drop(suite.ctx)
		_ = suite.client.Disconnect(suite.ctx)
	}
}

func (suite *ProductRepositoryTestSuite) SetupTest() {
	// Clear collection before each test
	_ = suite.db.Collection("products").Drop(suite.ctx)
}

func TestProductRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ProductRepositoryTestSuite))
}

func (suite *ProductRepositoryTestSuite) TestCreate() {
	product := &models.Product{
		TenantID:    "tenant-1",
		SKU:         "PROD-001",
		Name:        "Test Product",
		Description: "Test Description",
		CategoryID:  "cat-1",
		Brand:       "Test Brand",
		Price:       99.99,
		Images:      []string{"image1.jpg", "image2.jpg"},
		Tags:        []string{"tag1", "tag2"},
		Status:      models.ProductStatusDraft,
		CreatedBy:   "user-1",
	}

	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)
	suite.NotEqual(primitive.NilObjectID, product.ID)
	suite.NotZero(product.CreatedAt)
	suite.NotZero(product.UpdatedAt)
}

func (suite *ProductRepositoryTestSuite) TestGetByID() {
	// Create a product first
	product := &models.Product{
		TenantID:   "tenant-1",
		SKU:        "PROD-002",
		Name:       "Test Product 2",
		CategoryID: "cat-1",
		Price:      49.99,
		Status:     models.ProductStatusActive,
		CreatedBy:  "user-1",
	}
	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)

	// Get by ID
	retrieved, err := suite.repository.GetByID(suite.ctx, product.ID.Hex())
	suite.NoError(err)
	suite.Equal(product.ID, retrieved.ID)
	suite.Equal(product.SKU, retrieved.SKU)
	suite.Equal(product.Name, retrieved.Name)
}

func (suite *ProductRepositoryTestSuite) TestGetByID_NotFound() {
	nonExistentID := primitive.NewObjectID().Hex()
	_, err := suite.repository.GetByID(suite.ctx, nonExistentID)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *ProductRepositoryTestSuite) TestGetByID_InvalidID() {
	_, err := suite.repository.GetByID(suite.ctx, "invalid-id")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid")
}

func (suite *ProductRepositoryTestSuite) TestGetBySKU() {
	// Create a product
	product := &models.Product{
		TenantID:   "tenant-1",
		SKU:        "PROD-003",
		Name:       "Test Product 3",
		CategoryID: "cat-1",
		Price:      29.99,
		Status:     models.ProductStatusActive,
		CreatedBy:  "user-1",
	}
	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)

	// Get by SKU
	retrieved, err := suite.repository.GetBySKU(suite.ctx, "tenant-1", "PROD-003")
	suite.NoError(err)
	suite.Equal(product.ID, retrieved.ID)
	suite.Equal(product.SKU, retrieved.SKU)
}

func (suite *ProductRepositoryTestSuite) TestGetBySKU_NotFound() {
	_, err := suite.repository.GetBySKU(suite.ctx, "tenant-1", "NON-EXISTENT")
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *ProductRepositoryTestSuite) TestSKUExists() {
	// Create a product
	product := &models.Product{
		TenantID:   "tenant-1",
		SKU:        "PROD-004",
		Name:       "Test Product 4",
		CategoryID: "cat-1",
		Price:      19.99,
		Status:     models.ProductStatusActive,
		CreatedBy:  "user-1",
	}
	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)

	// Check if SKU exists
	exists, err := suite.repository.SKUExists(suite.ctx, "tenant-1", "PROD-004")
	suite.NoError(err)
	suite.True(exists)

	// Check non-existent SKU
	exists, err = suite.repository.SKUExists(suite.ctx, "tenant-1", "NON-EXISTENT")
	suite.NoError(err)
	suite.False(exists)
}

func (suite *ProductRepositoryTestSuite) TestList() {
	// Create multiple products
	for i := 0; i < 5; i++ {
		product := &models.Product{
			TenantID:   "tenant-1",
			SKU:        primitive.NewObjectID().Hex(),
			Name:       "Test Product",
			CategoryID: "cat-1",
			Price:      float64(10 + i),
			Status:     models.ProductStatusActive,
			CreatedBy:  "user-1",
		}
		err := suite.repository.Create(suite.ctx, product)
		suite.NoError(err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List products
	products, total, err := suite.repository.List(suite.ctx, "tenant-1", 0, 10)
	suite.NoError(err)
	suite.Equal(int64(5), total)
	suite.Len(products, 5)
}

func (suite *ProductRepositoryTestSuite) TestList_Pagination() {
	// Create 10 products
	for i := 0; i < 10; i++ {
		product := &models.Product{
			TenantID:   "tenant-1",
			SKU:        primitive.NewObjectID().Hex(),
			Name:       "Test Product",
			CategoryID: "cat-1",
			Price:      float64(10 + i),
			Status:     models.ProductStatusActive,
			CreatedBy:  "user-1",
		}
		err := suite.repository.Create(suite.ctx, product)
		suite.NoError(err)
		time.Sleep(10 * time.Millisecond)
	}

	// Get first page
	products, total, err := suite.repository.List(suite.ctx, "tenant-1", 0, 5)
	suite.NoError(err)
	suite.Equal(int64(10), total)
	suite.Len(products, 5)

	// Get second page
	products, total, err = suite.repository.List(suite.ctx, "tenant-1", 5, 5)
	suite.NoError(err)
	suite.Equal(int64(10), total)
	suite.Len(products, 5)
}

func (suite *ProductRepositoryTestSuite) TestListByCategory() {
	// Create products in different categories
	for i := 0; i < 3; i++ {
		product := &models.Product{
			TenantID:   "tenant-1",
			SKU:        primitive.NewObjectID().Hex(),
			Name:       "Test Product Cat1",
			CategoryID: "cat-1",
			Price:      float64(10 + i),
			Status:     models.ProductStatusActive,
			CreatedBy:  "user-1",
		}
		err := suite.repository.Create(suite.ctx, product)
		suite.NoError(err)
	}

	for i := 0; i < 2; i++ {
		product := &models.Product{
			TenantID:   "tenant-1",
			SKU:        primitive.NewObjectID().Hex(),
			Name:       "Test Product Cat2",
			CategoryID: "cat-2",
			Price:      float64(20 + i),
			Status:     models.ProductStatusActive,
			CreatedBy:  "user-1",
		}
		err := suite.repository.Create(suite.ctx, product)
		suite.NoError(err)
	}

	// List products by category
	products, total, err := suite.repository.ListByCategory(suite.ctx, "tenant-1", "cat-1", 0, 10)
	suite.NoError(err)
	suite.Equal(int64(3), total)
	suite.Len(products, 3)

	products, total, err = suite.repository.ListByCategory(suite.ctx, "tenant-1", "cat-2", 0, 10)
	suite.NoError(err)
	suite.Equal(int64(2), total)
	suite.Len(products, 2)
}

func (suite *ProductRepositoryTestSuite) TestSearch() {
	// Create products with different names and tags
	products := []struct {
		name string
		tags []string
	}{
		{"Blue Laptop", []string{"electronics", "computer"}},
		{"Red Laptop", []string{"electronics", "computer"}},
		{"Blue Shirt", []string{"clothing", "casual"}},
		{"Phone Case", []string{"accessories", "phone"}},
	}

	for _, p := range products {
		product := &models.Product{
			TenantID:   "tenant-1",
			SKU:        primitive.NewObjectID().Hex(),
			Name:       p.name,
			CategoryID: "cat-1",
			Price:      99.99,
			Tags:       p.tags,
			Status:     models.ProductStatusActive,
			CreatedBy:  "user-1",
		}
		err := suite.repository.Create(suite.ctx, product)
		suite.NoError(err)
	}

	// Search by name
	results, total, err := suite.repository.Search(suite.ctx, "tenant-1", "Laptop", 0, 10)
	suite.NoError(err)
	suite.Equal(int64(2), total)
	suite.Len(results, 2)

	// Search by tag
	results, total, err = suite.repository.Search(suite.ctx, "tenant-1", "electronics", 0, 10)
	suite.NoError(err)
	suite.GreaterOrEqual(int(total), 2)
}

func (suite *ProductRepositoryTestSuite) TestUpdate() {
	// Create a product
	product := &models.Product{
		TenantID:   "tenant-1",
		SKU:        "PROD-005",
		Name:       "Original Name",
		CategoryID: "cat-1",
		Price:      99.99,
		Status:     models.ProductStatusDraft,
		CreatedBy:  "user-1",
	}
	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)

	// Update the product
	product.Name = "Updated Name"
	product.Price = 149.99
	err = suite.repository.Update(suite.ctx, product.ID.Hex(), product)
	suite.NoError(err)

	// Verify update
	retrieved, err := suite.repository.GetByID(suite.ctx, product.ID.Hex())
	suite.NoError(err)
	suite.Equal("Updated Name", retrieved.Name)
	suite.Equal(149.99, retrieved.Price)
}

func (suite *ProductRepositoryTestSuite) TestDelete() {
	// Create a product
	product := &models.Product{
		TenantID:   "tenant-1",
		SKU:        "PROD-006",
		Name:       "To Delete",
		CategoryID: "cat-1",
		Price:      99.99,
		Status:     models.ProductStatusActive,
		CreatedBy:  "user-1",
	}
	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)

	// Delete the product
	err = suite.repository.Delete(suite.ctx, product.ID.Hex())
	suite.NoError(err)

	// Verify deletion (soft delete)
	_, err = suite.repository.GetByID(suite.ctx, product.ID.Hex())
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

func (suite *ProductRepositoryTestSuite) TestUpdateStatus() {
	// Create a product
	product := &models.Product{
		TenantID:   "tenant-1",
		SKU:        "PROD-007",
		Name:       "Test Product",
		CategoryID: "cat-1",
		Price:      99.99,
		Status:     models.ProductStatusDraft,
		CreatedBy:  "user-1",
	}
	err := suite.repository.Create(suite.ctx, product)
	suite.NoError(err)

	// Update status
	err = suite.repository.UpdateStatus(suite.ctx, product.ID.Hex(), models.ProductStatusActive)
	suite.NoError(err)

	// Verify status update
	retrieved, err := suite.repository.GetByID(suite.ctx, product.ID.Hex())
	suite.NoError(err)
	suite.Equal(models.ProductStatusActive, retrieved.Status)
}
