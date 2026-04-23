package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/ecommerce/product-service/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// KafkaPublisher defines the interface for publishing messages to Kafka
type KafkaPublisher interface {
	Publish(ctx context.Context, topic, key string, value []byte) error
}

// ProductService defines the interface for product operations
type ProductService interface {
	CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.ProductResponse, error)
	GetProductByID(ctx context.Context, id string) (*models.ProductResponse, error)
	GetProductBySKU(ctx context.Context, tenantID, sku string) (*models.ProductResponse, error)
	ListProducts(ctx context.Context, tenantID string, offset, limit int) ([]models.ProductResponse, int64, error)
	ListProductsByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.ProductResponse, int64, error)
	SearchProducts(ctx context.Context, tenantID, query string, offset, limit int) ([]models.ProductResponse, int64, error)
	UpdateProduct(ctx context.Context, id string, req *models.UpdateProductRequest) (*models.ProductResponse, error)
	DeleteProduct(ctx context.Context, id string) error
	UpdateProductStatus(ctx context.Context, id string, status models.ProductStatus) error
}

type productService struct {
	productRepo   repository.ProductRepository
	categoryRepo  repository.CategoryRepository
	kafkaProducer KafkaPublisher
	logger        *logrus.Logger
}

// NewProductService creates a new product service
func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	kafkaProducer KafkaPublisher,
	logger *logrus.Logger,
) ProductService {
	return &productService{
		productRepo:   productRepo,
		categoryRepo:  categoryRepo,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

// CreateProduct creates a new product
func (s *productService) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.ProductResponse, error) {
	// Check if SKU already exists
	exists, err := s.productRepo.SKUExists(ctx, req.TenantID, req.SKU)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check SKU existence")
		return nil, errors.New("failed to check SKU availability")
	}
	if exists {
		return nil, errors.New("SKU already exists")
	}

	// Verify category exists
	_, err = s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		s.logger.WithError(err).Error("Category not found")
		return nil, errors.New("category not found")
	}

	// Generate variant IDs if not provided
	for i := range req.Variants {
		if req.Variants[i].ID == "" {
			req.Variants[i].ID = uuid.New().String()
		}
	}

	// Generate slug from name if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}

	product := &models.Product{
		TenantID:       req.TenantID,
		SKU:            req.SKU,
		Name:           req.Name,
		Slug:           slug,
		Description:    req.Description,
		CategoryID:     req.CategoryID,
		Brand:          req.Brand,
		Price:          req.Price,
		CompareAtPrice: req.CompareAtPrice,
		CostPerItem:    req.CostPerItem,
		Images:         req.Images,
		Tags:           req.Tags,
		Status:         parseStatus(req.Status),
		Variants:       req.Variants,
		Attributes:     req.Attributes,
		SEO:            req.SEO,
		Weight:         req.Weight,
		Dimensions:     req.Dimensions,
		CreatedBy:      req.CreatedBy,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		s.logger.WithError(err).Error("Failed to create product")
		return nil, errors.New("failed to create product")
	}

	s.logger.WithFields(logrus.Fields{
		"product_id": product.ID.Hex(),
		"tenant_id":  product.TenantID,
		"sku":        product.SKU,
	}).Info("Product created successfully")

	// Publish ProductCreated event
	s.publishEvent(ctx, "ProductCreated", map[string]interface{}{
		"tenant_id":  product.TenantID,
		"product_id": product.ID.Hex(),
		"name":       product.Name,
		"sku":        product.SKU,
		"price":      product.Price,
		"status":     product.Status,
	})

	return product.ToResponse(), nil
}

// GetProductByID retrieves a product by ID
func (s *productService) GetProductByID(ctx context.Context, id string) (*models.ProductResponse, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("product_id", id).Error("Failed to get product")
		return nil, errors.New("product not found")
	}

	return product.ToResponse(), nil
}

// GetProductBySKU retrieves a product by SKU
func (s *productService) GetProductBySKU(ctx context.Context, tenantID, sku string) (*models.ProductResponse, error) {
	product, err := s.productRepo.GetBySKU(ctx, tenantID, sku)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"sku":       sku,
		}).Error("Failed to get product by SKU")
		return nil, errors.New("product not found")
	}

	return product.ToResponse(), nil
}

// ListProducts retrieves products with pagination
func (s *productService) ListProducts(ctx context.Context, tenantID string, offset, limit int) ([]models.ProductResponse, int64, error) {
	products, total, err := s.productRepo.List(ctx, tenantID, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithField("tenant_id", tenantID).Error("Failed to list products")
		return nil, 0, errors.New("failed to retrieve products")
	}

	responses := make([]models.ProductResponse, len(products))
	for i, product := range products {
		responses[i] = *product.ToResponse()
	}

	return responses, total, nil
}

// ListProductsByCategory retrieves products by category with pagination
func (s *productService) ListProductsByCategory(ctx context.Context, tenantID, categoryID string, offset, limit int) ([]models.ProductResponse, int64, error) {
	products, total, err := s.productRepo.ListByCategory(ctx, tenantID, categoryID, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id":   tenantID,
			"category_id": categoryID,
		}).Error("Failed to list products by category")
		return nil, 0, errors.New("failed to retrieve products")
	}

	responses := make([]models.ProductResponse, len(products))
	for i, product := range products {
		responses[i] = *product.ToResponse()
	}

	return responses, total, nil
}

