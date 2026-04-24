package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/ecommerce/product-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CategoryHandler struct {
	categoryService service.CategoryService
	logger          *logrus.Logger
}

func NewCategoryHandler(categoryService service.CategoryService, logger *logrus.Logger) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		logger:          logger,
	}
}

// CreateCategory handles category creation
// @Summary Create a new category
// @Tags Categories
// @Accept json
// @Produce json
// @Param category body models.CreateCategoryRequest true "Category data"
// @Success 201 {object} models.CategoryResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /categories [post]
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req models.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	category, err := h.categoryService.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create category")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// GetCategory handles retrieving a category by ID
// @Summary Get category by ID
// @Tags Categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} models.CategoryResponse
// @Failure 404 {object} map[string]string
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	id := c.Param("id")

	category, err := h.categoryService.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("category_id", id).Error("Failed to get category")
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// GetCategoryBySlug handles retrieving a category by slug
// @Summary Get category by slug
// @Tags Categories
// @Produce json
// @Param slug path string true "Category Slug"
// @Success 200 {object} models.CategoryResponse
// @Failure 404 {object} map[string]string
// @Router /categories/slug/{slug} [get]
func (h *CategoryHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")
	tenantID := c.GetString("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	category, err := h.categoryService.GetCategoryBySlug(c.Request.Context(), tenantID, slug)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"slug":      slug,
		}).Error("Failed to get category by slug")
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// ListCategories handles listing categories with pagination
// @Summary List categories
// @Tags Categories
// @Produce json
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /categories [get]
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	categories, total, err := h.categoryService.ListCategories(c.Request.Context(), tenantID, offset, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list categories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   categories,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

// ListCategoriesByParent handles listing categories by parent
// @Summary List categories by parent
// @Tags Categories
// @Produce json
// @Param parent_id query string false "Parent ID (empty for root categories)"
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /categories/by-parent [get]
func (h *CategoryHandler) ListCategoriesByParent(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	var parentID *string
	if pid := c.Query("parent_id"); pid != "" {
		parentID = &pid
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	categories, total, err := h.categoryService.ListCategoriesByParent(c.Request.Context(), tenantID, parentID, offset, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list categories by parent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   categories,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

// UpdateCategory handles category updates
// @Summary Update a category
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param category body models.UpdateCategoryRequest true "Category update data"
// @Success 200 {object} models.CategoryResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	category, err := h.categoryService.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithError(err).WithField("category_id", id).Error("Failed to update category")
		if err.Error() == "category not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, category)
}

// DeleteCategory handles category deletion (soft delete)
// @Summary Delete a category
// @Tags Categories
// @Param id path string true "Category ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	if err := h.categoryService.DeleteCategory(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).WithField("category_id", id).Error("Failed to delete category")
		if err.Error() == "failed to delete category" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateCategoryStatus handles category status updates
// @Summary Update category status
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param status body map[string]string true "Status update"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /categories/{id}/status [patch]
func (h *CategoryHandler) UpdateCategoryStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status models.CategoryStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate status
	validStatuses := []models.CategoryStatus{
		models.CategoryStatusActive,
		models.CategoryStatusInactive,
	}
	valid := false
	for _, s := range validStatuses {
		if req.Status == s {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	if err := h.categoryService.UpdateCategoryStatus(c.Request.Context(), id, req.Status); err != nil {
		h.logger.WithError(err).WithField("category_id", id).Error("Failed to update category status")
		if err.Error() == "failed to update category status" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category status updated successfully"})
}

// RegisterRoutes registers all category routes with auth middleware for write operations.
func (h *CategoryHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware ...gin.HandlerFunc) {
	categories := router.Group("/categories")
	{
		// Public read routes
		categories.GET("", h.ListCategories)
		categories.GET("/by-parent", h.ListCategoriesByParent)
		categories.GET("/slug/:slug", h.GetCategoryBySlug)
		categories.GET("/:id", h.GetCategory)

		// Protected write routes — require auth + admin or moderator role
		write := categories.Group("")
		if len(authMiddleware) > 0 {
			write.Use(authMiddleware...)
			write.Use(sharedmiddleware.RequireRole("admin", "moderator"))
		}
		write.POST("", h.CreateCategory)
		write.PUT("/:id", h.UpdateCategory)
		write.DELETE("/:id", h.DeleteCategory)
		write.PATCH("/:id/status", h.UpdateCategoryStatus)
	}
}
