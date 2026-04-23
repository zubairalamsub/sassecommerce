package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ecommerce/promotion-service/internal/models"
	"github.com/ecommerce/promotion-service/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// PromotionService defines the interface for promotion business logic
type PromotionService interface {
	// Promotions
	CreatePromotion(ctx context.Context, req *models.CreatePromotionRequest) (*models.PromotionResponse, error)
	GetPromotion(ctx context.Context, id string) (*models.PromotionResponse, error)
	GetActivePromotions(ctx context.Context, tenantID string) ([]models.PromotionResponse, error)

	// Coupons
	CreateCoupon(ctx context.Context, req *models.CreateCouponRequest) (*models.CouponResponse, error)
	ValidateCoupon(ctx context.Context, code string, req *models.ValidateCouponRequest) (*models.ValidateCouponResponse, error)
	ApplyCoupon(ctx context.Context, req *models.ApplyCouponRequest) (*models.ValidateCouponResponse, error)

	// Loyalty
	GetLoyaltyAccount(ctx context.Context, tenantID, userID string) (*models.LoyaltyAccountResponse, error)
	ProcessLoyaltyPoints(ctx context.Context, req *models.LoyaltyPointsRequest) (*models.LoyaltyAccountResponse, error)
}

type promotionService struct {
	repo   repository.PromotionRepository
	writer *kafka.Writer
	logger *logrus.Logger
}

// NewPromotionService creates a new PromotionService instance
func NewPromotionService(repo repository.PromotionRepository, writer *kafka.Writer, logger *logrus.Logger) PromotionService {
	return &promotionService{
		repo:   repo,
		writer: writer,
		logger: logger,
	}
}

// --- Promotions ---

func (s *promotionService) CreatePromotion(ctx context.Context, req *models.CreatePromotionRequest) (*models.PromotionResponse, error) {
	if req.EndDate.Before(req.StartDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	if req.DiscountType == models.DiscountPercentage && req.DiscountValue > 100 {
		return nil, fmt.Errorf("percentage discount cannot exceed 100")
	}

	now := time.Now().UTC()
	status := models.StatusDraft
	if now.After(req.StartDate) && now.Before(req.EndDate) {
		status = models.StatusActive
	}

	promotion := &models.Promotion{
		ID:             uuid.New().String(),
		TenantID:       req.TenantID,
		Name:           req.Name,
		Description:    req.Description,
		DiscountType:   req.DiscountType,
		DiscountValue:  req.DiscountValue,
		MinOrderAmount: req.MinOrderAmount,
		MaxDiscount:    req.MaxDiscount,
		Status:         status,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreatePromotion(ctx, promotion); err != nil {
		return nil, fmt.Errorf("failed to create promotion: %w", err)
	}

	s.publishEvent("PromotionCreated", promotion.TenantID, map[string]interface{}{
		"promotion_id":  promotion.ID,
		"tenant_id":     promotion.TenantID,
		"name":          promotion.Name,
		"discount_type": promotion.DiscountType,
		"discount_value": promotion.DiscountValue,
	})

	return toPromotionResponse(promotion), nil
}

func (s *promotionService) GetPromotion(ctx context.Context, id string) (*models.PromotionResponse, error) {
	promotion, err := s.repo.GetPromotionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPromotionResponse(promotion), nil
}

func (s *promotionService) GetActivePromotions(ctx context.Context, tenantID string) ([]models.PromotionResponse, error) {
	promotions, err := s.repo.GetActivePromotions(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active promotions: %w", err)
	}

	responses := make([]models.PromotionResponse, len(promotions))
	for i, p := range promotions {
		responses[i] = *toPromotionResponse(&p)
	}
	return responses, nil
}

// --- Coupons ---

func (s *promotionService) CreateCoupon(ctx context.Context, req *models.CreateCouponRequest) (*models.CouponResponse, error) {
	// Verify promotion exists
	promotion, err := s.repo.GetPromotionByID(ctx, req.PromotionID)
	if err != nil {
		return nil, fmt.Errorf("promotion not found")
	}

	code := strings.ToUpper(req.Code)
	maxUsesPerUser := req.MaxUsesPerUser
	if maxUsesPerUser <= 0 {
		maxUsesPerUser = 1
	}

	coupon := &models.Coupon{
		ID:             uuid.New().String(),
		TenantID:       req.TenantID,
		PromotionID:    req.PromotionID,
		Code:           code,
		MaxUses:        req.MaxUses,
		UsedCount:      0,
		MaxUsesPerUser: maxUsesPerUser,
		IsActive:       true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.CreateCoupon(ctx, coupon); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("coupon code already exists")
		}
		return nil, fmt.Errorf("failed to create coupon: %w", err)
	}

	return &models.CouponResponse{
		ID:             coupon.ID,
		TenantID:       coupon.TenantID,
		PromotionID:    coupon.PromotionID,
		Code:           coupon.Code,
		MaxUses:        coupon.MaxUses,
		UsedCount:      coupon.UsedCount,
		MaxUsesPerUser: coupon.MaxUsesPerUser,
		IsActive:       coupon.IsActive,
		DiscountType:   promotion.DiscountType,
		DiscountValue:  promotion.DiscountValue,
	}, nil
}

func (s *promotionService) ValidateCoupon(ctx context.Context, code string, req *models.ValidateCouponRequest) (*models.ValidateCouponResponse, error) {
	code = strings.ToUpper(code)

	coupon, err := s.repo.GetCouponByCode(ctx, code)
	if err != nil {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "coupon not found"}, nil
	}

	if !coupon.IsActive {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "coupon is inactive"}, nil
	}

	if coupon.TenantID != req.TenantID {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "coupon not valid for this tenant"}, nil
	}

	// Check max uses
	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "coupon usage limit reached"}, nil
	}

	// Check per-user usage
	userUsageCount, err := s.repo.GetUserCouponUsageCount(ctx, coupon.ID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check coupon usage: %w", err)
	}
	if userUsageCount >= int64(coupon.MaxUsesPerUser) {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "you have already used this coupon"}, nil
	}

	// Get promotion for discount details
	promotion, err := s.repo.GetPromotionByID(ctx, coupon.PromotionID)
	if err != nil {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "associated promotion not found"}, nil
	}

	// Check promotion is active and within date range
	now := time.Now().UTC()
	if promotion.Status != models.StatusActive || now.Before(promotion.StartDate) || now.After(promotion.EndDate) {
		return &models.ValidateCouponResponse{Valid: false, Code: code, Message: "promotion is not active"}, nil
	}

	// Check minimum order amount
	if promotion.MinOrderAmount > 0 && req.OrderTotal < promotion.MinOrderAmount {
		return &models.ValidateCouponResponse{
			Valid:   false,
			Code:    code,
			Message: fmt.Sprintf("minimum order amount is %.2f", promotion.MinOrderAmount),
		}, nil
	}

	discountAmount := calculateDiscount(promotion, req.OrderTotal)

	return &models.ValidateCouponResponse{
		Valid:          true,
		Code:           code,
		DiscountType:   promotion.DiscountType,
		DiscountValue:  promotion.DiscountValue,
		DiscountAmount: discountAmount,
	}, nil
}

