package api

import (
	"net/http"

	"github.com/ecommerce/review-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	sharedvalidator "github.com/ecommerce/shared/go/pkg/validator"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AttachmentHandler exposes the review-attachment upload endpoints.
type AttachmentHandler struct {
	attachmentService service.AttachmentService
	logger            *logrus.Logger
}

func NewAttachmentHandler(attachmentService service.AttachmentService, logger *logrus.Logger) *AttachmentHandler {
	return &AttachmentHandler{attachmentService: attachmentService, logger: logger}
}

type AttachmentPresignRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	Filename    string `json:"filename"`
}

type AttachmentRemoveRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
}

// PresignAttachmentUpload issues a presigned PUT URL for a review-attachment image.
func (h *AttachmentHandler) PresignAttachmentUpload(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	userID := sharedmiddleware.GetUserID(c)
	if tenantID == "" || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	var req AttachmentPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": sharedvalidator.SanitizedBindingErrors(err)})
		return
	}
	res, err := h.attachmentService.PresignUpload(c.Request.Context(), tenantID, userID, req.ContentType, req.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// RemoveAttachment deletes a previously-uploaded review attachment.
func (h *AttachmentHandler) RemoveAttachment(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	var req AttachmentRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": sharedvalidator.SanitizedBindingErrors(err)})
		return
	}
	if err := h.attachmentService.Remove(c.Request.Context(), tenantID, req.ImageURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
