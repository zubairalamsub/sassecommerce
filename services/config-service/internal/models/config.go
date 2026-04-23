package models

import (
	"time"
)

// Value types for config entries
const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeJSON    = "json"
)

// === Database Models ===

// ConfigEntry represents a single configuration key-value pair
type ConfigEntry struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Namespace   string    `json:"namespace" gorm:"type:varchar(100);uniqueIndex:idx_ns_key_env_tenant;not null"`
	Key         string    `json:"key" gorm:"type:varchar(200);uniqueIndex:idx_ns_key_env_tenant;not null"`
	Value       string    `json:"value" gorm:"type:text;not null"`
	ValueType   string    `json:"value_type" gorm:"type:varchar(20);default:'string'"`
	Description string    `json:"description" gorm:"type:text"`
	Environment string    `json:"environment" gorm:"type:varchar(20);uniqueIndex:idx_ns_key_env_tenant;default:'all'"`
	TenantID    string    `json:"tenant_id" gorm:"type:varchar(36);uniqueIndex:idx_ns_key_env_tenant;default:''"`
	IsSecret    bool      `json:"is_secret" gorm:"default:false"`
	Version     int       `json:"version" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	UpdatedBy   string    `json:"updated_by" gorm:"type:varchar(100)"`
}

// ConfigAuditLog tracks all changes to configuration
type ConfigAuditLog struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ConfigID    string    `json:"config_id" gorm:"type:varchar(36);index"`
	Namespace   string    `json:"namespace" gorm:"type:varchar(100)"`
	Key         string    `json:"key" gorm:"type:varchar(200)"`
	OldValue    string    `json:"old_value" gorm:"type:text"`
	NewValue    string    `json:"new_value" gorm:"type:text"`
	Action      string    `json:"action" gorm:"type:varchar(20)"` // create, update, delete
	ChangedBy   string    `json:"changed_by" gorm:"type:varchar(100)"`
	Environment string    `json:"environment" gorm:"type:varchar(20)"`
	TenantID    string    `json:"tenant_id" gorm:"type:varchar(36)"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// === Request DTOs ===

type SetConfigRequest struct {
	Namespace   string `json:"namespace" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value" binding:"required"`
	ValueType   string `json:"value_type"`
	Description string `json:"description"`
	Environment string `json:"environment"`
	TenantID    string `json:"tenant_id"`
	IsSecret    bool   `json:"is_secret"`
	UpdatedBy   string `json:"updated_by"`
}

type BulkSetRequest struct {
	Entries   []SetConfigRequest `json:"entries" binding:"required"`
	UpdatedBy string             `json:"updated_by"`
}

type BulkGetRequest struct {
	Keys []NamespaceKey `json:"keys" binding:"required"`
}

type NamespaceKey struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

// === Response DTOs ===

type ConfigEntryResponse struct {
	ID          string    `json:"id"`
	Namespace   string    `json:"namespace"`
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	Description string    `json:"description,omitempty"`
	Environment string    `json:"environment"`
	TenantID    string    `json:"tenant_id,omitempty"`
	IsSecret    bool      `json:"is_secret"`
	Version     int       `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
}

type ConfigAuditResponse struct {
	ID          string    `json:"id"`
	ConfigID    string    `json:"config_id"`
	Namespace   string    `json:"namespace"`
	Key         string    `json:"key"`
	OldValue    string    `json:"old_value,omitempty"`
	NewValue    string    `json:"new_value"`
	Action      string    `json:"action"`
	ChangedBy   string    `json:"changed_by"`
	Environment string    `json:"environment"`
	TenantID    string    `json:"tenant_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type NamespaceListResponse struct {
	Namespaces []NamespaceSummary `json:"namespaces"`
}

type NamespaceSummary struct {
	Namespace string `json:"namespace"`
	Count     int64  `json:"count"`
}