func (s *promotionService) ApplyCoupon(ctx context.Context, req *models.ApplyCouponRequest) (*models.ValidateCouponResponse, error) {
	// Validate first
	validateReq := &models.ValidateCouponRequest{
		TenantID:   req.TenantID,
		UserID:     req.UserID,
		OrderTotal: req.OrderTotal,
	}

	result, err := s.ValidateCoupon(ctx, req.Code, validateReq)
	if err != nil {
		return nil, err
	}

	if !result.Valid {
		return result, nil
	}

	// Record usage
	code := strings.ToUpper(req.Code)
	coupon, _ := s.repo.GetCouponByCode(ctx, code)

	usage := &models.CouponUsage{
		ID:       uuid.New().String(),
		CouponID: coupon.ID,
		UserID:   req.UserID,
		OrderID:  req.OrderID,
		Discount: result.DiscountAmount,
		UsedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateCouponUsage(ctx, usage); err != nil {
		return nil, fmt.Errorf("failed to record coupon usage: %w", err)
	}

	// Increment usage count
	coupon.UsedCount++
	coupon.UpdatedAt = time.Now().UTC()
	s.repo.UpdateCoupon(ctx, coupon)

	s.publishEvent("CouponApplied", req.TenantID, map[string]interface{}{
		"coupon_code":     code,
		"user_id":         req.UserID,
		"order_id":        req.OrderID,
		"discount_amount": result.DiscountAmount,
	})

	return result, nil
}

// --- Loyalty ---

func (s *promotionService) GetLoyaltyAccount(ctx context.Context, tenantID, userID string) (*models.LoyaltyAccountResponse, error) {
	account, err := s.repo.GetLoyaltyAccount(ctx, tenantID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// Create new account
			account = &models.LoyaltyAccount{
				ID:        uuid.New().String(),
				TenantID:  tenantID,
				UserID:    userID,
				Points:    0,
				TierLevel: "bronze",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := s.repo.CreateLoyaltyAccount(ctx, account); err != nil {
				return nil, fmt.Errorf("failed to create loyalty account: %w", err)
			}
		} else {
			return nil, err
		}
	}

	return &models.LoyaltyAccountResponse{
		UserID:    account.UserID,
		TenantID:  account.TenantID,
		Points:    account.Points,
		TierLevel: account.TierLevel,
	}, nil
}

func (s *promotionService) ProcessLoyaltyPoints(ctx context.Context, req *models.LoyaltyPointsRequest) (*models.LoyaltyAccountResponse, error) {
	// Get or create account
	account, err := s.repo.GetLoyaltyAccount(ctx, req.TenantID, req.UserID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			account = &models.LoyaltyAccount{
				ID:        uuid.New().String(),
				TenantID:  req.TenantID,
				UserID:    req.UserID,
				Points:    0,
				TierLevel: "bronze",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := s.repo.CreateLoyaltyAccount(ctx, account); err != nil {
				return nil, fmt.Errorf("failed to create loyalty account: %w", err)
			}
		} else {
			return nil, err
		}
	}

	switch req.Type {
	case models.TransactionEarn:
		account.Points += req.Points
	case models.TransactionRedeem:
		if account.Points < req.Points {
			return nil, fmt.Errorf("insufficient loyalty points: have %d, need %d", account.Points, req.Points)
		}
		account.Points -= req.Points
	default:
		return nil, fmt.Errorf("invalid transaction type: %s", req.Type)
	}

	// Update tier
	account.TierLevel = calculateTier(account.Points)
	account.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateLoyaltyAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to update loyalty account: %w", err)
	}

	// Record transaction
	tx := &models.LoyaltyTransaction{
		ID:          uuid.New().String(),
		AccountID:   account.ID,
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Type:        req.Type,
		Points:      req.Points,
		OrderID:     req.OrderID,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateLoyaltyTransaction(ctx, tx); err != nil {
		s.logger.WithError(err).Error("Failed to record loyalty transaction")
	}

	eventType := "LoyaltyPointsEarned"
	if req.Type == models.TransactionRedeem {
		eventType = "LoyaltyPointsRedeemed"
	}
	s.publishEvent(eventType, req.TenantID, map[string]interface{}{
		"user_id":  req.UserID,
		"points":   req.Points,
		"type":     req.Type,
		"balance":  account.Points,
		"order_id": req.OrderID,
	})

	return &models.LoyaltyAccountResponse{
		UserID:    account.UserID,
		TenantID:  account.TenantID,
		Points:    account.Points,
		TierLevel: account.TierLevel,
	}, nil
}

// --- Helpers ---

func calculateDiscount(promotion *models.Promotion, orderTotal float64) float64 {
	var discount float64

	switch promotion.DiscountType {
	case models.DiscountPercentage:
		discount = orderTotal * (promotion.DiscountValue / 100)
	case models.DiscountFixedAmount:
		discount = promotion.DiscountValue
	case models.DiscountFreeShipping:
		discount = 0 // handled separately
	}

	// Cap at max discount
	if promotion.MaxDiscount > 0 && discount > promotion.MaxDiscount {
		discount = promotion.MaxDiscount
	}

	// Can't discount more than order total
	if discount > orderTotal {
		discount = orderTotal
	}

	return discount
}

func calculateTier(points int) string {
	switch {
	case points >= 10000:
		return "platinum"
	case points >= 5000:
		return "gold"
	case points >= 1000:
		return "silver"
	default:
		return "bronze"
	}
}

func toPromotionResponse(p *models.Promotion) *models.PromotionResponse {
	return &models.PromotionResponse{
		ID:             p.ID,
		TenantID:       p.TenantID,
		Name:           p.Name,
		Description:    p.Description,
		DiscountType:   p.DiscountType,
		DiscountValue:  p.DiscountValue,
		MinOrderAmount: p.MinOrderAmount,
		MaxDiscount:    p.MaxDiscount,
		Status:         p.Status,
		StartDate:      p.StartDate,
		EndDate:        p.EndDate,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func (s *promotionService) publishEvent(eventType, tenantID string, payload interface{}) {
	if s.writer == nil {
		return
	}

	event := models.PromotionEvent{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal promotion event")
		return
	}

	err = s.writer.WriteMessages(context.Background(), kafka.Message{
		Topic: "promotion-events",
		Key:   []byte(tenantID),
		Value: data,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to publish promotion event")
	}
}
