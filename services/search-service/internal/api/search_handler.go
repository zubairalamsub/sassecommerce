package api

import (
	"errors"
	"net/http"

	"github.com/ecommerce/search-service/internal/models"
	"github.com/ecommerce/search-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type SearchHandler struct {
	service service.SearchService
	logger  *logrus.Logger
}

func NewSearchHandler(service service.SearchService, logger *logrus.Logger) *SearchHandler {
	return &SearchHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterRoutes(router *gin.Engine, handler *SearchHandler) {
	v1 := router.Group("/api/v1/search")
	{
		v1.GET("/products", handler.SearchProducts)
		v1.GET("/autocomplete", handler.Autocomplete)
		// Reindex mutates the shared search index; restrict to staff roles.
		// Tenant identity is taken from the JWT inside the handler, never the body.
		v1.POST("/reindex",
			sharedmiddleware.RequireRole("super_admin", "admin", "moderator"),
			handler.ReindexProduct,
		)
	}
}

func (h *SearchHandler) SearchProducts(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse tags from comma-separated query param
	if tagsStr := c.Query("tags"); tagsStr != "" {
		req.Tags = splitTags(tagsStr)
	}

	result, err := h.service.Search(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Search failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *SearchHandler) Autocomplete(c *gin.Context) {
	var req models.AutocompleteRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.Autocomplete(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Autocomplete failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "autocomplete failed"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *SearchHandler) ReindexProduct(c *gin.Context) {
	// Tenant is always derived from the verified JWT, never from the request
	// body, so a caller cannot pollute another tenant's search documents.
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in token"})
		return
	}

	var req models.ReindexProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := req.ProductDocument
	// Force the indexed doc's tenant to the caller's JWT tenant.
	product.TenantID = tenantID

	if product.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id is required"})
		return
	}

	if err := h.service.IndexProduct(c.Request.Context(), &product); err != nil {
		if errors.Is(err, service.ErrTenantMismatch) {
			// The document exists under a different tenant — do not disclose it.
			h.logger.WithError(err).Warn("Reindex blocked: cross-tenant product")
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to reindex product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reindex product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product indexed successfully", "product_id": product.ID})
}

func splitTags(s string) []string {
	tags := make([]string, 0)
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				tags = append(tags, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		tags = append(tags, current)
	}
	return tags
}
