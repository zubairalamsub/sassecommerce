package repository

import (
	"context"
	"errors"
	"github.com/ecommerce/tenant-service/internal/models"
	"gorm.io/gorm"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id string) (*models.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*models.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*models.Tenant, error)
	List(ctx context.Context, page, pageSize int) ([]models.Tenant, int64, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type tenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{
		db: db,
	}
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *tenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("slug = ? AND deleted_at IS NULL", slug).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("domain = ? AND deleted_at IS NULL", domain).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *tenantRepository) List(ctx context.Context, page, pageSize int) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64

	offset := (page - 1) * pageSize

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tenants).Error

	if err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (r *tenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}

func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Tenant{}).Error
}

func (r *tenantRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}
