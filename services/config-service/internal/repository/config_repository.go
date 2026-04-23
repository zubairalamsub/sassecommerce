package repository

import (
	"context"

	"github.com/ecommerce/config-service/internal/models"
	"gorm.io/gorm"
)

type ConfigRepository interface {
	// Config entries
	Get(ctx context.Context, namespace, key, environment, tenantID string) (*models.ConfigEntry, error)
	Set(ctx context.Context, entry *models.ConfigEntry) error
	Delete(ctx context.Context, id string) error
	ListByNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntry, error)
	ListNamespaces(ctx context.Context) ([]models.NamespaceSummary, error)
	GetByID(ctx context.Context, id string) (*models.ConfigEntry, error)
	BulkGet(ctx context.Context, keys []models.NamespaceKey, environment, tenantID string) ([]models.ConfigEntry, error)
	Search(ctx context.Context, query, namespace, environment string, page, pageSize int) ([]models.ConfigEntry, int64, error)

	// Audit log
	RecordAudit(ctx context.Context, log *models.ConfigAuditLog) error
	GetAuditLog(ctx context.Context, namespace, key string, page, pageSize int) ([]models.ConfigAuditLog, int64, error)
	GetAuditByConfigID(ctx context.Context, configID string, page, pageSize int) ([]models.ConfigAuditLog, int64, error)
}

type configRepository struct {
	db *gorm.DB
}

func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{db: db}
}

func (r *configRepository) Get(ctx context.Context, namespace, key, environment, tenantID string) (*models.ConfigEntry, error) {
	var entry models.ConfigEntry

	// Priority: tenant+env specific > env specific > global
	// Try tenant+env specific first
	if tenantID != "" {
		err := r.db.WithContext(ctx).
			Where("namespace = ? AND key = ? AND environment = ? AND tenant_id = ?", namespace, key, environment, tenantID).
			First(&entry).Error
		if err == nil {
			return &entry, nil
		}
	}

	// Try environment-specific
	if environment != "" && environment != "all" {
		err := r.db.WithContext(ctx).
			Where("namespace = ? AND key = ? AND environment = ? AND tenant_id = ''", namespace, key, environment).
			First(&entry).Error
		if err == nil {
			return &entry, nil
		}
	}

	// Fall back to global (environment = "all")
	err := r.db.WithContext(ctx).
		Where("namespace = ? AND key = ? AND environment = 'all' AND tenant_id = ''", namespace, key).
		First(&entry).Error
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *configRepository) Set(ctx context.Context, entry *models.ConfigEntry) error {
	return r.db.WithContext(ctx).Save(entry).Error
}

func (r *configRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ConfigEntry{}).Error
}

func (r *configRepository) GetByID(ctx context.Context, id string) (*models.ConfigEntry, error) {
	var entry models.ConfigEntry
	if err := r.db.WithContext(ctx).First(&entry, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *configRepository) ListByNamespace(ctx context.Context, namespace, environment, tenantID string) ([]models.ConfigEntry, error) {
	var entries []models.ConfigEntry
	query := r.db.WithContext(ctx).Where("namespace = ?", namespace)

	if environment != "" && environment != "all" {
		query = query.Where("environment IN ('all', ?)", environment)
	}
	if tenantID != "" {
		query = query.Where("tenant_id IN ('', ?)", tenantID)
	} else {
		query = query.Where("tenant_id = ''")
	}

	err := query.Order("key ASC").Find(&entries).Error
	return entries, err
}

func (r *configRepository) ListNamespaces(ctx context.Context) ([]models.NamespaceSummary, error) {
	var summaries []models.NamespaceSummary
	err := r.db.WithContext(ctx).Model(&models.ConfigEntry{}).
		Select("namespace, COUNT(*) as count").
		Group("namespace").
		Order("namespace ASC").
		Find(&summaries).Error
	return summaries, err
}

func (r *configRepository) BulkGet(ctx context.Context, keys []models.NamespaceKey, environment, tenantID string) ([]models.ConfigEntry, error) {
	var results []models.ConfigEntry

	for _, nk := range keys {
		entry, err := r.Get(ctx, nk.Namespace, nk.Key, environment, tenantID)
		if err == nil {
			results = append(results, *entry)
		}
	}

	return results, nil
}

func (r *configRepository) Search(ctx context.Context, query, namespace, environment string, page, pageSize int) ([]models.ConfigEntry, int64, error) {
	var entries []models.ConfigEntry
	var total int64

	dbQuery := r.db.WithContext(ctx).Model(&models.ConfigEntry{})

	if query != "" {
		pattern := "%" + query + "%"
		dbQuery = dbQuery.Where("key ILIKE ? OR description ILIKE ? OR value ILIKE ?", pattern, pattern, pattern)
	}
	if namespace != "" {
		dbQuery = dbQuery.Where("namespace = ?", namespace)
	}
	if environment != "" {
		dbQuery = dbQuery.Where("environment IN ('all', ?)", environment)
	}

	dbQuery.Count(&total)

	offset := (page - 1) * pageSize
	err := dbQuery.Order("namespace ASC, key ASC").Offset(offset).Limit(pageSize).Find(&entries).Error
	return entries, total, err
}

func (r *configRepository) RecordAudit(ctx context.Context, log *models.ConfigAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *configRepository) GetAuditLog(ctx context.Context, namespace, key string, page, pageSize int) ([]models.ConfigAuditLog, int64, error) {
	var logs []models.ConfigAuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ConfigAuditLog{})
	if namespace != "" {
		query = query.Where("namespace = ?", namespace)
	}
	if key != "" {
		query = query.Where("key = ?", key)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (r *configRepository) GetAuditByConfigID(ctx context.Context, configID string, page, pageSize int) ([]models.ConfigAuditLog, int64, error) {
	var logs []models.ConfigAuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ConfigAuditLog{}).Where("config_id = ?", configID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}
