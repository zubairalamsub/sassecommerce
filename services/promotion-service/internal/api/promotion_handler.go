package api

import (
	"net/http"
	"strings"

	"github.com/ecommerce/promotion-service/internal/models"
	"github.com/ecommerce/promotion-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PromotionHandler struct {
	service service.PromotionService
	logger  *logrus.Logger
}

func NewPromotionHandler(service service.PromotionService, logger *logrus.Logger) *PromotionHandler {
	return &PromotionHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *PromotionHandler) {
	v1 := router.Group("/api/v1")
	{
		// Promotions
		v1.POST("/promotions", handler.CreatePromotion)
		v1.GET("/promotions/:id", handler.GetPromotion)
		v1.GET("/promotions/active", handler.GetActivePromotions)

		// Coupons
		v1.POST("/coupons", handler.CreateCoupon)
		v1.POST("/coupons/validate/:code", handler.ValidateCoupon)
		v1.POST("/coupons/apply", handler.ApplyCoupon)

		// Loyalty
		v1.GET("/loyalty/:userId", handler.GetLoyaltyAccount)
		v1.POST("/loyalty/points", handler.ProcessLoyaltyPoints)
	}
}

func (h *PromotionHandler) CreatePromotion(c *gin.Context) {
	var req models.CreatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreatePromotion(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "end date") || strings.Contains(err.Error(), "percentage") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to create promotion")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create promotion"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *PromotionHandler) GetPromotion(c *gin.Context) {
	id := c.Param("id")

	result, err := h.service.GetPromotion(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get promotion")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get promotion"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PromotionHandler) GetActivePromotions(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	result, err := h.service.GetActivePromotions(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get active promotions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get active promotions"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PromotionHandler) CreateCoupon(c *gin.Context) {
	var req models.CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateCoupon(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to create coupon")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create coupon"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *PromotionHandler) ValidateCoupon(c *gin.Context) {
	code := c.Param("code")

	var req models.ValidateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ValidateCoupon(c.Request.Context(), code, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to validate coupon")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate coupon"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PromotionHandler) ApplyCoupon(c *gin.Context) {
	var req models.ApplyCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ApplyCoupon(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to apply coupon")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply coupon"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PromotionHandler) GetLoyaltyAccount(c *gin.Context) {
	userID := c.Param("userId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	result, err := h.service.GetLoyaltyAccount(c.Request.Context(), tenantID, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get loyalty account")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get loyalty account"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PromotionHandler) ProcessLoyaltyPoints(c *gin.Context) {
	var req models.LoyaltyPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.ProcessLoyaltyPoints(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "invalid transaction") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to process loyalty points")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process loyalty points"})
		return
	}

	c.JSON(http.StatusOK, result)
}
