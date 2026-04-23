package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantTier represents the tenant subscription tier
type TenantTier string

const (
	TierFree         TenantTier = "free"
	TierStarter      TenantTier = "starter"
	TierProfessional TenantTier = "professional"
	TierEnterprise   TenantTier = "enterprise"
)

// TenantStatus represents the current status of a tenant
type TenantStatus string

const (
	StatusActive    TenantStatus = "active"
	StatusSuspended TenantStatus = "suspended"
	StatusCancelled TenantStatus = "cancelled"
	StatusPending   TenantStatus = "pending"
)

// Tenant represents a multi-tenant business entity
type Tenant struct {
	ID          string       `gorm:"primaryKey" json:"id"`
	Name        string       `gorm:"not null" json:"name"`
	Slug        string       `gorm:"uniqueIndex;not null" json:"slug"`
	Domain      string       `json:"domain,omitempty"`
	Email       string       `gorm:"not null" json:"email"`
	Status      TenantStatus `gorm:"not null;default:pending" json:"status"`
	Tier        TenantTier   `gorm:"not null;default:free" json:"tier"`

	// Configuration
	Config      TenantConfig `gorm:"type:text" json:"config"`

	// Limits based on tier
	MaxUsers    int `gorm:"default:10" json:"max_users"`
	MaxProducts int `gorm:"default:100" json:"max_products"`
	MaxOrders   int `gorm:"default:1000" json:"max_orders"`

	// Database isolation strategy
	DatabaseStrategy string `gorm:"default:pool" json:"database_strategy"` // pool, bridge, silo
	DatabaseName     string `json:"database_name,omitempty"`

	// Billing
	StripeCustomerID     string     `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string     `json:"stripe_subscription_id,omitempty"`
	TrialEndsAt          *time.Time `json:"trial_ends_at,omitempty"`
	SubscriptionEndsAt   *time.Time `json:"subscription_ends_at,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// TenantConfig holds tenant-specific configuration
type TenantConfig struct {
	Branding BrandingConfig `json:"branding"`
	General  GeneralConfig  `json:"general"`
	Features FeatureConfig  `json:"features"`
}

// BrandingConfig holds branding settings
type BrandingConfig struct {
	LogoURL       string            `json:"logo_url"`
	FaviconURL    string            `json:"favicon_url"`
	PrimaryColor  string            `json:"primary_color"`
	SecondaryColor string           `json:"secondary_color"`
	CustomCSS     string            `json:"custom_css"`
	CustomFonts   map[string]string `json:"custom_fonts"`
}

// GeneralConfig holds general settings
type GeneralConfig struct {
	Timezone        string `json:"timezone"`
	Currency        string `json:"currency"`
	Language        string `json:"language"`
	DateFormat      string `json:"date_format"`
	TimeFormat      string `json:"time_format"`
	ContactEmail    string `json:"contact_email"`
	ContactPhone    string `json:"contact_phone"`
	SupportURL      string `json:"support_url"`
}

// FeatureConfig holds feature flags
type FeatureConfig struct {
	MultiCurrency      bool `json:"multi_currency"`
	Wishlist           bool `json:"wishlist"`
	ProductReviews     bool `json:"product_reviews"`
	GuestCheckout      bool `json:"guest_checkout"`
	SocialLogin        bool `json:"social_login"`
	AIRecommendations  bool `json:"ai_recommendations"`
	LoyaltyProgram     bool `json:"loyalty_program"`
	Subscriptions      bool `json:"subscriptions"`
	GiftCards          bool `json:"gift_cards"`
}

// CreateTenantRequest represents the request to create a new tenant
type CreateTenantRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Tier  string `json:"tier" binding:"required,oneof=free starter professional enterprise"`
}

// UpdateTenantRequest represents the request to update a tenant
type UpdateTenantRequest struct {
	Name   *string       `json:"name,omitempty"`
	Domain *string       `json:"domain,omitempty"`
	Status *string       `json:"status,omitempty"`
	Config *TenantConfig `json:"config,omitempty"`
}

// TenantResponse represents the tenant response
type TenantResponse struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Slug      string        `json:"slug"`
	Domain    string        `json:"domain,omitempty"`
	Email     string        `json:"email"`
	Status    TenantStatus  `json:"status"`
	Tier      TenantTier    `json:"tier"`
	Config    TenantConfig  `json:"config"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (Tenant) TableName() string {
	return "tenants"
}

// BeforeCreate hook to generate UUID and slug
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// Value implements driver.Valuer for database serialization
func (c TenantConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner for database deserialization
func (c *TenantConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("unsupported type for TenantConfig: %T", value)
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, c)
}
