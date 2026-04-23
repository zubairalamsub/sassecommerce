package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService service.UserService
	logger      *logrus.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService service.UserService, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// GetUser retrieves a user by ID
// @Summary Get user
// @Description Get user by ID
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} models.UserResponse
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// ListUsers retrieves users with pagination
// @Summary List users
// @Description List all users for a tenant with pagination
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Tenant ID required",
		})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	users, total, err := h.userService.ListUsers(c.Request.Context(), tenantID, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_items": total,
			"total_pages": totalPages,
		},
	})
}

// UpdateUser updates a user's profile
// @Summary Update user
// @Description Update user profile
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body models.UpdateUserRequest true "Update request"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := middleware.GetUserID(c)
	currentUserRole := middleware.GetUserRole(c)

	// Users can only update their own profile unless they're admin
	if userID != currentUserID && currentUserRole != models.UserRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You can only update your own profile",
		})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid update user request")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
		"message": "User updated successfully",
	})
}

// DeleteUser deletes a user
// @Summary Delete user
// @Description Delete a user (soft delete)
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := middleware.GetUserID(c)

	// Users can only delete their own account
	if userID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You can only delete your own account",
		})
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// UpdateUserRole updates a user's role (admin only)
// @Summary Update user role
// @Description Update user role (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param role body string true "New role"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/users/{id}/role [patch]
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Role models.UserRole `json:"role" binding:"required,oneof=admin moderator customer guest"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := h.userService.UpdateUserRole(c.Request.Context(), userID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User role updated successfully",
	})
}

// UpdateUserStatus updates a user's status (admin only)
// @Summary Update user status
// @Description Update user status (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param status body string true "New status"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/users/{id}/status [patch]
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Status models.UserStatus `json:"status" binding:"required,oneof=active inactive suspended deleted"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if err := h.userService.UpdateUserStatus(c.Request.Context(), userID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User status updated successfully",
	})
}
