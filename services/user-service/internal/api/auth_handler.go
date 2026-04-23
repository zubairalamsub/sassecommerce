package api

import (
	"net/http"

	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService service.AuthService
	logger      *logrus.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService service.AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration request"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid registration request")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "email already exists" || err.Error() == "username already exists" {
			status = http.StatusConflict
		}

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
		"message": "User registered successfully",
	})
}

// Login handles user login
// @Summary Login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login request"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	loginResp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "user account is not active" {
			status = http.StatusForbidden
		}

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    loginResp,
	})
}

// ChangePassword handles password change
// @Summary Change password
// @Description Change user password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "Change password request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid change password request")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "incorrect old password" {
			status = http.StatusBadRequest
		}

		c.JSON(status, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// GetProfile returns the current user's profile
// @Summary Get profile
// @Description Get current user profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)
	email := middleware.GetUserEmail(c)
	role := middleware.GetUserRole(c)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":   userID,
			"tenant_id": tenantID,
			"email":     email,
			"role":      role,
		},
	})
}
