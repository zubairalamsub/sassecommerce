package repository

import (
	"context"
	"encoding/json"

	"github.com/ecommerce/tenant-service/internal/models"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	GetByID(ctx context.Context, id string) (*models.AuditLog, error)
	List(ctx context.Context, filters AuditFilters) ([]models.AuditLog, int64, error)
}

type AuditFilters struct {
	TenantID   string
	UserID     string
	Action     string
	Resource   string
	ResourceID string
	StartDate  string
	EndDate    string
	Page       int
	PageSize   int
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{
		db: db,
	}
}

func (r *auditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *auditRepository) GetByID(ctx context.Context, id string) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *auditRepository) List(ctx context.Context, filters AuditFilters) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{})

	// Apply filters
	if filters.TenantID != "" {
		query = query.Where("tenant_id = ?", filters.TenantID)
	}
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if filters.ResourceID != "" {
		query = query.Where("resource_id = ?", filters.ResourceID)
	}
	if filters.StartDate != "" {
		query = query.Where("created_at >= ?", filters.StartDate)
	}
	if filters.EndDate != "" {
		query = query.Where("created_at <= ?", filters.EndDate)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (filters.Page - 1) * filters.PageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(filters.PageSize).
		Find(&logs).Error

	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// Helper function to convert data to JSON string
func ToJSONString(data interface{}) string {
	if data == nil {
		return ""
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
