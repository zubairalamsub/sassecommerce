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

// CategoryRepository defines the interface for category data operations
type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetBySlug(ctx context.Context, tenantID, slug string) (*models.Category, error)
	List(ctx context.Context, tenantID string, offset, limit int) ([]models.Category, int64, error)
	ListByParent(ctx context.Context, tenantID string, parentID *string, offset, limit int) ([]models.Category, int64, error)
	Update(ctx context.Context, tenantID, id string, category *models.Category) error
	Delete(ctx context.Context, tenantID, id string) error
	SlugExists(ctx context.Context, tenantID, slug string) (bool, error)
	UpdateStatus(ctx context.Context, tenantID, id string, status models.CategoryStatus) error
	UpdateImage(ctx context.Context, id string, imageURL string) error
}

type categoryRepository struct {
	collection *mongo.Collection
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *mongo.Database) CategoryRepository {
	return &categoryRepository{
		collection: db.Collection("categories"),
	}
}

// Create creates a new category
func (r *categoryRepository) Create(ctx context.Context, category *models.Category) error {
	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, category)
	if err != nil {
		return err
	}

	category.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a category by ID
func (r *categoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid category ID")
	}

	var category models.Category
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	err = r.collection.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	return &category, nil
}

// GetBySlug retrieves a category by slug within a tenant
func (r *categoryRepository) GetBySlug(ctx context.Context, tenantID, slug string) (*models.Category, error) {
	var category models.Category
	filter := bson.M{
		"tenant_id":  tenantID,
		"slug":       slug,
		"deleted_at": bson.M{"$exists": false},
	}

	err := r.collection.FindOne(ctx, filter).Decode(&category)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("category not found")
		}
		return nil, err
	}

	return &category, nil
}

// List retrieves categories with pagination
func (r *categoryRepository) List(ctx context.Context, tenantID string, offset, limit int) ([]models.Category, int64, error) {
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
		SetSort(bson.D{{Key: "sort_order", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var categories []models.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// ListByParent retrieves categories by parent ID
func (r *categoryRepository) ListByParent(ctx context.Context, tenantID string, parentID *string, offset, limit int) ([]models.Category, int64, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
	}

	if parentID == nil {
		filter["parent_id"] = bson.M{"$exists": false}
	} else {
		filter["parent_id"] = *parentID
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
		SetSort(bson.D{{Key: "sort_order", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var categories []models.Category
	if err := cursor.All(ctx, &categories); err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// Update updates a category. The tenant_id predicate ensures a tenant can only
// modify its own categories; a cross-tenant id yields no match (404 upstream).
func (r *categoryRepository) Update(ctx context.Context, tenantID, id string, category *models.Category) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid category ID")
	}

	category.UpdatedAt = time.Now()
	filter := bson.M{
		"_id":        objectID,
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
	}

	update := bson.M{"$set": category}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("category not found")
	}

	return nil
}

// Delete soft deletes a category. The tenant_id predicate ensures a tenant can
// only delete its own categories; a cross-tenant id yields no match (404 upstream).
func (r *categoryRepository) Delete(ctx context.Context, tenantID, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid category ID")
	}

	now := time.Now()
	filter := bson.M{
		"_id":        objectID,
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
			"status":     models.CategoryStatusInactive,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("category not found")
	}

	return nil
}

// SlugExists checks if a slug already exists for a tenant
func (r *categoryRepository) SlugExists(ctx context.Context, tenantID, slug string) (bool, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"slug":       slug,
		"deleted_at": bson.M{"$exists": false},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateImage sets the image field on a category. Passing an empty string clears it.
// Used by the category image service after a successful upload confirmation
// or when an image is removed.
func (r *categoryRepository) UpdateImage(ctx context.Context, id string, imageURL string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid category ID")
	}

	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"image":      imageURL,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("category not found")
	}

	return nil
}

// UpdateStatus updates a category's status. The tenant_id predicate ensures a
// tenant can only change its own categories' status; a cross-tenant id yields no
// match (404 upstream).
func (r *categoryRepository) UpdateStatus(ctx context.Context, tenantID, id string, status models.CategoryStatus) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid category ID")
	}

	filter := bson.M{
		"_id":        objectID,
		"tenant_id":  tenantID,
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
		return errors.New("category not found")
	}

	return nil
}