// SearchProducts searches products
func (s *productService) SearchProducts(ctx context.Context, tenantID, query string, offset, limit int) ([]models.ProductResponse, int64, error) {
	products, total, err := s.productRepo.Search(ctx, tenantID, query, offset, limit)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"query":     query,
		}).Error("Failed to search products")
		return nil, 0, errors.New("failed to search products")
	}

	responses := make([]models.ProductResponse, len(products))
	for i, product := range products {
		responses[i] = *product.ToResponse()
	}

	return responses, total, nil
}

// UpdateProduct updates a product
func (s *productService) UpdateProduct(ctx context.Context, id string, req *models.UpdateProductRequest) (*models.ProductResponse, error) {
	// Get existing product
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("product_id", id).Error("Failed to get product for update")
		return nil, errors.New("product not found")
	}

	// Update fields if provided
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.CategoryID != nil {
		// Verify category exists
		_, err := s.categoryRepo.GetByID(ctx, *req.CategoryID)
		if err != nil {
			return nil, errors.New("category not found")
		}
		product.CategoryID = *req.CategoryID
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.CompareAtPrice != nil {
		product.CompareAtPrice = *req.CompareAtPrice
	}
	if req.CostPerItem != nil {
		product.CostPerItem = *req.CostPerItem
	}
	if req.Images != nil {
		product.Images = *req.Images
	}
	if req.Tags != nil {
		product.Tags = *req.Tags
	}
	if req.Status != nil {
		product.Status = parseStatus(*req.Status)
	}
	if req.Variants != nil {
		product.Variants = *req.Variants
	}
	if req.Attributes != nil {
		product.Attributes = *req.Attributes
	}
	if req.SEO != nil {
		product.SEO = *req.SEO
	}
	if req.Weight != nil {
		product.Weight = *req.Weight
	}
	if req.Dimensions != nil {
		product.Dimensions = *req.Dimensions
	}
	if req.Name != nil {
		product.Slug = generateSlug(*req.Name)
	}
	product.UpdatedBy = req.UpdatedBy

	// Save changes
	if err := s.productRepo.Update(ctx, id, product); err != nil {
		s.logger.WithError(err).WithField("product_id", id).Error("Failed to update product")
		return nil, errors.New("failed to update product")
	}

	s.logger.WithField("product_id", id).Info("Product updated successfully")

	// Publish ProductUpdated event
	s.publishEvent(ctx, "ProductUpdated", map[string]interface{}{
		"tenant_id":  product.TenantID,
		"product_id": id,
		"name":       product.Name,
		"sku":        product.SKU,
		"price":      product.Price,
		"status":     product.Status,
	})

	return product.ToResponse(), nil
}

// DeleteProduct deletes a product (soft delete)
func (s *productService) DeleteProduct(ctx context.Context, id string) error {
	if err := s.productRepo.Delete(ctx, id); err != nil {
		s.logger.WithError(err).WithField("product_id", id).Error("Failed to delete product")
		return errors.New("failed to delete product")
	}

	s.logger.WithField("product_id", id).Info("Product deleted successfully")

	// Publish ProductDeleted event
	s.publishEvent(ctx, "ProductDeleted", map[string]interface{}{
		"product_id": id,
	})

	return nil
}

// UpdateProductStatus updates a product's status
func (s *productService) UpdateProductStatus(ctx context.Context, id string, status models.ProductStatus) error {
	if err := s.productRepo.UpdateStatus(ctx, id, status); err != nil {
		s.logger.WithError(err).WithField("product_id", id).Error("Failed to update product status")
		return errors.New("failed to update product status")
	}

	s.logger.WithFields(logrus.Fields{
		"product_id": id,
		"status":     status,
	}).Info("Product status updated successfully")

	return nil
}

func parseStatus(s string) models.ProductStatus {
	switch models.ProductStatus(s) {
	case models.ProductStatusActive, models.ProductStatusInactive, models.ProductStatusArchived:
		return models.ProductStatus(s)
	default:
		return models.ProductStatusDraft
	}
}

// publishEvent publishes an event to Kafka (non-blocking, logs warning on failure)
func (s *productService) publishEvent(ctx context.Context, eventType string, payload map[string]interface{}) {
	event := map[string]interface{}{
		"event_id":   uuid.New().String(),
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"version":    "1.0.0",
		"payload":    payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to marshal product event")
		return
	}

	if err := s.kafkaProducer.Publish(ctx, "product-events", event["event_id"].(string), data); err != nil {
		s.logger.WithError(err).Warn("Failed to publish product event")
	}
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9\s-]`)
var whitespace = regexp.MustCompile(`\s+`)
var multiDash = regexp.MustCompile(`-+`)

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = nonAlphaNum.ReplaceAllString(slug, "")
	slug = whitespace.ReplaceAllString(slug, "-")
	slug = multiDash.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}
