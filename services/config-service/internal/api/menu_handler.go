package api

import (
	"net/http"
	"strings"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type MenuHandler struct {
	service service.MenuService
	logger  *logrus.Logger
}

func NewMenuHandler(service service.MenuService, logger *logrus.Logger) *MenuHandler {
	return &MenuHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterMenuRoutes(router *gin.Engine, handler *MenuHandler) {
	v1 := router.Group("/api/v1/menus")
	{
		v1.POST("", handler.CreateMenu)
		v1.GET("", handler.ListMenus)
		v1.GET("/:id", handler.GetMenu)
		v1.PUT("/:id", handler.UpdateMenu)
		v1.DELETE("/:id", handler.DeleteMenu)
	}

	// Separate groups to avoid Gin wildcard param name conflicts
	slug := router.Group("/api/v1/menus/slug")
	{
		slug.GET("/:slug", handler.GetMenuBySlug)
	}

	location := router.Group("/api/v1/menus/location")
	{
		location.GET("/:location", handler.ListMenusByLocation)
	}

	items := router.Group("/api/v1/menu-items")
	{
		items.POST("/:menuId", handler.CreateMenuItem)
		items.PUT("/:itemId", handler.UpdateMenuItem)
		items.DELETE("/:itemId", handler.DeleteMenuItem)
	}

	reorder := router.Group("/api/v1/menus-reorder")
	{
		reorder.PUT("/:menuId", handler.ReorderItems)
	}
}

func (h *MenuHandler) CreateMenu(c *gin.Context) {
	var req models.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateMenu(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid location") || strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to create menu")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create menu"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *MenuHandler) GetMenu(c *gin.Context) {
	id := c.Param("id")

	result, err := h.service.GetMenu(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get menu")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get menu"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *MenuHandler) GetMenuBySlug(c *gin.Context) {
	slug := c.Param("slug")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	result, err := h.service.GetMenuBySlug(c.Request.Context(), tenantID, slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get menu by slug")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get menu"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *MenuHandler) UpdateMenu(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateMenu(c.Request.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "invalid location") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to update menu")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update menu"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *MenuHandler) DeleteMenu(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteMenu(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to delete menu")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete menu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "menu deleted"})
}

func (h *MenuHandler) ListMenus(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	results, err := h.service.ListMenus(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list menus")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list menus"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "count": len(results)})
}

func (h *MenuHandler) ListMenusByLocation(c *gin.Context) {
	location := c.Param("location")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	results, err := h.service.ListMenusByLocation(c.Request.Context(), tenantID, location)
	if err != nil {
		if strings.Contains(err.Error(), "invalid location") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to list menus by location")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list menus"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "count": len(results)})
}

func (h *MenuHandler) CreateMenuItem(c *gin.Context) {
	menuID := c.Param("menuId")

	var req models.CreateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.MenuID = menuID

	result, err := h.service.CreateMenuItem(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to create menu item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create menu item"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *MenuHandler) UpdateMenuItem(c *gin.Context) {
	itemID := c.Param("itemId")

	var req models.UpdateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateMenuItem(c.Request.Context(), itemID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to update menu item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update menu item"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *MenuHandler) DeleteMenuItem(c *gin.Context) {
	itemID := c.Param("itemId")

	err := h.service.DeleteMenuItem(c.Request.Context(), itemID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to delete menu item")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete menu item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "menu item deleted"})
}

func (h *MenuHandler) ReorderItems(c *gin.Context) {
	menuID := c.Param("menuId")

	var req models.ReorderItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.ReorderItems(c.Request.Context(), menuID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to reorder items")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reorder items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "items reordered"})
}
