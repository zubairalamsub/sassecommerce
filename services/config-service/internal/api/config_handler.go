package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ConfigHandler struct {
	service service.ConfigService
	logger  *logrus.Logger
}

func NewConfigHandler(service service.ConfigService, logger *logrus.Logger) *ConfigHandler {
	return &ConfigHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *ConfigHandler) {
	v1 := router.Group("/api/v1/config")
	{
		// Config CRUD
		v1.GET("/get", handler.GetConfig)
		v1.POST("/set", handler.SetConfig)
		v1.DELETE("/:id", handler.DeleteConfig)

		// Listing & search
		v1.GET("/namespace/:namespace", handler.ListByNamespace)
		v1.GET("/namespaces", handler.ListNamespaces)
		v1.GET("/search", handler.SearchConfigs)

		// Bulk operations
		v1.POST("/bulk/get", handler.BulkGet)
		v1.POST("/bulk/set", handler.BulkSet)

		// Export
		v1.GET("/export/:namespace", handler.ExportNamespace)

		// Audit
		v1.GET("/audit", handler.GetAuditLog)
		v1.GET("/audit/:configId", handler.GetConfigHistory)
	}
}

func (h *ConfigHandler) GetConfig(c *gin.Context) {
	namespace := c.Query("namespace")
	key := c.Query("key")
	if namespace == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and key are required"})
		return
	}

	env := c.DefaultQuery("environment", "all")
	tenantID := c.Query("tenant_id")

	result, err := h.service.GetConfig(c.Request.Context(), namespace, key, env, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to get config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ConfigHandler) SetConfig(c *gin.Context) {
	var req models.SetConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.SetConfig(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to set config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set config"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteConfig(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		h.logger.WithError(err).Error("Failed to delete config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config deleted"})
}

func (h *ConfigHandler) ListByNamespace(c *gin.Context) {
	namespace := c.Param("namespace")
	env := c.DefaultQuery("environment", "")
	tenantID := c.Query("tenant_id")

	results, err := h.service.ListByNamespace(c.Request.Context(), namespace, env, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "count": len(results)})
}

func (h *ConfigHandler) ListNamespaces(c *gin.Context) {
	result, err := h.service.ListNamespaces(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list namespaces")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list namespaces"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ConfigHandler) SearchConfigs(c *gin.Context) {
	query := c.Query("q")
	namespace := c.Query("namespace")
	env := c.Query("environment")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	results, total, err := h.service.SearchConfigs(c.Request.Context(), query, namespace, env, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *ConfigHandler) BulkGet(c *gin.Context) {
	var req models.BulkGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env := c.DefaultQuery("environment", "all")
	tenantID := c.Query("tenant_id")

	results, err := h.service.BulkGet(c.Request.Context(), &req, env, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to bulk get configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to bulk get configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "count": len(results)})
}

func (h *ConfigHandler) BulkSet(c *gin.Context) {
	var req models.BulkSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.service.BulkSet(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to bulk set configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to bulk set configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "count": len(results)})
}

func (h *ConfigHandler) ExportNamespace(c *gin.Context) {
	namespace := c.Param("namespace")
	env := c.DefaultQuery("environment", "")
	tenantID := c.Query("tenant_id")

	results, err := h.service.ExportNamespace(c.Request.Context(), namespace, env, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to export namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"namespace": namespace, "data": results, "count": len(results)})
}

func (h *ConfigHandler) GetAuditLog(c *gin.Context) {
	namespace := c.Query("namespace")
	key := c.Query("key")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	results, total, err := h.service.GetAuditLog(c.Request.Context(), namespace, key, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit log")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit log"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *ConfigHandler) GetConfigHistory(c *gin.Context) {
	configID := c.Param("configId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	results, total, err := h.service.GetConfigHistory(c.Request.Context(), configID, page, pageSize)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get config history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get config history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
