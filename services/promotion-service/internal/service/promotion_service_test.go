package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ecommerce/promotion-service/internal/models"
	repoMocks "github.com/ecommerce/promotion-service/internal/repository/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestService() (*promotionService, *repoMocks.MockPromotionRepository) {
	mockRepo := new(repoMocks.MockPromotionRepository)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := &promotionService{
		repo:   mockRepo,
		writer: nil,
		logger: logger,
	}

	return svc, mockRepo
}

func createTestPromotion() *models.Promotion {
	return &models.Promotion{
		ID:             "promo-1",
		TenantID:       "tenant-1",
		Name:           "Summer Sale",
		Description:    "20% off everything",
		DiscountType:   models.DiscountPercentage,
		DiscountValue:  20,
		MinOrderAmount: 50,
		MaxDiscount:    100,
		Status:         models.StatusActive,
		StartDate:      time.Now().UTC().Add(-24 * time.Hour),
		EndDate:        time.Now().UTC().Add(7 * 24 * time.Hour),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

func createTestCoupon() *models.Coupon {
	return &models.Coupon{
		ID:             "coupon-1",
		TenantID:       "tenant-1",
		PromotionID:    "promo-1",
		Code:           "SUMMER20",
		MaxUses:        100,
		UsedCount:      5,
		MaxUsesPerUser: 1,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
}

// === CreatePromotion Tests ===

func TestCreatePromotion_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreatePromotion", ctx, mock.AnythingOfType("*models.Promotion")).Return(nil)

	req := &models.CreatePromotionRequest{
		TenantID:       "tenant-1",
		Name:           "Summer Sale",
		Description:    "20% off",
		DiscountType:   models.DiscountPercentage,
		DiscountValue:  20,
		MinOrderAmount: 50,
		MaxDiscount:    100,
		StartDate:      time.Now().UTC().Add(-1 * time.Hour),
		EndDate:        time.Now().UTC().Add(7 * 24 * time.Hour),
	}

	result, err := svc.CreatePromotion(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Summer Sale", result.Name)
	assert.Equal(t, models.StatusActive, result.Status)
}

func TestCreatePromotion_FutureStart(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreatePromotion", ctx, mock.AnythingOfType("*models.Promotion")).Return(nil)

	req := &models.CreatePromotionRequest{
		TenantID:      "tenant-1",
		Name:          "Future Sale",
		DiscountType:  models.DiscountFixedAmount,
		DiscountValue: 10,
		StartDate:     time.Now().UTC().Add(24 * time.Hour),
		EndDate:       time.Now().UTC().Add(48 * time.Hour),
	}

	result, err := svc.CreatePromotion(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, models.StatusDraft, result.Status)
}

func TestCreatePromotion_EndBeforeStart(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.CreatePromotionRequest{
		TenantID:      "tenant-1",
		Name:          "Bad Sale",
		DiscountType:  models.DiscountPercentage,
		DiscountValue: 10,
		StartDate:     time.Now().UTC().Add(48 * time.Hour),
		EndDate:       time.Now().UTC().Add(24 * time.Hour),
	}

	result, err := svc.CreatePromotion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "end date must be after start date")
}

func TestCreatePromotion_PercentageOver100(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	req := &models.CreatePromotionRequest{
		TenantID:      "tenant-1",
		Name:          "Bad Discount",
		DiscountType:  models.DiscountPercentage,
		DiscountValue: 150,
		StartDate:     time.Now().UTC(),
		EndDate:       time.Now().UTC().Add(24 * time.Hour),
	}

	result, err := svc.CreatePromotion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "percentage discount cannot exceed 100")
}

func TestCreatePromotion_RepoFailure(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("CreatePromotion", ctx, mock.AnythingOfType("*models.Promotion")).Return(errors.New("db error"))

	req := &models.CreatePromotionRequest{
		TenantID:      "tenant-1",
		Name:          "Sale",
		DiscountType:  models.DiscountPercentage,
		DiscountValue: 10,
		StartDate:     time.Now().UTC(),
		EndDate:       time.Now().UTC().Add(24 * time.Hour),
	}

	result, err := svc.CreatePromotion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetPromotion Tests ===

func TestGetPromotion_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	promo := createTestPromotion()
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)

	result, err := svc.GetPromotion(ctx, "promo-1")

	assert.NoError(t, err)
	assert.Equal(t, "promo-1", result.ID)
	assert.Equal(t, "Summer Sale", result.Name)
}

