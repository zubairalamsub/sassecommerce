package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Category represents a product category
type Category struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID    string             `bson:"tenant_id" json:"tenant_id"`
	Name        string             `bson:"name" json:"name"`
	Slug        string             `bson:"slug" json:"slug"`
	Description string             `bson:"description" json:"description,omitempty"`
	ParentID    *string            `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
	Image       string             `bson:"image" json:"image,omitempty"`
	Icon        string             `bson:"icon" json:"icon,omitempty"`
	SortOrder   int                `bson:"sort_order" json:"sort_order"`
	Status      CategoryStatus     `bson:"status" json:"status"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	UpdatedBy   string             `bson:"updated_by" json:"updated_by,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// CategoryStatus represents the status of a category
type CategoryStatus string

const (
	CategoryStatusActive   CategoryStatus = "active"
	CategoryStatusInactive CategoryStatus = "inactive"
)

// CreateCategoryRequest represents a category creation request
type CreateCategoryRequest struct {
	TenantID    string `json:"tenant_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
	ParentID    *string `json:"parent_id"`
	Image       string `json:"image"`
	Icon        string `json:"icon"`
	SortOrder   int    `json:"sort_order"`
	CreatedBy   string `json:"created_by" binding:"required"`
}

// UpdateCategoryRequest represents a category update request
type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	ParentID    *string `json:"parent_id"`
	Image       *string `json:"image"`
	Icon        *string `json:"icon"`
	SortOrder   *int    `json:"sort_order"`
	UpdatedBy   string  `json:"updated_by" binding:"required"`
}

// CategoryResponse represents a category response
type CategoryResponse struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description,omitempty"`
	ParentID    *string        `json:"parent_id,omitempty"`
	Image       string         `json:"image,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	SortOrder   int            `json:"sort_order"`
	Status      CategoryStatus `json:"status"`
	CreatedBy   string         `json:"created_by"`
	UpdatedBy   string         `json:"updated_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ToResponse converts Category to CategoryResponse
func (c *Category) ToResponse() *CategoryResponse {
	return &CategoryResponse{
		ID:          c.ID.Hex(),
		TenantID:    c.TenantID,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		ParentID:    c.ParentID,
		Image:       c.Image,
		Icon:        c.Icon,
		SortOrder:   c.SortOrder,
		Status:      c.Status,
		CreatedBy:   c.CreatedBy,
		UpdatedBy:   c.UpdatedBy,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
