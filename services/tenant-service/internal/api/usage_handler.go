package api

import (
	"net/http"

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
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /admin/usage [get]
func (h *UsageHandler) GetUsage(c *gin.Context) {
	rows, err := h.service.GetTenantUsage(c.Request.Context())
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
