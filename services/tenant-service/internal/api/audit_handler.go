package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AuditHandler struct {
	service service.AuditService
	logger  *logrus.Logger
}

func NewAuditHandler(service service.AuditService, logger *logrus.Logger) *AuditHandler {
	return &AuditHandler{
		service: service,
		logger:  logger,
	}
}

// GetAuditLog godoc
// @Summary Get a single audit log
// @Description Get a single audit log entry by ID
// @Tags audit
// @Produce json
// @Param id path string true "Audit Log ID"
// @Success 200 {object} models.AuditLog
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /audit-logs/{id} [get]
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "id is required"})
		return
	}

	log, err := h.service.GetAuditLog(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit log")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Audit log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}

// ListAuditLogs godoc
// @Summary List audit logs
// @Description List audit logs with filtering and pagination
// @Tags audit
// @Produce json
// @Param tenant_id query string true "Tenant ID"
// @Param user_id query string false "User ID"
// @Param action query string false "Action (CREATE, READ, UPDATE, DELETE, LOGIN, LOGOUT)"
// @Param resource query string false "Resource type (tenant, user, product, order)"
// @Param resource_id query string false "Resource ID"
// @Param start_date query string false "Start date (ISO 8601)"
// @Param end_date query string false "End date (ISO 8601)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(25)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /audit-logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		// Also check header
		tenantID = c.GetHeader("X-Tenant-Id")
	}
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "25"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 25
	}

	filters := repository.AuditFilters{
		TenantID:   tenantID,
		UserID:     c.Query("user_id"),
		Action:     c.Query("action"),
		Resource:   c.Query("resource"),
		ResourceID: c.Query("resource_id"),
		StartDate:  c.Query("start_date"),
		EndDate:    c.Query("end_date"),
		Page:       page,
		PageSize:   pageSize,
	}

	logs, total, err := h.service.GetAuditLogs(c.Request.Context(), filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list audit logs")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       logs,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}