func TestGetPromotion_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetPromotionByID", ctx, "bad").Return(nil, errors.New("promotion not found"))

	result, err := svc.GetPromotion(ctx, "bad")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// === GetActivePromotions Tests ===

func TestGetActivePromotions_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	promos := []models.Promotion{*createTestPromotion()}
	mockRepo.On("GetActivePromotions", ctx, "tenant-1").Return(promos, nil)

	results, err := svc.GetActivePromotions(ctx, "tenant-1")

	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestGetActivePromotions_Empty(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetActivePromotions", ctx, "tenant-1").Return([]models.Promotion{}, nil)

	results, err := svc.GetActivePromotions(ctx, "tenant-1")

	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

// === CreateCoupon Tests ===

func TestCreateCoupon_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	promo := createTestPromotion()
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)
	mockRepo.On("CreateCoupon", ctx, mock.AnythingOfType("*models.Coupon")).Return(nil)

	req := &models.CreateCouponRequest{
		TenantID:    "tenant-1",
		PromotionID: "promo-1",
		Code:        "summer20",
		MaxUses:     100,
	}

	result, err := svc.CreateCoupon(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, "SUMMER20", result.Code) // uppercased
	assert.Equal(t, models.DiscountPercentage, result.DiscountType)
	assert.Equal(t, 20.0, result.DiscountValue)
}

func TestCreateCoupon_PromotionNotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetPromotionByID", ctx, "bad").Return(nil, errors.New("not found"))

	req := &models.CreateCouponRequest{
		TenantID:    "tenant-1",
		PromotionID: "bad",
		Code:        "TEST",
	}

	result, err := svc.CreateCoupon(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCreateCoupon_DuplicateCode(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	promo := createTestPromotion()
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)
	mockRepo.On("CreateCoupon", ctx, mock.AnythingOfType("*models.Coupon")).Return(errors.New("duplicate key"))

	req := &models.CreateCouponRequest{
		TenantID:    "tenant-1",
		PromotionID: "promo-1",
		Code:        "SUMMER20",
	}

	result, err := svc.CreateCoupon(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "coupon code already exists")
}

// === ValidateCoupon Tests ===

func TestValidateCoupon_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	promo := createTestPromotion()

	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)
	mockRepo.On("GetUserCouponUsageCount", ctx, "coupon-1", "user-1").Return(int64(0), nil)
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)

	req := &models.ValidateCouponRequest{
		TenantID:   "tenant-1",
		UserID:     "user-1",
		OrderTotal: 200.0,
	}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, 40.0, result.DiscountAmount) // 20% of 200
}

func TestValidateCoupon_NotFound(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCouponByCode", ctx, "BAD").Return(nil, errors.New("coupon not found"))

	req := &models.ValidateCouponRequest{TenantID: "tenant-1", UserID: "user-1", OrderTotal: 100}

	result, err := svc.ValidateCoupon(ctx, "BAD", req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "coupon not found")
}

func TestValidateCoupon_Inactive(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	coupon.IsActive = false
	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)

	req := &models.ValidateCouponRequest{TenantID: "tenant-1", UserID: "user-1", OrderTotal: 100}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "inactive")
}

func TestValidateCoupon_WrongTenant(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)

	req := &models.ValidateCouponRequest{TenantID: "tenant-2", UserID: "user-1", OrderTotal: 100}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "not valid for this tenant")
}

func TestValidateCoupon_UsageLimitReached(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	coupon.UsedCount = 100
	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)

	req := &models.ValidateCouponRequest{TenantID: "tenant-1", UserID: "user-1", OrderTotal: 100}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "usage limit reached")
}

func TestValidateCoupon_AlreadyUsedByUser(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)
	mockRepo.On("GetUserCouponUsageCount", ctx, "coupon-1", "user-1").Return(int64(1), nil)

	req := &models.ValidateCouponRequest{TenantID: "tenant-1", UserID: "user-1", OrderTotal: 100}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "already used")
}

func TestValidateCoupon_BelowMinOrder(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	promo := createTestPromotion()
	promo.MinOrderAmount = 100

	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)
	mockRepo.On("GetUserCouponUsageCount", ctx, "coupon-1", "user-1").Return(int64(0), nil)
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)

	req := &models.ValidateCouponRequest{TenantID: "tenant-1", UserID: "user-1", OrderTotal: 30}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Message, "minimum order amount")
}

