package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// UsageHandler serves the admin-only tenant usage report.
type UsageHandler struct {
	service service.UsageService
	logger  *logrus.Logger
}

func NewUsageHandler(svc service.UsageService, logger *logrus.Logger) *UsageHandler {
	return &UsageHandler{
		service: svc,
		logger:  logger,
	}
}

// GetUsage godoc
// @Summary List per-tenant usage statistics
// @Description Returns one row per tenant with audit log activity/size from tenant_db.
// @Description Gated by the frontend super-admin layout — no auth check here.
// @Tags admin
// @Produce json
// @Param window_days query int false "Bound the audit-log aggregate to the last N days (omit for all-time)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/usage [get]
func (h *UsageHandler) GetUsage(c *gin.Context) {
	// Optional ?window_days=N bounds the aggregate to recent activity; absent
	// keeps the historical all-time behavior.
	var since *time.Time
	if windowStr := c.Query("window_days"); windowStr != "" {
		days, err := strconv.Atoi(windowStr)
		if err != nil || days <= 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "window_days must be a positive integer"})
			return
		}
		t := time.Now().UTC().AddDate(0, 0, -days)
		since = &t
	}

	rows, err := h.service.GetTenantUsage(c.Request.Context(), since)
	if err != nil {
		h.logger.WithError(err).Error("Failed to load tenant usage report")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to load tenant usage report"})
		return
	}

	// Always return an empty array (never null) when there are no rows so the
	// frontend can safely iterate over `data`.
	if rows == nil {
		rows = []repository.TenantUsageRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rows,
	})
}
