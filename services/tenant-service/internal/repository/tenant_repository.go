package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	tenantCacheByID     = "tenant:id:"
	tenantCacheBySlug   = "tenant:slug:"
	tenantCacheByDomain = "tenant:domain:"
	// Tenant records change rarely and are read on nearly every request, so a
	// slightly longer TTL is fine; writes invalidate explicitly.
	tenantCacheTTL = 5 * time.Minute
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
	// cache is optional: nil disables caching. All cache ops are best-effort —
	// any Redis error falls through to the database.
	cache *redis.Client
}

// NewTenantRepository creates a tenant repository. Pass a non-nil redis client
// to enable cache-aside on the GetByID/GetBySlug/GetByDomain lookups; nil
// disables caching.
func NewTenantRepository(db *gorm.DB, cache *redis.Client) TenantRepository {
	return &tenantRepository{
		db:    db,
		cache: cache,
	}
}

// cacheGet returns a cached tenant for a key, or (nil, false) on any miss/error.
func (r *tenantRepository) cacheGet(ctx context.Context, key string) (*models.Tenant, bool) {
	if r.cache == nil {
		return nil, false
	}
	data, err := r.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var t models.Tenant
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, false
	}
	return &t, true
}

// cacheStore writes a tenant under all three lookup keys. Best-effort.
func (r *tenantRepository) cacheStore(ctx context.Context, t *models.Tenant) {
	if r.cache == nil || t == nil {
		return
	}
	data, err := json.Marshal(t)
	if err != nil {
		return
	}
	pipe := r.cache.Pipeline()
	pipe.Set(ctx, tenantCacheByID+t.ID, data, tenantCacheTTL)
	if t.Slug != "" {
		pipe.Set(ctx, tenantCacheBySlug+t.Slug, data, tenantCacheTTL)
	}
	if t.Domain != "" {
		pipe.Set(ctx, tenantCacheByDomain+t.Domain, data, tenantCacheTTL)
	}
	_, _ = pipe.Exec(ctx)
}

// cacheInvalidate drops all cache entries for a tenant. Best-effort.
func (r *tenantRepository) cacheInvalidate(ctx context.Context, t *models.Tenant) {
	if r.cache == nil || t == nil {
		return
	}
	keys := []string{tenantCacheByID + t.ID}
	if t.Slug != "" {
		keys = append(keys, tenantCacheBySlug+t.Slug)
	}
	if t.Domain != "" {
		keys = append(keys, tenantCacheByDomain+t.Domain)
	}
	_ = r.cache.Del(ctx, keys...).Err()
}

func (r *tenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *tenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	if t, ok := r.cacheGet(ctx, tenantCacheByID+id); ok {
		return t, nil
	}
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	r.cacheStore(ctx, &tenant)
	return &tenant, nil
}

func (r *tenantRepository) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	if t, ok := r.cacheGet(ctx, tenantCacheBySlug+slug); ok {
		return t, nil
	}
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("slug = ? AND deleted_at IS NULL", slug).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	r.cacheStore(ctx, &tenant)
	return &tenant, nil
}

func (r *tenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	if t, ok := r.cacheGet(ctx, tenantCacheByDomain+domain); ok {
		return t, nil
	}
	var tenant models.Tenant
	err := r.db.WithContext(ctx).Where("domain = ? AND deleted_at IS NULL", domain).First(&tenant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tenant not found")
		}
		return nil, err
	}
	r.cacheStore(ctx, &tenant)
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
	if err := r.db.WithContext(ctx).Save(tenant).Error; err != nil {
		return err
	}
	// Refresh the cache with the new values. NOTE: if the slug/domain changed,
	// the entry under the OLD slug/domain is not dropped here (we no longer have
	// the old value) and expires via TTL — acceptable for a rare rename.
	r.cacheStore(ctx, tenant)
	return nil
}

func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	// Load first so we can drop the id/slug/domain keys together (best-effort).
	if existing, err := r.GetByID(ctx, id); err == nil {
		r.cacheInvalidate(ctx, existing)
	}
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Tenant{}).Error
}

func (r *tenantRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Tenant{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}
