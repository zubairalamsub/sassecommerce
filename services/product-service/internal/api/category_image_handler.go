package api

import (
	"net/http"

	"github.com/ecommerce/product-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	sharedvalidator "github.com/ecommerce/shared/go/pkg/validator"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CategoryImageHandler exposes the category image upload endpoints.
type CategoryImageHandler struct {
	imageService service.CategoryImageService
	logger       *logrus.Logger
}

func NewCategoryImageHandler(imageService service.CategoryImageService, logger *logrus.Logger) *CategoryImageHandler {
	return &CategoryImageHandler{imageService: imageService, logger: logger}
}

// PresignCategoryImageUploadRequest is the request body for POST /categories/:id/image/presign.
type PresignCategoryImageUploadRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	Filename    string `json:"filename"`
}

// ConfirmCategoryImageUploadRequest is the request body for POST /categories/:id/image.
type ConfirmCategoryImageUploadRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
}

// PresignImageUpload issues a presigned URL for direct browser → storage upload.
// @Summary Presign a category image upload
// @Tags Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param request body PresignCategoryImageUploadRequest true "Upload metadata"
// @Success 200 {object} service.PresignUploadResult
// @Router /categories/{id}/image/presign [post]
func (h *CategoryImageHandler) PresignImageUpload(c *gin.Context) {
	categoryID := c.Param("id")
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req PresignCategoryImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": sharedvalidator.SanitizedBindingErrors(err)})
		return
	}

	res, err := h.imageService.PresignUpload(c.Request.Context(), tenantID, categoryID, req.ContentType, req.Filename)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to presign category image upload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ConfirmImageUpload records a successfully uploaded image on the category.
// @Summary Confirm an uploaded category image
// @Tags Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param request body ConfirmCategoryImageUploadRequest true "Image URL"
// @Success 200 {object} map[string]interface{}
// @Router /categories/{id}/image [post]
func (h *CategoryImageHandler) ConfirmImageUpload(c *gin.Context) {
	categoryID := c.Param("id")
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req ConfirmCategoryImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": sharedvalidator.SanitizedBindingErrors(err)})
		return
	}

	if err := h.imageService.ConfirmUpload(c.Request.Context(), tenantID, categoryID, req.ImageURL); err != nil {
		h.logger.WithError(err).Warn("Failed to confirm category image upload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "image_url": req.ImageURL})
}

// RemoveImage clears the category's image and deletes the underlying object.
// @Summary Remove a category image
// @Tags Categories
// @Security BearerAuth
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} map[string]interface{}
// @Router /categories/{id}/image [delete]
func (h *CategoryImageHandler) RemoveImage(c *gin.Context) {
	categoryID := c.Param("id")
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	if err := h.imageService.RemoveImage(c.Request.Context(), tenantID, categoryID); err != nil {
		h.logger.WithError(err).Warn("Failed to remove category image")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RegisterRoutes registers the category image-management routes onto the
// categories group. All routes require admin/moderator auth — matching the
// existing write routes on the category handler.
func (h *CategoryImageHandler) RegisterRoutes(categoriesGroup *gin.RouterGroup) {
	categoriesGroup.POST("/:id/image/presign", h.PresignImageUpload)
	categoriesGroup.POST("/:id/image", h.ConfirmImageUpload)
	categoriesGroup.DELETE("/:id/image", h.RemoveImage)
}
