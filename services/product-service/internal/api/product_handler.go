package api

import (
	"net/http"
	"strconv"

	"github.com/ecommerce/product-service/internal/models"
	"github.com/ecommerce/product-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ProductHandler struct {
	productService service.ProductService
	logger         *logrus.Logger
}

func NewProductHandler(productService service.ProductService, logger *logrus.Logger) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         logger,
	}
}

// CreateProduct handles product creation
// @Summary Create a new product
// @Tags Products
// @Accept json
// @Produce json
// @Param product body models.CreateProductRequest true "Product data"
// @Success 201 {object} models.ProductResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	product, err := h.productService.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create product")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// GetProduct handles retrieving a product by ID
// @Summary Get product by ID
// @Tags Products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} models.ProductResponse
// @Failure 404 {object} map[string]string
// @Router /products/{id} [get]
func (h *ProductHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")

	product, err := h.productService.GetProductByID(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).WithField("product_id", id).Error("Failed to get product")
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// GetProductBySKU handles retrieving a product by SKU
// @Summary Get product by SKU
// @Tags Products
// @Produce json
// @Param sku path string true "Product SKU"
// @Success 200 {object} models.ProductResponse
// @Failure 404 {object} map[string]string
// @Router /products/sku/{sku} [get]
func (h *ProductHandler) GetProductBySKU(c *gin.Context) {
	sku := c.Param("sku")
	tenantID := c.GetString("tenant_id")

	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	product, err := h.productService.GetProductBySKU(c.Request.Context(), tenantID, sku)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"tenant_id": tenantID,
			"sku":       sku,
		}).Error("Failed to get product by SKU")
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

// ListProducts handles listing products with pagination
// @Summary List products
// @Tags Products
// @Produce json
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	products, total, err := h.productService.ListProducts(c.Request.Context(), tenantID, offset, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

// ListProductsByCategory handles listing products by category
// @Summary List products by category
// @Tags Products
// @Produce json
// @Param category_id path string true "Category ID"
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /products/category/{category_id} [get]
func (h *ProductHandler) ListProductsByCategory(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	categoryID := c.Param("category_id")
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	products, total, err := h.productService.ListProductsByCategory(c.Request.Context(), tenantID, categoryID, offset, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list products by category")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

// SearchProducts handles product search
// @Summary Search products
// @Tags Products
// @Produce json
// @Param q query string true "Search query"
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /products/search [get]
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in context"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	products, total, err := h.productService.SearchProducts(c.Request.Context(), tenantID, query, offset, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search products")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"offset": offset,
		"limit":  limit,
		"query":  query,
	})
}

// UpdateProduct handles product updates
// @Summary Update a product
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param product body models.UpdateProductRequest true "Product update data"
// @Success 200 {object} models.ProductResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	product, err := h.productService.UpdateProduct(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithError(err).WithField("product_id", id).Error("Failed to update product")
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, product)
}

// DeleteProduct handles product deletion (soft delete)
// @Summary Delete a product
// @Tags Products
// @Param id path string true "Product ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	if err := h.productService.DeleteProduct(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).WithField("product_id", id).Error("Failed to delete product")
		if err.Error() == "failed to delete product" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateProductStatus handles product status updates
// @Summary Update product status
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param status body map[string]string true "Status update"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /products/{id}/status [patch]
func (h *ProductHandler) UpdateProductStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status models.ProductStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate status
	validStatuses := []models.ProductStatus{
		models.ProductStatusDraft,
		models.ProductStatusActive,
		models.ProductStatusInactive,
		models.ProductStatusArchived,
	}
	valid := false
	for _, s := range validStatuses {
		if req.Status == s {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	if err := h.productService.UpdateProductStatus(c.Request.Context(), id, req.Status); err != nil {
		h.logger.WithError(err).WithField("product_id", id).Error("Failed to update product status")
		if err.Error() == "failed to update product status" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product status updated successfully"})
}

// RegisterRoutes registers all product routes
func (h *ProductHandler) RegisterRoutes(router *gin.RouterGroup) {
	products := router.Group("/products")
	{
		products.POST("", h.CreateProduct)
		products.GET("", h.ListProducts)
		products.GET("/search", h.SearchProducts)
		products.GET("/sku/:sku", h.GetProductBySKU)
		products.GET("/category/:category_id", h.ListProductsByCategory)
		products.GET("/:id", h.GetProduct)
		products.PUT("/:id", h.UpdateProduct)
		products.DELETE("/:id", h.DeleteProduct)
		products.PATCH("/:id/status", h.UpdateProductStatus)
	}
}