func TestValidateCoupon_MaxDiscountCapped(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	promo := createTestPromotion()
	promo.MaxDiscount = 30

	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)
	mockRepo.On("GetUserCouponUsageCount", ctx, "coupon-1", "user-1").Return(int64(0), nil)
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)

	req := &models.ValidateCouponRequest{TenantID: "tenant-1", UserID: "user-1", OrderTotal: 200}

	result, err := svc.ValidateCoupon(ctx, "SUMMER20", req)

	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, 30.0, result.DiscountAmount) // capped at max_discount
}

// === ApplyCoupon Tests ===

func TestApplyCoupon_Success(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	coupon := createTestCoupon()
	promo := createTestPromotion()

	mockRepo.On("GetCouponByCode", ctx, "SUMMER20").Return(coupon, nil)
	mockRepo.On("GetUserCouponUsageCount", ctx, "coupon-1", "user-1").Return(int64(0), nil)
	mockRepo.On("GetPromotionByID", ctx, "promo-1").Return(promo, nil)
	mockRepo.On("CreateCouponUsage", ctx, mock.AnythingOfType("*models.CouponUsage")).Return(nil)
	mockRepo.On("UpdateCoupon", ctx, mock.AnythingOfType("*models.Coupon")).Return(nil)

	req := &models.ApplyCouponRequest{
		TenantID:   "tenant-1",
		UserID:     "user-1",
		OrderID:    "order-1",
		OrderTotal: 200.0,
		Code:       "summer20",
	}

	result, err := svc.ApplyCoupon(ctx, req)

	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, 40.0, result.DiscountAmount)
}

func TestApplyCoupon_InvalidCoupon(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetCouponByCode", ctx, "BAD").Return(nil, errors.New("coupon not found"))

	req := &models.ApplyCouponRequest{
		TenantID: "tenant-1", UserID: "user-1", OrderID: "order-1",
		OrderTotal: 100, Code: "BAD",
	}

	result, err := svc.ApplyCoupon(ctx, req)

	assert.NoError(t, err)
	assert.False(t, result.Valid)
}

// === Loyalty Tests ===

func TestGetLoyaltyAccount_Existing(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	account := &models.LoyaltyAccount{
		ID: "acc-1", TenantID: "tenant-1", UserID: "user-1",
		Points: 500, TierLevel: "bronze",
	}
	mockRepo.On("GetLoyaltyAccount", ctx, "tenant-1", "user-1").Return(account, nil)

	result, err := svc.GetLoyaltyAccount(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 500, result.Points)
	assert.Equal(t, "bronze", result.TierLevel)
}

