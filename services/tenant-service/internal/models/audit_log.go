package models

import (
	"time"
)

// AuditLog represents an audit trail entry
type AuditLog struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	TenantID  string    `gorm:"index" json:"tenant_id,omitempty"`
	UserID    string    `gorm:"index" json:"user_id,omitempty"`
	Action    string    `gorm:"not null;index" json:"action"`
	Resource  string    `gorm:"not null;index" json:"resource"`
	ResourceID string   `gorm:"index" json:"resource_id,omitempty"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`

	// Request/Response details
	RequestBody  string    `gorm:"type:text" json:"request_body,omitempty"`
	ResponseCode int       `json:"response_code"`

	// Old and new values for updates
	OldValue     string    `gorm:"type:text" json:"old_value,omitempty"`
	NewValue     string    `gorm:"type:text" json:"new_value,omitempty"`

	// Metadata
	Metadata     string    `gorm:"type:text" json:"metadata,omitempty"`
	ErrorMessage string    `gorm:"type:text" json:"error_message,omitempty"`
	Duration     int64     `json:"duration_ms"` // Request duration in milliseconds

	CreatedAt    time.Time `json:"created_at"`
}

// TableName specifies the table name for GORM
func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditAction represents different types of audit actions
type AuditAction string

const (
	ActionCreate AuditAction = "CREATE"
	ActionRead   AuditAction = "READ"
	ActionUpdate AuditAction = "UPDATE"
	ActionDelete AuditAction = "DELETE"
	ActionLogin  AuditAction = "LOGIN"
	ActionLogout AuditAction = "LOGOUT"
	ActionExport AuditAction = "EXPORT"
	ActionImport AuditAction = "IMPORT"
)

// AuditResource represents different resource types
type AuditResource string

const (
	ResourceTenant       AuditResource = "tenant"
	ResourceTenantConfig AuditResource = "tenant_config"
	ResourceUser         AuditResource = "user"
	ResourceProduct      AuditResource = "product"
	ResourceOrder        AuditResource = "order"
)

// CreateAuditLogRequest represents audit log creation request
type CreateAuditLogRequest struct {
	TenantID     string        `json:"tenant_id,omitempty"`
	UserID       string        `json:"user_id,omitempty"`
	Action       AuditAction   `json:"action"`
	Resource     AuditResource `json:"resource"`
	ResourceID   string        `json:"resource_id,omitempty"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	IPAddress    string        `json:"ip_address"`
	UserAgent    string        `json:"user_agent"`
	RequestBody  string        `json:"request_body,omitempty"`
	ResponseCode int           `json:"response_code"`
	OldValue     interface{}   `json:"old_value,omitempty"`
	NewValue     interface{}   `json:"new_value,omitempty"`
	Metadata     interface{}   `json:"metadata,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Duration     int64         `json:"duration_ms"`
}
