package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ecommerce/recommendation-service/internal/models"
	"github.com/ecommerce/recommendation-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type RecommendationHandler struct {
	service service.RecommendationService
	logger  *logrus.Logger
}

func NewRecommendationHandler(service service.RecommendationService, logger *logrus.Logger) *RecommendationHandler {
	return &RecommendationHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *RecommendationHandler) {
	v1 := router.Group("/api/v1/recommendations")
	{
		v1.GET("/user/:userId", handler.GetUserRecommendations)
		v1.GET("/product/:productId", handler.GetProductRecommendations)
		v1.POST("/train", handler.TrainModel)
		v1.GET("/train/:jobId", handler.GetTrainingJob)
	}
}

func (h *RecommendationHandler) GetUserRecommendations(c *gin.Context) {
	userID := c.Param("userId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.service.GetUserRecommendations(c.Request.Context(), tenantID, userID, limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get user recommendations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *RecommendationHandler) GetProductRecommendations(c *gin.Context) {
	productID := c.Param("productId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.service.GetProductRecommendations(c.Request.Context(), tenantID, productID, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get product recommendations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recommendations"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *RecommendationHandler) TrainModel(c *gin.Context) {
	var req models.TrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.TrainModel(c.Request.Context(), req.TenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start training")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start training"})
		return
	}

	c.JSON(http.StatusAccepted, result)
}

func (h *RecommendationHandler) GetTrainingJob(c *gin.Context) {
	jobID := c.Param("jobId")

	result, err := h.service.GetTrainingJob(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get training job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get training job"})
		return
	}

	c.JSON(http.StatusOK, result)
}
