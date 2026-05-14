package api

import (
	"net/http"

	"github.com/ecommerce/product-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ImageHandler exposes the product image upload endpoints.
type ImageHandler struct {
	imageService service.ImageService
	logger       *logrus.Logger
}

func NewImageHandler(imageService service.ImageService, logger *logrus.Logger) *ImageHandler {
	return &ImageHandler{imageService: imageService, logger: logger}
}

// PresignImageUploadRequest is the request body for POST /products/:id/images/presign.
type PresignImageUploadRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	Filename    string `json:"filename"`
}

// ConfirmImageUploadRequest is the request body for POST /products/:id/images.
type ConfirmImageUploadRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
}

// RemoveImageRequest is the request body for DELETE /products/:id/images.
type RemoveImageRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
}

// PresignImageUpload issues a presigned URL for direct browser → storage upload.
// @Summary Presign a product image upload
// @Tags Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body PresignImageUploadRequest true "Upload metadata"
// @Success 200 {object} service.PresignUploadResult
// @Router /products/{id}/images/presign [post]
func (h *ImageHandler) PresignImageUpload(c *gin.Context) {
	productID := c.Param("id")
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req PresignImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	res, err := h.imageService.PresignUpload(c.Request.Context(), tenantID, productID, req.ContentType, req.Filename)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to presign image upload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ConfirmImageUpload records a successfully uploaded image on the product.
// @Summary Confirm an uploaded product image
// @Tags Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body ConfirmImageUploadRequest true "Image URL"
// @Success 200 {object} map[string]interface{}
// @Router /products/{id}/images [post]
func (h *ImageHandler) ConfirmImageUpload(c *gin.Context) {
	productID := c.Param("id")
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req ConfirmImageUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if err := h.imageService.ConfirmUpload(c.Request.Context(), tenantID, productID, req.ImageURL); err != nil {
		h.logger.WithError(err).Warn("Failed to confirm image upload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "image_url": req.ImageURL})
}

// RemoveImage removes an image from the product and deletes the object.
// @Summary Remove a product image
// @Tags Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body RemoveImageRequest true "Image URL"
// @Success 200 {object} map[string]interface{}
// @Router /products/{id}/images [delete]
func (h *ImageHandler) RemoveImage(c *gin.Context) {
	productID := c.Param("id")
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	var req RemoveImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if err := h.imageService.RemoveImage(c.Request.Context(), tenantID, productID, req.ImageURL); err != nil {
		h.logger.WithError(err).Warn("Failed to remove product image")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RegisterRoutes registers the image-management routes onto the products group.
// All routes require the same auth + admin/moderator role as other write routes.
func (h *ImageHandler) RegisterRoutes(productsGroup *gin.RouterGroup) {
	productsGroup.POST("/:id/images/presign", h.PresignImageUpload)
	productsGroup.POST("/:id/images", h.ConfirmImageUpload)
	productsGroup.DELETE("/:id/images", h.RemoveImage)
}
