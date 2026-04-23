package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ecommerce/promotion-service/internal/models"
	"gorm.io/gorm"
)

// PromotionRepository defines the interface for promotion data access
type PromotionRepository interface {
	// Promotions
	CreatePromotion(ctx context.Context, promotion *models.Promotion) error
	GetPromotionByID(ctx context.Context, id string) (*models.Promotion, error)
	GetActivePromotions(ctx context.Context, tenantID string) ([]models.Promotion, error)
	UpdatePromotion(ctx context.Context, promotion *models.Promotion) error

	// Coupons
	CreateCoupon(ctx context.Context, coupon *models.Coupon) error
	GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error)
	GetCouponsByPromotion(ctx context.Context, promotionID string) ([]models.Coupon, error)
	UpdateCoupon(ctx context.Context, coupon *models.Coupon) error

	// Coupon Usage
	CreateCouponUsage(ctx context.Context, usage *models.CouponUsage) error
	GetUserCouponUsageCount(ctx context.Context, couponID, userID string) (int64, error)

	// Loyalty
	GetLoyaltyAccount(ctx context.Context, tenantID, userID string) (*models.LoyaltyAccount, error)
	CreateLoyaltyAccount(ctx context.Context, account *models.LoyaltyAccount) error
	UpdateLoyaltyAccount(ctx context.Context, account *models.LoyaltyAccount) error
	CreateLoyaltyTransaction(ctx context.Context, tx *models.LoyaltyTransaction) error
	GetLoyaltyTransactions(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.LoyaltyTransaction, int64, error)
}

type gormPromotionRepository struct {
	db *gorm.DB
}

// NewPromotionRepository creates a new GORM-backed promotion repository
func NewPromotionRepository(db *gorm.DB) PromotionRepository {
	return &gormPromotionRepository{db: db}
}

// --- Promotions ---

func (r *gormPromotionRepository) CreatePromotion(ctx context.Context, promotion *models.Promotion) error {
	return r.db.WithContext(ctx).Create(promotion).Error
}

func (r *gormPromotionRepository) GetPromotionByID(ctx context.Context, id string) (*models.Promotion, error) {
	var promotion models.Promotion
	if err := r.db.WithContext(ctx).First(&promotion, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("promotion not found")
		}
		return nil, err
	}
	return &promotion, nil
}

func (r *gormPromotionRepository) GetActivePromotions(ctx context.Context, tenantID string) ([]models.Promotion, error) {
	var promotions []models.Promotion
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ? AND start_date <= ? AND end_date >= ?",
			tenantID, models.StatusActive, now, now).
		Order("created_at DESC").
		Find(&promotions).Error
	return promotions, err
}

func (r *gormPromotionRepository) UpdatePromotion(ctx context.Context, promotion *models.Promotion) error {
	return r.db.WithContext(ctx).Save(promotion).Error
}

// --- Coupons ---

func (r *gormPromotionRepository) CreateCoupon(ctx context.Context, coupon *models.Coupon) error {
	return r.db.WithContext(ctx).Create(coupon).Error
}

func (r *gormPromotionRepository) GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error) {
	var coupon models.Coupon
	if err := r.db.WithContext(ctx).First(&coupon, "code = ?", code).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("coupon not found")
		}
		return nil, err
	}
	return &coupon, nil
}

func (r *gormPromotionRepository) GetCouponsByPromotion(ctx context.Context, promotionID string) ([]models.Coupon, error) {
	var coupons []models.Coupon
	err := r.db.WithContext(ctx).Where("promotion_id = ?", promotionID).Find(&coupons).Error
	return coupons, err
}

func (r *gormPromotionRepository) UpdateCoupon(ctx context.Context, coupon *models.Coupon) error {
	return r.db.WithContext(ctx).Save(coupon).Error
}

// --- Coupon Usage ---

func (r *gormPromotionRepository) CreateCouponUsage(ctx context.Context, usage *models.CouponUsage) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

func (r *gormPromotionRepository) GetUserCouponUsageCount(ctx context.Context, couponID, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.CouponUsage{}).
		Where("coupon_id = ? AND user_id = ?", couponID, userID).
		Count(&count).Error
	return count, err
}

// --- Loyalty ---

func (r *gormPromotionRepository) GetLoyaltyAccount(ctx context.Context, tenantID, userID string) (*models.LoyaltyAccount, error) {
	var account models.LoyaltyAccount
	if err := r.db.WithContext(ctx).First(&account, "tenant_id = ? AND user_id = ?", tenantID, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("loyalty account not found")
		}
		return nil, err
	}
	return &account, nil
}

func (r *gormPromotionRepository) CreateLoyaltyAccount(ctx context.Context, account *models.LoyaltyAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *gormPromotionRepository) UpdateLoyaltyAccount(ctx context.Context, account *models.LoyaltyAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

func (r *gormPromotionRepository) CreateLoyaltyTransaction(ctx context.Context, tx *models.LoyaltyTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *gormPromotionRepository) GetLoyaltyTransactions(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.LoyaltyTransaction, int64, error) {
	var transactions []models.LoyaltyTransaction
	var total int64

	r.db.WithContext(ctx).Model(&models.LoyaltyTransaction{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&transactions).Error

	return transactions, total, err
}
