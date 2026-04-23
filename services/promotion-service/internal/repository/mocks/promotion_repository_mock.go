package mocks

import (
	"context"

	"github.com/ecommerce/promotion-service/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockPromotionRepository struct {
	mock.Mock
}

func (m *MockPromotionRepository) CreatePromotion(ctx context.Context, promotion *models.Promotion) error {
	args := m.Called(ctx, promotion)
	return args.Error(0)
}

func (m *MockPromotionRepository) GetPromotionByID(ctx context.Context, id string) (*models.Promotion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Promotion), args.Error(1)
}

func (m *MockPromotionRepository) GetActivePromotions(ctx context.Context, tenantID string) ([]models.Promotion, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]models.Promotion), args.Error(1)
}

func (m *MockPromotionRepository) UpdatePromotion(ctx context.Context, promotion *models.Promotion) error {
	args := m.Called(ctx, promotion)
	return args.Error(0)
}

func (m *MockPromotionRepository) CreateCoupon(ctx context.Context, coupon *models.Coupon) error {
	args := m.Called(ctx, coupon)
	return args.Error(0)
}

func (m *MockPromotionRepository) GetCouponByCode(ctx context.Context, code string) (*models.Coupon, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Coupon), args.Error(1)
}

func (m *MockPromotionRepository) GetCouponsByPromotion(ctx context.Context, promotionID string) ([]models.Coupon, error) {
	args := m.Called(ctx, promotionID)
	return args.Get(0).([]models.Coupon), args.Error(1)
}

func (m *MockPromotionRepository) UpdateCoupon(ctx context.Context, coupon *models.Coupon) error {
	args := m.Called(ctx, coupon)
	return args.Error(0)
}

func (m *MockPromotionRepository) CreateCouponUsage(ctx context.Context, usage *models.CouponUsage) error {
	args := m.Called(ctx, usage)
	return args.Error(0)
}

func (m *MockPromotionRepository) GetUserCouponUsageCount(ctx context.Context, couponID, userID string) (int64, error) {
	args := m.Called(ctx, couponID, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPromotionRepository) GetLoyaltyAccount(ctx context.Context, tenantID, userID string) (*models.LoyaltyAccount, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LoyaltyAccount), args.Error(1)
}

func (m *MockPromotionRepository) CreateLoyaltyAccount(ctx context.Context, account *models.LoyaltyAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockPromotionRepository) UpdateLoyaltyAccount(ctx context.Context, account *models.LoyaltyAccount) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockPromotionRepository) CreateLoyaltyTransaction(ctx context.Context, tx *models.LoyaltyTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockPromotionRepository) GetLoyaltyTransactions(ctx context.Context, tenantID, userID string, page, pageSize int) ([]models.LoyaltyTransaction, int64, error) {
	args := m.Called(ctx, tenantID, userID, page, pageSize)
	return args.Get(0).([]models.LoyaltyTransaction), args.Get(1).(int64), args.Error(2)
}
