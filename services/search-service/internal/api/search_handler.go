package api

import (
	"net/http"

	"github.com/ecommerce/search-service/internal/models"
	"github.com/ecommerce/search-service/internal/service"
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
		v1.POST("/reindex", handler.ReindexProduct)
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
	var product models.ProductDocument
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if product.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "product id is required"})
		return
	}

	if product.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	if err := h.service.IndexProduct(c.Request.Context(), &product); err != nil {
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
