package repository

import (
	"context"
	"errors"

	"github.com/ecommerce/user-service/internal/models"
	"gorm.io/gorm"
)

// WishlistRepository defines data operations for wishlist items
type WishlistRepository interface {
	List(ctx context.Context, userID, tenantID string) ([]models.WishlistItem, error)
	Add(ctx context.Context, item *models.WishlistItem) error
	Remove(ctx context.Context, userID, tenantID, productID string) error
	Exists(ctx context.Context, userID, tenantID, productID string) (bool, error)
	Clear(ctx context.Context, userID, tenantID string) error
}

type wishlistRepository struct {
	db *gorm.DB
}

// NewWishlistRepository creates a new wishlist repository
func NewWishlistRepository(db *gorm.DB) WishlistRepository {
	return &wishlistRepository{db: db}
}

func (r *wishlistRepository) List(ctx context.Context, userID, tenantID string) ([]models.WishlistItem, error) {
	var items []models.WishlistItem
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Order("added_at DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *wishlistRepository) Add(ctx context.Context, item *models.WishlistItem) error {
	exists, err := r.Exists(ctx, item.UserID, item.TenantID, item.ProductID)
	if err != nil {
		return err
	}
	if exists {
		return nil // idempotent — already in wishlist
	}
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *wishlistRepository) Remove(ctx context.Context, userID, tenantID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ? AND product_id = ?", userID, tenantID, productID).
		Delete(&models.WishlistItem{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("wishlist item not found")
	}
	return nil
}

func (r *wishlistRepository) Exists(ctx context.Context, userID, tenantID, productID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.WishlistItem{}).
		Where("user_id = ? AND tenant_id = ? AND product_id = ?", userID, tenantID, productID).
		Count(&count).Error
	return count > 0, err
}

func (r *wishlistRepository) Clear(ctx context.Context, userID, tenantID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&models.WishlistItem{}).Error
}
