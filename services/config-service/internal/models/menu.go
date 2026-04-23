package models

import "time"

// Menu locations
const (
	MenuLocationHeader  = "header"
	MenuLocationFooter  = "footer"
	MenuLocationSidebar = "sidebar"
	MenuLocationMobile  = "mobile"
)

// Menu link targets
const (
	TargetSelf  = "_self"
	TargetBlank = "_blank"
)

// === Database Models ===

// Menu represents a named menu container (e.g. "Main Navigation", "Footer Links")
type Menu struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID    string     `json:"tenant_id" gorm:"type:varchar(36);uniqueIndex:idx_menu_tenant_slug"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Slug        string     `json:"slug" gorm:"type:varchar(100);uniqueIndex:idx_menu_tenant_slug;not null"`
	Location    string     `json:"location" gorm:"type:varchar(50)"` // header, footer, sidebar, mobile
	Description string     `json:"description" gorm:"type:text"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	Items       []MenuItem `json:"items,omitempty" gorm:"foreignKey:MenuID;constraint:OnDelete:CASCADE"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// MenuItem represents a single link/node in a menu tree
type MenuItem struct {
	ID        string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	MenuID    string     `json:"menu_id" gorm:"type:varchar(36);index;not null"`
	ParentID  string     `json:"parent_id" gorm:"type:varchar(36);index;default:''"`
	Label     string     `json:"label" gorm:"type:varchar(200);not null"`
	URL       string     `json:"url" gorm:"type:varchar(500)"`
	Icon      string     `json:"icon" gorm:"type:varchar(100)"`
	Target    string     `json:"target" gorm:"type:varchar(20);default:'_self'"`
	CSSClass  string     `json:"css_class" gorm:"type:varchar(100)"`
	Position  int        `json:"position" gorm:"default:0"`
	IsActive  bool       `json:"is_active" gorm:"default:true"`
	Metadata  string     `json:"metadata" gorm:"type:text"` // JSON for custom attributes
	Children  []MenuItem `json:"children,omitempty" gorm:"-"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// === Request DTOs ===

type CreateMenuRequest struct {
	TenantID    string `json:"tenant_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Location    string `json:"location" binding:"required"`
	Description string `json:"description"`
}

type UpdateMenuRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type CreateMenuItemRequest struct {
	MenuID   string `json:"menu_id"`
	ParentID string `json:"parent_id"`
	Label    string `json:"label" binding:"required"`
	URL      string `json:"url"`
	Icon     string `json:"icon"`
	Target   string `json:"target"`
	CSSClass string `json:"css_class"`
	Position int    `json:"position"`
	Metadata string `json:"metadata"`
}

type UpdateMenuItemRequest struct {
	ParentID *string `json:"parent_id"`
	Label    string  `json:"label"`
	URL      string  `json:"url"`
	Icon     string  `json:"icon"`
	Target   string  `json:"target"`
	CSSClass string  `json:"css_class"`
	Position *int    `json:"position"`
	IsActive *bool   `json:"is_active"`
	Metadata string  `json:"metadata"`
}

type ReorderItemsRequest struct {
	Items []ReorderItem `json:"items" binding:"required"`
}

type ReorderItem struct {
	ID       string `json:"id" binding:"required"`
	Position int    `json:"position"`
	ParentID string `json:"parent_id"`
}

// === Response DTOs ===

type MenuResponse struct {
	ID          string             `json:"id"`
	TenantID    string             `json:"tenant_id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Location    string             `json:"location"`
	Description string             `json:"description,omitempty"`
	IsActive    bool               `json:"is_active"`
	Items       []MenuItemResponse `json:"items,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type MenuItemResponse struct {
	ID        string             `json:"id"`
	MenuID    string             `json:"menu_id"`
	ParentID  string             `json:"parent_id,omitempty"`
	Label     string             `json:"label"`
	URL       string             `json:"url,omitempty"`
	Icon      string             `json:"icon,omitempty"`
	Target    string             `json:"target"`
	CSSClass  string             `json:"css_class,omitempty"`
	Position  int                `json:"position"`
	IsActive  bool               `json:"is_active"`
	Metadata  string             `json:"metadata,omitempty"`
	Children  []MenuItemResponse `json:"children,omitempty"`
}