func TestGetLoyaltyAccount_CreateNew(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetLoyaltyAccount", ctx, "tenant-1", "user-1").Return(nil, errors.New("loyalty account not found"))
	mockRepo.On("CreateLoyaltyAccount", ctx, mock.AnythingOfType("*models.LoyaltyAccount")).Return(nil)

	result, err := svc.GetLoyaltyAccount(ctx, "tenant-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Points)
	assert.Equal(t, "bronze", result.TierLevel)
}

func TestProcessLoyaltyPoints_Earn(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	account := &models.LoyaltyAccount{
		ID: "acc-1", TenantID: "tenant-1", UserID: "user-1",
		Points: 500, TierLevel: "bronze",
	}
	mockRepo.On("GetLoyaltyAccount", ctx, "tenant-1", "user-1").Return(account, nil)
	mockRepo.On("UpdateLoyaltyAccount", ctx, mock.AnythingOfType("*models.LoyaltyAccount")).Return(nil)
	mockRepo.On("CreateLoyaltyTransaction", ctx, mock.AnythingOfType("*models.LoyaltyTransaction")).Return(nil)

	req := &models.LoyaltyPointsRequest{
		TenantID: "tenant-1", UserID: "user-1",
		Type: models.TransactionEarn, Points: 600,
		OrderID: "order-1", Description: "Order purchase",
	}

	result, err := svc.ProcessLoyaltyPoints(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 1100, result.Points) // 500 + 600
	assert.Equal(t, "silver", result.TierLevel)
}

func TestProcessLoyaltyPoints_Redeem(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	account := &models.LoyaltyAccount{
		ID: "acc-1", TenantID: "tenant-1", UserID: "user-1",
		Points: 500, TierLevel: "bronze",
	}
	mockRepo.On("GetLoyaltyAccount", ctx, "tenant-1", "user-1").Return(account, nil)
	mockRepo.On("UpdateLoyaltyAccount", ctx, mock.AnythingOfType("*models.LoyaltyAccount")).Return(nil)
	mockRepo.On("CreateLoyaltyTransaction", ctx, mock.AnythingOfType("*models.LoyaltyTransaction")).Return(nil)

	req := &models.LoyaltyPointsRequest{
		TenantID: "tenant-1", UserID: "user-1",
		Type: models.TransactionRedeem, Points: 200,
	}

	result, err := svc.ProcessLoyaltyPoints(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 300, result.Points) // 500 - 200
}

func TestProcessLoyaltyPoints_InsufficientPoints(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	account := &models.LoyaltyAccount{
		ID: "acc-1", TenantID: "tenant-1", UserID: "user-1",
		Points: 100, TierLevel: "bronze",
	}
	mockRepo.On("GetLoyaltyAccount", ctx, "tenant-1", "user-1").Return(account, nil)

	req := &models.LoyaltyPointsRequest{
		TenantID: "tenant-1", UserID: "user-1",
		Type: models.TransactionRedeem, Points: 500,
	}

	result, err := svc.ProcessLoyaltyPoints(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "insufficient loyalty points")
}

func TestProcessLoyaltyPoints_NewAccount(t *testing.T) {
	svc, mockRepo := newTestService()
	ctx := context.Background()

	mockRepo.On("GetLoyaltyAccount", ctx, "tenant-1", "user-1").Return(nil, errors.New("loyalty account not found"))
	mockRepo.On("CreateLoyaltyAccount", ctx, mock.AnythingOfType("*models.LoyaltyAccount")).Return(nil)
	mockRepo.On("UpdateLoyaltyAccount", ctx, mock.AnythingOfType("*models.LoyaltyAccount")).Return(nil)
	mockRepo.On("CreateLoyaltyTransaction", ctx, mock.AnythingOfType("*models.LoyaltyTransaction")).Return(nil)

	req := &models.LoyaltyPointsRequest{
		TenantID: "tenant-1", UserID: "user-1",
		Type: models.TransactionEarn, Points: 100,
	}

	result, err := svc.ProcessLoyaltyPoints(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, 100, result.Points)
}

// === calculateDiscount Tests ===

func TestCalculateDiscount_Percentage(t *testing.T) {
	promo := &models.Promotion{DiscountType: models.DiscountPercentage, DiscountValue: 20}
	assert.Equal(t, 20.0, calculateDiscount(promo, 100))
}

func TestCalculateDiscount_FixedAmount(t *testing.T) {
	promo := &models.Promotion{DiscountType: models.DiscountFixedAmount, DiscountValue: 15}
	assert.Equal(t, 15.0, calculateDiscount(promo, 100))
}

func TestCalculateDiscount_FixedAmountExceedsTotal(t *testing.T) {
	promo := &models.Promotion{DiscountType: models.DiscountFixedAmount, DiscountValue: 150}
	assert.Equal(t, 100.0, calculateDiscount(promo, 100)) // capped at order total
}

func TestCalculateDiscount_MaxDiscountCap(t *testing.T) {
	promo := &models.Promotion{
		DiscountType: models.DiscountPercentage, DiscountValue: 50,
		MaxDiscount: 30,
	}
	assert.Equal(t, 30.0, calculateDiscount(promo, 100)) // 50% = 50, capped at 30
}

func TestCalculateDiscount_FreeShipping(t *testing.T) {
	promo := &models.Promotion{DiscountType: models.DiscountFreeShipping}
	assert.Equal(t, 0.0, calculateDiscount(promo, 100))
}

// === calculateTier Tests ===

func TestCalculateTier(t *testing.T) {
	assert.Equal(t, "bronze", calculateTier(0))
	assert.Equal(t, "bronze", calculateTier(999))
	assert.Equal(t, "silver", calculateTier(1000))
	assert.Equal(t, "silver", calculateTier(4999))
	assert.Equal(t, "gold", calculateTier(5000))
	assert.Equal(t, "gold", calculateTier(9999))
	assert.Equal(t, "platinum", calculateTier(10000))
	assert.Equal(t, "platinum", calculateTier(50000))
}
