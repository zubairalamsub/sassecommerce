package api

import (
	"net/http"

	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TwoFactorHandler exposes TOTP enrollment management. The login-time
// challenge flow lives on AuthHandler (VerifyTwoFactor).
type TwoFactorHandler struct {
	twoFactor   service.TwoFactorService
	authService service.AuthService
	logger      *logrus.Logger
}

func NewTwoFactorHandler(twoFactor service.TwoFactorService, authService service.AuthService, logger *logrus.Logger) *TwoFactorHandler {
	return &TwoFactorHandler{twoFactor: twoFactor, authService: authService, logger: logger}
}

// BeginSetup starts TOTP enrollment: generates the secret and provisioning QR.
// @Summary Begin 2FA setup
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/2fa/setup [post]
func (h *TwoFactorHandler) BeginSetup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}

	resp, err := h.twoFactor.BeginSetup(c.Request.Context(), userID, middleware.GetTenantID(c), middleware.GetUserEmail(c))
	if err != nil {
		h.logger.WithError(err).Error("Failed to begin 2FA setup")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to begin 2FA setup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// ConfirmSetup verifies the first TOTP code and activates 2FA. All existing
// sessions are revoked so every device re-authenticates through the new gate.
// @Summary Confirm 2FA setup
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TwoFactorConfirmRequest true "First TOTP code"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/2fa/confirm [post]
func (h *TwoFactorHandler) ConfirmSetup(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}

	var req models.TwoFactorConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body"})
		return
	}

	resp, err := h.twoFactor.ConfirmSetup(c.Request.Context(), userID, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Security-state change: force every session through the new 2FA gate.
	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).Warn("Failed to revoke sessions after enabling 2FA")
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// Disable turns 2FA off. Requires re-authentication with the password AND a
// valid TOTP/backup code, then revokes all sessions.
// @Summary Disable 2FA
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TwoFactorDisableRequest true "Password and current code"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/2fa/disable [post]
func (h *TwoFactorHandler) Disable(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}

	var req models.TwoFactorDisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request body"})
		return
	}

	if err := h.authService.VerifyUserPassword(c.Request.Context(), userID, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "incorrect password"})
		return
	}
	ok, err := h.twoFactor.Verify(c.Request.Context(), userID, req.Code)
	if err != nil || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid 2fa code"})
		return
	}

	if err := h.twoFactor.Disable(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).Error("Failed to disable 2FA")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to disable 2FA"})
		return
	}

	// Security-state change: revoke all sessions.
	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).Warn("Failed to revoke sessions after disabling 2FA")
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Two-factor authentication disabled"})
}

// Status reports whether 2FA is enabled for the authenticated user.
// @Summary 2FA status
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/2fa [get]
func (h *TwoFactorHandler) Status(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "User not authenticated"})
		return
	}

	enabled, err := h.twoFactor.IsEnabled(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check 2FA status")
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to check 2FA status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"enabled": enabled}})
}
