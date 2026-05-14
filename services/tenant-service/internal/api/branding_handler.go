package api

import (
	"net/http"

	"github.com/ecommerce/tenant-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// BrandingHandler exposes the tenant branding upload endpoints.
type BrandingHandler struct {
	brandingService service.BrandingService
	logger          *logrus.Logger
}

func NewBrandingHandler(brandingService service.BrandingService, logger *logrus.Logger) *BrandingHandler {
	return &BrandingHandler{brandingService: brandingService, logger: logger}
}

type BrandingPresignRequest struct {
	Kind        string `json:"kind" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Filename    string `json:"filename"`
}

type BrandingRemoveRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
}

// PresignBrandingUpload issues a presigned PUT URL for a tenant branding asset.
func (h *BrandingHandler) PresignBrandingUpload(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		// Fall back to URL param when middleware isn't strict-tenant.
		tenantID = c.Param("id")
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "tenant_id is required"})
		return
	}

	var req BrandingPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body", "details": err.Error()})
		return
	}

	res, err := h.brandingService.PresignUpload(
		c.Request.Context(), tenantID,
		service.BrandingKind(req.Kind), req.ContentType, req.Filename,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

// RemoveBrandingAsset deletes a previously-uploaded branding object from storage.
func (h *BrandingHandler) RemoveBrandingAsset(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		tenantID = c.Param("id")
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "tenant_id is required"})
		return
	}
	var req BrandingRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body", "details": err.Error()})
		return
	}
	if err := h.brandingService.RemoveAsset(c.Request.Context(), tenantID, req.ImageURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
