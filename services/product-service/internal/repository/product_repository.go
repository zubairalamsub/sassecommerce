package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// productCacheKeyPrefix + hex ObjectID is the cache key for GetByID.
	productCacheKeyPrefix = "product:"
	// Short TTL: the cache is invalidated on every write, so this is only a
	// backstop for the rare invalidation miss (e.g. a stock update addressed
	// by SKU rather than id).
	productCacheTTL = 60 * time.Second
)

// ProductRepository defines the interface for product data operations
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id string) (*models.Product, error)
	GetBySKU(ctx context.Context, tenantID, sku string) (*models.Product, error)
	List(ctx context.Context, tenantID string, offset, limit int) ([]models.Product, int64, error)
	ListByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.Product, int64, error)
	Search(ctx context.Context, tenantID, query string, offset, limit int) ([]models.Product, int64, error)
	Update(ctx context.Context, tenantID, id string, product *models.Product) error
	Delete(ctx context.Context, tenantID, id string) error
	SKUExists(ctx context.Context, tenantID, sku string) (bool, error)
	UpdateStatus(ctx context.Context, tenantID, id string, status models.ProductStatus) error
	UpdateStock(ctx context.Context, tenantID, productID string, quantity int, inStock bool) error
	AddImage(ctx context.Context, id, imageURL string) error
	RemoveImage(ctx context.Context, id, imageURL string) error
	EnsureIndexes(ctx context.Context) error
}

type productRepository struct {
	collection *mongo.Collection
	// cache is optional: nil disables caching entirely. All cache operations
	// are best-effort — any Redis error falls through to MongoDB.
	cache *redis.Client
}

// NewProductRepository creates a new product repository. Pass a non-nil redis
// client to enable cache-aside on the GetByID read path; pass nil to disable.
func NewProductRepository(db *mongo.Database, cache *redis.Client) ProductRepository {
	return &productRepository{
		collection: db.Collection("products"),
		cache:      cache,
	}
}

func productCacheKey(id string) string { return productCacheKeyPrefix + id }

// cacheGetByID returns a cached product for the hex id, or (nil, false) on any
// miss/error. Best-effort — never surfaces cache errors to the caller.
func (r *productRepository) cacheGetByID(ctx context.Context, id string) (*models.Product, bool) {
	if r.cache == nil {
		return nil, false
	}
	data, err := r.cache.Get(ctx, productCacheKey(id)).Bytes()
	if err != nil {
		return nil, false
	}
	var p models.Product
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, false
	}
	return &p, true
}

// cacheSet stores a product under its hex id. Best-effort.
func (r *productRepository) cacheSet(ctx context.Context, p *models.Product) {
	if r.cache == nil || p == nil {
		return
	}
	data, err := json.Marshal(p)
	if err != nil {
		return
	}
	_ = r.cache.Set(ctx, productCacheKey(p.ID.Hex()), data, productCacheTTL).Err()
}

// cacheInvalidate drops the cached product for an id (hex ObjectID). Best-effort.
func (r *productRepository) cacheInvalidate(ctx context.Context, id string) {
	if r.cache == nil {
		return
	}
	_ = r.cache.Del(ctx, productCacheKey(id)).Err()
}

