package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// publicNamespaces are the config namespaces whose values are inherently public
// (rendered on the anonymous storefront), so their per-tenant reads may be
// served without authentication as long as the tenant is supplied. Every other
// namespace requires an authenticated request and is scoped to the JWT tenant.
var publicNamespaces = map[string]bool{
	"storefront":        true,
	"delivery_profiles": true,
}

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

func RegisterRoutes(router *gin.Engine, handler *ConfigHandler, authMiddleware ...gin.HandlerFunc) {
	// Public read endpoints — limited to inherently-public storefront config.
	// The handler further restricts these to the publicNamespaces allowlist and
	// requires a tenant_id query param, so no cross-tenant secrets can leak.
	pub := router.Group("/api/v1/config")
	{
		pub.GET("/namespace/:namespace", handler.ListByNamespace)
	}

	// Protected endpoints (require auth). Tenant identity is always taken from
	// the verified JWT, never from client-supplied query/body parameters.
	prot := router.Group("/api/v1/config")
	prot.Use(authMiddleware...)
	{
		prot.GET("/get", handler.GetConfig)
		prot.GET("/namespaces", handler.ListNamespaces)
		prot.GET("/search", handler.SearchConfigs)
		prot.POST("/bulk/get", handler.BulkGet)
		prot.GET("/export/:namespace", handler.ExportNamespace)
		prot.GET("/audit", handler.GetAuditLog)
		prot.GET("/audit/:configId", handler.GetConfigHistory)

		prot.POST("/set", handler.SetConfig)
		prot.DELETE("/:id", handler.DeleteConfig)
		prot.POST("/bulk/set", handler.BulkSet)
	}
}

func (h *ConfigHandler) GetConfig(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	namespace := c.Query("namespace")
	key := c.Query("key")
	if namespace == "" || key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "namespace and key are required"})
		return
	}

	env := c.DefaultQuery("environment", "all")

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
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.SetConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Tenant is always the authenticated tenant, never a client-supplied value,
	// so a caller cannot write into another tenant's config namespace.
	req.TenantID = tenantID

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
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")

	err := h.service.DeleteConfig(c.Request.Context(), id, tenantID)
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

	// This is a public endpoint: only inherently-public storefront namespaces
	// may be read anonymously. Anything else must go through an authenticated
	// path so tenant config (and secrets) is never exposed.
	if !publicNamespaces[namespace] {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Anonymous storefront requests carry no JWT, so the tenant is supplied
	// explicitly and is required. The repository scopes results to this tenant
	// (plus shared globals) and secret values are masked in the response.
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	env := c.DefaultQuery("environment", "")

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
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	query := c.Query("q")
	namespace := c.Query("namespace")
	env := c.Query("environment")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	results, total, err := h.service.SearchConfigs(c.Request.Context(), query, namespace, env, tenantID, page, pageSize)
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
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.BulkGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	env := c.DefaultQuery("environment", "all")

	results, err := h.service.BulkGet(c.Request.Context(), &req, env, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to bulk get configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to bulk get configs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results, "count": len(results)})
}

func (h *ConfigHandler) BulkSet(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req models.BulkSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force every entry to the authenticated tenant so a bulk write cannot span
	// or target another tenant's namespace.
	for i := range req.Entries {
		req.Entries[i].TenantID = tenantID
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
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	namespace := c.Param("namespace")
	env := c.DefaultQuery("environment", "")

	results, err := h.service.ExportNamespace(c.Request.Context(), namespace, env, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to export namespace")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export namespace"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"namespace": namespace, "data": results, "count": len(results)})
}

func (h *ConfigHandler) GetAuditLog(c *gin.Context) {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	namespace := c.Query("namespace")
	key := c.Query("key")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	results, total, err := h.service.GetAuditLog(c.Request.Context(), namespace, key, tenantID, page, pageSize)
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
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	configID := c.Param("configId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	results, total, err := h.service.GetConfigHistory(c.Request.Context(), configID, tenantID, page, pageSize)
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
