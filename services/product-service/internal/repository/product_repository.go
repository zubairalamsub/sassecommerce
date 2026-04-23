package repository

import (
	"context"
	"errors"
	"time"

	"github.com/ecommerce/product-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ProductRepository defines the interface for product data operations
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id string) (*models.Product, error)
	GetBySKU(ctx context.Context, tenantID, sku string) (*models.Product, error)
	List(ctx context.Context, tenantID string, offset, limit int) ([]models.Product, int64, error)
	ListByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.Product, int64, error)
	Search(ctx context.Context, tenantID, query string, offset, limit int) ([]models.Product, int64, error)
	Update(ctx context.Context, id string, product *models.Product) error
	Delete(ctx context.Context, id string) error
	SKUExists(ctx context.Context, tenantID, sku string) (bool, error)
	UpdateStatus(ctx context.Context, id string, status models.ProductStatus) error
}

type productRepository struct {
	collection *mongo.Collection
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *mongo.Database) ProductRepository {
	return &productRepository{
		collection: db.Collection("products"),
	}
}

// Create creates a new product
func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, product)
	if err != nil {
		return err
	}

	product.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a product by ID
func (r *productRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid product ID")
	}

	var product models.Product
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	err = r.collection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &product, nil
}

// GetBySKU retrieves a product by SKU within a tenant
func (r *productRepository) GetBySKU(ctx context.Context, tenantID, sku string) (*models.Product, error) {
	var product models.Product
	filter := bson.M{
		"tenant_id":  tenantID,
		"sku":        sku,
		"deleted_at": bson.M{"$exists": false},
	}

	err := r.collection.FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("product not found")
		}
		return nil, err
	}

	return &product, nil
}

// List retrieves products with pagination
func (r *productRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]models.Product, int64, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// ListByCategory retrieves products by category with pagination
func (r *productRepository) ListByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.Product, int64, error) {
	filter := bson.M{
		"tenant_id":   tenantID,
		"category_id": categoryID,
		"deleted_at":  bson.M{"$exists": false},
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Search searches products by name, description, or tags
func (r *productRepository) Search(ctx context.Context, tenantID, query string, offset, limit int) ([]models.Product, int64, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
		"$or": []bson.M{
			{"name": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
			{"tags": bson.M{"$in": []string{query}}},
		},
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	opts := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Update updates a product
func (r *productRepository) Update(ctx context.Context, id string, product *models.Product) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
	}

	product.UpdatedAt = time.Now()
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	update := bson.M{"$set": product}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("product not found")
	}

	return nil
}

// Delete soft deletes a product
func (r *productRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
	}

	now := time.Now()
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
			"status":     models.ProductStatusArchived,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("product not found")
	}

	return nil
}

// SKUExists checks if a SKU already exists for a tenant
func (r *productRepository) SKUExists(ctx context.Context, tenantID, sku string) (bool, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"sku":        sku,
		"deleted_at": bson.M{"$exists": false},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateStatus updates a product's status
func (r *productRepository) UpdateStatus(ctx context.Context, id string, status models.ProductStatus) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
	}

	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("product not found")
	}

	return nil
}
