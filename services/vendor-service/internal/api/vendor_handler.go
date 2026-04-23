package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ecommerce/vendor-service/internal/models"
	"github.com/ecommerce/vendor-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type VendorHandler struct {
	service service.VendorService
	logger  *logrus.Logger
}

func NewVendorHandler(service service.VendorService, logger *logrus.Logger) *VendorHandler {
	return &VendorHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *VendorHandler) {
	v1 := router.Group("/api/v1/vendors")
	{
		v1.POST("/register", handler.RegisterVendor)
		v1.GET("", handler.ListVendors)
		v1.GET("/:vendorId", handler.GetVendor)
		v1.PUT("/:vendorId", handler.UpdateVendor)
		v1.PUT("/:vendorId/status", handler.UpdateVendorStatus)
		v1.GET("/:vendorId/orders", handler.GetVendorOrders)
		v1.GET("/:vendorId/analytics", handler.GetVendorAnalytics)
	}
}

func (h *VendorHandler) RegisterVendor(c *gin.Context) {
	var req models.RegisterVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.RegisterVendor(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to register vendor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register vendor"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *VendorHandler) GetVendor(c *gin.Context) {
	vendorID := c.Param("vendorId")

	result, err := h.service.GetVendor(c.Request.Context(), vendorID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get vendor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vendor"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *VendorHandler) ListVendors(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	results, total, err := h.service.ListVendors(c.Request.Context(), tenantID, status, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list vendors")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list vendors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

func (h *VendorHandler) UpdateVendor(c *gin.Context) {
	vendorID := c.Param("vendorId")

	var req models.UpdateVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateVendor(c.Request.Context(), vendorID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to update vendor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vendor"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *VendorHandler) UpdateVendorStatus(c *gin.Context) {
	vendorID := c.Param("vendorId")

	var req models.UpdateVendorStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UpdateVendorStatus(c.Request.Context(), vendorID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "invalid status transition") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to update vendor status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update vendor status"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *VendorHandler) GetVendorOrders(c *gin.Context) {
	vendorID := c.Param("vendorId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	results, total, err := h.service.GetVendorOrders(c.Request.Context(), vendorID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get vendor orders")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vendor orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

func (h *VendorHandler) GetVendorAnalytics(c *gin.Context) {
	vendorID := c.Param("vendorId")

	result, err := h.service.GetVendorAnalytics(c.Request.Context(), vendorID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get vendor analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get vendor analytics"})
		return
	}

	c.JSON(http.StatusOK, result)
}
