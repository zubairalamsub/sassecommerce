package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type NotificationHandler struct {
	service service.NotificationService
	logger  *logrus.Logger
}

func NewNotificationHandler(service service.NotificationService, logger *logrus.Logger) *NotificationHandler {
	return &NotificationHandler{
		service: service,
		logger:  logger,
	}
}

func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req models.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	notification, err := h.service.SendNotification(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to send notification")
		c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *NotificationHandler) GetNotification(c *gin.Context) {
	id := c.Param("id")

	notification, err := h.service.GetNotification(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, notification)
}

func (h *NotificationHandler) GetUserNotifications(c *gin.Context) {
	userID := c.Param("userId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	notifications, total, err := h.service.GetUserNotifications(c.Request.Context(), tenantID, userID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user notifications")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ListNotificationsResponse{
		Data: notifications,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.MarkAsRead(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Notification marked as read"})
}

func (h *NotificationHandler) GetPreference(c *gin.Context) {
	userID := c.Param("userId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	pref, err := h.service.GetPreference(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (h *NotificationHandler) UpdatePreference(c *gin.Context) {
	userID := c.Param("userId")
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id query parameter is required"})
		return
	}

	var req models.UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	pref, err := h.service.UpdatePreference(c.Request.Context(), tenantID, userID, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update preferences")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, pref)
}

// RegisterRoutes sets up the notification API routes
func RegisterRoutes(router *gin.Engine, handler *NotificationHandler) {
	v1 := router.Group("/api/v1")
	{
		notifications := v1.Group("/notifications")
		{
			notifications.POST("/send", handler.SendNotification)
			notifications.GET("/:id", handler.GetNotification)
			notifications.GET("/user/:userId", handler.GetUserNotifications)
			notifications.PUT("/:id/read", handler.MarkAsRead)
		}

		preferences := v1.Group("/preferences")
		{
			preferences.GET("/:userId", handler.GetPreference)
			preferences.PUT("/:userId", handler.UpdatePreference)
		}
	}
}

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ListNotificationsResponse struct {
	Data       []models.NotificationResponse `json:"data"`
	Pagination Pagination                    `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
