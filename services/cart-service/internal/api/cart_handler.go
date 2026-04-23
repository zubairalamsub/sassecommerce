package api

import (
	"net/http"
	"strings"

	"github.com/ecommerce/cart-service/internal/models"
	"github.com/ecommerce/cart-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CartHandler struct {
	service service.CartService
	logger  *logrus.Logger
}

func NewCartHandler(service service.CartService, logger *logrus.Logger) *CartHandler {
	return &CartHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *CartHandler) {
	v1 := router.Group("/api/v1/cart")
	{
		v1.POST("/items", handler.AddItem)
		v1.GET("", handler.GetCart)
		v1.PUT("/items/:itemId", handler.UpdateItem)
		v1.DELETE("/items/:itemId", handler.RemoveItem)
		v1.DELETE("", handler.ClearCart)
	}
}

func (h *CartHandler) AddItem(c *gin.Context) {
	var req models.AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.AddItem(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to add item to cart")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add item to cart"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *CartHandler) GetCart(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	userID := c.Query("user_id")

	if tenantID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and user_id are required"})
		return
	}

	result, err := h.service.GetCart(c.Request.Context(), tenantID, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get cart")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get cart"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *CartHandler) UpdateItem(c *gin.Context) {
	itemID := c.Param("itemId")
	tenantID := c.Query("tenant_id")
	userID := c.Query("user_id")

	if tenantID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and user_id are required"})
		return
	}

	var req models.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateItem(c.Request.Context(), tenantID, userID, itemID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to update cart item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update cart item"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	itemID := c.Param("itemId")
	tenantID := c.Query("tenant_id")
	userID := c.Query("user_id")

	if tenantID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and user_id are required"})
		return
	}

	result, err := h.service.RemoveItem(c.Request.Context(), tenantID, userID, itemID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to remove cart item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove cart item"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *CartHandler) ClearCart(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	userID := c.Query("user_id")

	if tenantID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and user_id are required"})
		return
	}

	if err := h.service.ClearCart(c.Request.Context(), tenantID, userID); err != nil {
		h.logger.WithError(err).Error("Failed to clear cart")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear cart"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
