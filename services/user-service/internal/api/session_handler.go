package api

import (
	"net/http"

	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
)

// sessionContext derives the per-request session metadata recorded on issued
// refresh tokens.
func sessionContext(c *gin.Context) service.SessionContext {
	return service.SessionContext{
		UserAgent: c.Request.UserAgent(),
		IPAddress: c.ClientIP(),
	}
}

// VerifyTwoFactor completes a 2FA-gated login.
// @Summary Complete two-factor login
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.TwoFactorVerifyRequest true "Challenge token and TOTP/backup code"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/login/2fa [post]
func (h *AuthHandler) VerifyTwoFactor(c *gin.Context) {
	var req models.TwoFactorVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	resp, err := h.authService.VerifyTwoFactorChallenge(c.Request.Context(), &req, sessionContext(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// Refresh rotates a refresh token and returns a new access + refresh pair.
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	resp, err := h.authService.RefreshAccessToken(c.Request.Context(), &req, sessionContext(c))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// logoutRequest carries the optional refresh token to revoke on logout.
type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout revokes the presented refresh token. Always succeeds: logout is
// idempotent and must not enable token enumeration.
// @Summary Logout
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req) // body is optional

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.logger.WithError(err).Warn("Logout failed")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out",
	})
}

// LogoutAll revokes every live session for the authenticated user.
// @Summary Logout all sessions
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).Error("Failed to revoke all sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to revoke sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All sessions revoked",
	})
}

// ListSessions lists the authenticated user's live sessions. The optional
// X-Refresh-Token header lets the client have its own session flagged as
// current (the token is never echoed back).
// @Summary List sessions
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/sessions [get]
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	sessions, err := h.authService.ListSessions(c.Request.Context(), userID, c.GetHeader("X-Refresh-Token"))
	if err != nil {
		h.logger.WithError(err).Error("Failed to list sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sessions,
	})
}

// RevokeSession revokes one of the authenticated user's sessions by id.
// @Summary Revoke a session
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/auth/sessions/{id} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if err := h.authService.RevokeSession(c.Request.Context(), userID, c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "session not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session revoked",
	})
}