// EnsureIndexes creates the indexes backing the hot read paths. Without these,
// every storefront read (List/ListByCategory/GetBySKU/Search) is a full
// collection scan. Safe to call on every startup — CreateMany is idempotent
// for index specs that already exist.
func (r *productRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// List: filter by tenant_id + deleted_at, sort by created_at desc.
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "deleted_at", Value: 1}, {Key: "created_at", Value: -1}}},
		// ListByCategory.
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "category_id", Value: 1}, {Key: "deleted_at", Value: 1}}},
		// GetBySKU / SKUExists.
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "sku", Value: 1}}},
		// Status filters.
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "status", Value: 1}}},
		// Full-text search across name/description/tags (backs Search()).
		// A collection may have only one text index; this is it.
		{Keys: bson.D{{Key: "name", Value: "text"}, {Key: "description", Value: "text"}, {Key: "tags", Value: "text"}}},
	}
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
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

	// Cache-aside: the cached document still passes through the caller's
	// tenant/status checks, so an id-keyed cache cannot leak across tenants.
	if cached, ok := r.cacheGetByID(ctx, id); ok {
		return cached, nil
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

	r.cacheSet(ctx, &product)
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
	defer func() { _ = cursor.Close(ctx) }()

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
	defer func() { _ = cursor.Close(ctx) }()

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Search searches products by name, description, or tags.
//
// This uses a MongoDB $text query backed by the text index created in
// EnsureIndexes, so it uses an index instead of the previous full-collection
// case-insensitive $regex scan. NOTE: $text matches whole, stemmed words
// (case-insensitive) rather than arbitrary substrings — e.g. "shoe" matches
// "shoes" but "sho" no longer matches "shoes". $text is inherently safe from
// the ReDoS/pattern-injection concerns of the previous regex approach.
func (r *productRepository) Search(ctx context.Context, tenantID, query string, offset, limit int) ([]models.Product, int64, error) {
	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
		"$text":      bson.M{"$search": query},
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
	defer func() { _ = cursor.Close(ctx) }()

	var products []models.Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// Update updates a product. The tenant_id predicate ensures a tenant can only
// modify its own products; a cross-tenant id yields no match (404 upstream).
func (r *productRepository) Update(ctx context.Context, tenantID, id string, product *models.Product) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
	}

	product.UpdatedAt = time.Now()
	filter := bson.M{
		"_id":        objectID,
		"tenant_id":  tenantID,
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

	r.cacheInvalidate(ctx, id)
	return nil
}

// Delete soft deletes a product. The tenant_id predicate ensures a tenant can
// only delete its own products; a cross-tenant id yields no match (404 upstream).
func (r *productRepository) Delete(ctx context.Context, tenantID, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
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

	r.cacheInvalidate(ctx, id)
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

// UpdateStatus updates a product's status. The tenant_id predicate ensures a
// tenant can only change its own products' status; a cross-tenant id yields no
// match (404 upstream).
func (r *productRepository) UpdateStatus(ctx context.Context, tenantID, id string, status models.ProductStatus) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
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
		return errors.New("product not found")
	}

	r.cacheInvalidate(ctx, id)
	return nil
}

// UpdateStock updates the stock quantity and in_stock status for a product
func (r *productRepository) UpdateStock(ctx context.Context, tenantID, productID string, quantity int, inStock bool) error {
	filter := bson.M{
		"tenant_id":  tenantID,
		"deleted_at": bson.M{"$exists": false},
	}

	objectID, err := primitive.ObjectIDFromHex(productID)
	if err == nil {
		filter["_id"] = objectID
	} else {
		filter["sku"] = productID
	}

	update := bson.M{
		"$set": bson.M{
			"stock_quantity": quantity,
			"in_stock":       inStock,
			"updated_at":     time.Now(),
		},
	}

	if _, err = r.collection.UpdateOne(ctx, filter, update); err != nil {
		return err
	}
	// productID may be a hex id or a SKU; invalidating by it clears the entry
	// when it's the id (the common case) and is a harmless no-op otherwise.
	r.cacheInvalidate(ctx, productID)
	return nil
}

// AddImage atomically appends an image URL to a product's Images list.
func (r *productRepository) AddImage(ctx context.Context, id, imageURL string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
	}
	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$addToSet": bson.M{"images": imageURL},
		"$set":      bson.M{"updated_at": time.Now()},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("product not found")
	}
	r.cacheInvalidate(ctx, id)
	return nil
}

// RemoveImage atomically pulls an image URL from a product's Images list.
func (r *productRepository) RemoveImage(ctx context.Context, id, imageURL string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid product ID")
	}
	filter := bson.M{"_id": objectID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{
		"$pull": bson.M{"images": imageURL},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("product not found")
	}
	r.cacheInvalidate(ctx, id)
	return nil
}
