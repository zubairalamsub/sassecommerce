package api

import (
	"net/http"

	sharedvalidator "github.com/ecommerce/shared/go/pkg/validator"
	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AvatarHandler exposes the avatar upload endpoints. Each user can manage
// their own avatar; admins can manage any user's via the /:userID variants.
type AvatarHandler struct {
	avatarService service.AvatarService
	logger        *logrus.Logger
}

func NewAvatarHandler(avatarService service.AvatarService, logger *logrus.Logger) *AvatarHandler {
	return &AvatarHandler{avatarService: avatarService, logger: logger}
}

type AvatarPresignRequest struct {
	ContentType string `json:"content_type" binding:"required"`
	Filename    string `json:"filename"`
}

type AvatarConfirmRequest struct {
	ImageURL string `json:"image_url" binding:"required"`
}

// PresignAvatarUpload presigns an upload URL for the authenticated user's avatar.
func (h *AvatarHandler) PresignAvatarUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	if userID == "" || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}
	var req AvatarPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body", "details": sharedvalidator.SanitizedBindingErrors(err)})
		return
	}
	res, err := h.avatarService.PresignUpload(c.Request.Context(), tenantID, userID, req.ContentType, req.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

// ConfirmAvatarUpload records the avatar URL on the user.
func (h *AvatarHandler) ConfirmAvatarUpload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	if userID == "" || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}
	var req AvatarConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body", "details": sharedvalidator.SanitizedBindingErrors(err)})
		return
	}
	if err := h.avatarService.ConfirmUpload(c.Request.Context(), tenantID, userID, req.ImageURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "image_url": req.ImageURL})
}

// RemoveAvatar clears the avatar and deletes the storage object.
func (h *AvatarHandler) RemoveAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	if userID == "" || tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}
	if err := h.avatarService.Remove(c.Request.Context(), tenantID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
