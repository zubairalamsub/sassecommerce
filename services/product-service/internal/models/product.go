package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product represents a product in the catalog
type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID    string             `bson:"tenant_id" json:"tenant_id"`
	SKU         string             `bson:"sku" json:"sku"`
	Name        string             `bson:"name" json:"name"`
	Slug        string             `bson:"slug" json:"slug"`
	Description string             `bson:"description" json:"description"`
	CategoryID  string             `bson:"category_id" json:"category_id"`
	Brand       string             `bson:"brand" json:"brand,omitempty"`
	Price       float64            `bson:"price" json:"price"`
	CompareAtPrice float64         `bson:"compare_at_price" json:"compare_at_price,omitempty"`
	CostPerItem float64            `bson:"cost_per_item" json:"cost_per_item,omitempty"`
	Images      []string           `bson:"images" json:"images"`
	Tags        []string           `bson:"tags" json:"tags"`
	Status      ProductStatus      `bson:"status" json:"status"`
	Variants    []ProductVariant   `bson:"variants" json:"variants,omitempty"`
	Attributes  map[string]string  `bson:"attributes" json:"attributes,omitempty"`
	SEO         SEOMetadata        `bson:"seo" json:"seo,omitempty"`
	Weight      float64            `bson:"weight" json:"weight,omitempty"`
	Dimensions  Dimensions         `bson:"dimensions" json:"dimensions,omitempty"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	UpdatedBy   string             `bson:"updated_by" json:"updated_by,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// ProductStatus represents the status of a product
type ProductStatus string

const (
	ProductStatusDraft     ProductStatus = "draft"
	ProductStatusActive    ProductStatus = "active"
	ProductStatusInactive  ProductStatus = "inactive"
	ProductStatusArchived  ProductStatus = "archived"
)

// ProductVariant represents a product variant
type ProductVariant struct {
	ID       string             `bson:"id" json:"id"`
	SKU      string             `bson:"sku" json:"sku"`
	Name     string             `bson:"name" json:"name"`
	Price    float64            `bson:"price" json:"price"`
	Image    string             `bson:"image" json:"image,omitempty"`
	Options  map[string]string  `bson:"options" json:"options"` // e.g., {"Size": "L", "Color": "Red"}
	Weight   float64            `bson:"weight" json:"weight,omitempty"`
}

// SEOMetadata represents SEO information
type SEOMetadata struct {
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description" json:"description"`
	Keywords    []string `bson:"keywords" json:"keywords,omitempty"`
}

// Dimensions represents product dimensions
type Dimensions struct {
	Length float64 `bson:"length" json:"length"`
	Width  float64 `bson:"width" json:"width"`
	Height float64 `bson:"height" json:"height"`
	Unit   string  `bson:"unit" json:"unit"` // cm, in, etc.
}

// CreateProductRequest represents a product creation request
type CreateProductRequest struct {
	TenantID       string             `json:"tenant_id" binding:"required"`
	SKU            string             `json:"sku" binding:"required"`
	Name           string             `json:"name" binding:"required"`
	Slug           string             `json:"slug"`
	Description    string             `json:"description"`
	CategoryID     string             `json:"category_id" binding:"required"`
	Brand          string             `json:"brand"`
	Price          float64            `json:"price" binding:"required,gt=0"`
	CompareAtPrice float64            `json:"compare_at_price"`
	CostPerItem    float64            `json:"cost_per_item"`
	Images         []string           `json:"images"`
	Tags           []string           `json:"tags"`
	Status         string             `json:"status"`
	Variants       []ProductVariant   `json:"variants"`
	Attributes     map[string]string  `json:"attributes"`
	SEO            SEOMetadata        `json:"seo"`
	Weight         float64            `json:"weight"`
	Dimensions     Dimensions         `json:"dimensions"`
	CreatedBy      string             `json:"created_by" binding:"required"`
}

// UpdateProductRequest represents a product update request
type UpdateProductRequest struct {
	Name           *string            `json:"name"`
	Description    *string            `json:"description"`
	CategoryID     *string            `json:"category_id"`
	Brand          *string            `json:"brand"`
	Price          *float64           `json:"price"`
	CompareAtPrice *float64           `json:"compare_at_price"`
	CostPerItem    *float64           `json:"cost_per_item"`
	Images         *[]string          `json:"images"`
	Tags           *[]string          `json:"tags"`
	Status         *string            `json:"status"`
	Variants       *[]ProductVariant  `json:"variants"`
	Attributes     *map[string]string `json:"attributes"`
	SEO            *SEOMetadata       `json:"seo"`
	Weight         *float64           `json:"weight"`
	Dimensions     *Dimensions        `json:"dimensions"`
	UpdatedBy      string             `json:"updated_by" binding:"required"`
}

// ProductResponse represents a product response
type ProductResponse struct {
	ID             string             `json:"id"`
	TenantID       string             `json:"tenant_id"`
	SKU            string             `json:"sku"`
	Name           string             `json:"name"`
	Slug           string             `json:"slug"`
	Description    string             `json:"description"`
	CategoryID     string             `json:"category_id"`
	Brand          string             `json:"brand,omitempty"`
	Price          float64            `json:"price"`
	CompareAtPrice float64            `json:"compare_at_price,omitempty"`
	CostPerItem    float64            `json:"cost_per_item,omitempty"`
	Images         []string           `json:"images"`
	Tags           []string           `json:"tags"`
	Status         ProductStatus      `json:"status"`
	Variants       []ProductVariant   `json:"variants,omitempty"`
	Attributes     map[string]string  `json:"attributes,omitempty"`
	SEO            SEOMetadata        `json:"seo,omitempty"`
	Weight         float64            `json:"weight,omitempty"`
	Dimensions     Dimensions         `json:"dimensions,omitempty"`
	CreatedBy      string             `json:"created_by"`
	UpdatedBy      string             `json:"updated_by,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// ToResponse converts Product to ProductResponse
func (p *Product) ToResponse() *ProductResponse {
	return &ProductResponse{
		ID:             p.ID.Hex(),
		TenantID:       p.TenantID,
		SKU:            p.SKU,
		Name:           p.Name,
		Slug:           p.Slug,
		Description:    p.Description,
		CategoryID:     p.CategoryID,
		Brand:          p.Brand,
		Price:          p.Price,
		CompareAtPrice: p.CompareAtPrice,
		CostPerItem:    p.CostPerItem,
		Images:         p.Images,
		Tags:           p.Tags,
		Status:         p.Status,
		Variants:       p.Variants,
		Attributes:     p.Attributes,
		SEO:            p.SEO,
		Weight:         p.Weight,
		Dimensions:     p.Dimensions,
		CreatedBy:      p.CreatedBy,
		UpdatedBy:      p.UpdatedBy,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}
