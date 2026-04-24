package api

import (
	"net/http"

	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// WishlistHandler handles wishlist HTTP requests
type WishlistHandler struct {
	wishlistRepo repository.WishlistRepository
	logger       *logrus.Logger
}

// NewWishlistHandler creates a new wishlist handler
func NewWishlistHandler(wishlistRepo repository.WishlistRepository, logger *logrus.Logger) *WishlistHandler {
	return &WishlistHandler{
		wishlistRepo: wishlistRepo,
		logger:       logger,
	}
}

// GetWishlist returns all wishlist items for the authenticated user
func (h *WishlistHandler) GetWishlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		tenantID = c.Query("tenant_id")
	}

	items, err := h.wishlistRepo.List(c.Request.Context(), userID, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch wishlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"count": len(items),
	})
}

// AddWishlistItem adds a product to the user's wishlist
func (h *WishlistHandler) AddWishlistItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var req models.AddWishlistItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := &models.WishlistItem{
		UserID:    userID,
		TenantID:  tenantID,
		ProductID: req.ProductID,
		Name:      req.Name,
		Slug:      req.Slug,
		Price:     req.Price,
		Image:     req.Image,
	}

	if err := h.wishlistRepo.Add(c.Request.Context(), item); err != nil {
		h.logger.WithError(err).Error("Failed to add wishlist item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items, _ := h.wishlistRepo.List(c.Request.Context(), userID, tenantID)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Item added to wishlist",
		"items":   items,
		"count":   len(items),
	})
}

// RemoveWishlistItem removes a product from the user's wishlist
func (h *WishlistHandler) RemoveWishlistItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	productID := c.Param("productId")

	if err := h.wishlistRepo.Remove(c.Request.Context(), userID, tenantID, productID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found in wishlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item removed from wishlist"})
}

// ClearWishlist removes all items from the user's wishlist
func (h *WishlistHandler) ClearWishlist(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	if err := h.wishlistRepo.Clear(c.Request.Context(), userID, tenantID); err != nil {
		h.logger.WithError(err).Error("Failed to clear wishlist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wishlist cleared"})
}
