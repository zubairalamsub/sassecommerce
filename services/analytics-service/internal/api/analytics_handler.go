package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ecommerce/analytics-service/internal/models"
	"github.com/ecommerce/analytics-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type AnalyticsHandler struct {
	service service.AnalyticsService
	logger  *logrus.Logger
}

func NewAnalyticsHandler(service service.AnalyticsService, logger *logrus.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *AnalyticsHandler) {
	v1 := router.Group("/api/v1/analytics")
	{
		v1.GET("/sales", handler.GetSalesReport)
		v1.GET("/customers", handler.GetCustomerInsights)
		v1.GET("/products", handler.GetProductPerformance)
		v1.POST("/reports", handler.CreateReport)
		v1.GET("/reports/:reportId", handler.GetReport)
		v1.GET("/reports", handler.ListReports)
	}
}

func (h *AnalyticsHandler) GetSalesReport(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	req := &models.SalesReportRequest{
		TenantID: tenantID,
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
		Period:   c.DefaultQuery("period", "daily"),
	}

	result, err := h.service.GetSalesReport(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sales report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate sales report"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AnalyticsHandler) GetCustomerInsights(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	req := &models.CustomerInsightsRequest{
		TenantID: tenantID,
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
	}

	result, err := h.service.GetCustomerInsights(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get customer insights")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate customer insights"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AnalyticsHandler) GetProductPerformance(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	req := &models.ProductPerformanceRequest{
		TenantID: tenantID,
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
		Limit:    limit,
	}

	result, err := h.service.GetProductPerformance(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get product performance")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate product performance"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AnalyticsHandler) CreateReport(c *gin.Context) {
	var req models.CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateReport(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "date_to must be") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to create report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create report"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *AnalyticsHandler) GetReport(c *gin.Context) {
	reportID := c.Param("reportId")

	result, err := h.service.GetReport(c.Request.Context(), reportID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get report"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AnalyticsHandler) ListReports(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	results, total, err := h.service.ListReports(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list reports")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
