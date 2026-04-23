package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/ecommerce/order-service/internal/projection"
	"go.uber.org/zap"
)

// QueryHandler handles HTTP requests for queries
type QueryHandler struct {
	projection *projection.OrderProjection
	logger     *zap.Logger
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(projection *projection.OrderProjection, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		projection: projection,
		logger:     logger,
	}
}

// GetOrder handles GET /api/v1/orders/:id
func (h *QueryHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	// Get order from read model
	order, err := h.projection.GetOrder(orderID)
	if err != nil {
		h.logger.Error("Failed to get order", zap.String("order_id", orderID), zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "order_not_found",
			Message: err.Error(),
		})
		return
	}

	// Get order items
	items, err := h.projection.GetOrderItems(orderID)
	if err != nil {
		h.logger.Error("Failed to get order items", zap.String("order_id", orderID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed_to_get_items",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := map[string]interface{}{
		"order": order,
		"items": items,
	}

	c.JSON(http.StatusOK, response)
}

// GetOrdersByCustomer handles GET /api/v1/customers/:customerId/orders
func (h *QueryHandler) GetOrdersByCustomer(c *gin.Context) {
	customerID := c.Param("customerId")

	// Parse pagination parameters
	limit := h.getIntQueryParam(c, "limit", 10)
	offset := h.getIntQueryParam(c, "offset", 0)

	// Get orders
	orders, err := h.projection.GetOrdersByCustomer(customerID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get orders by customer",
			zap.String("customer_id", customerID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed_to_get_orders",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := map[string]interface{}{
		"orders": orders,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"count":  len(orders),
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetOrdersByTenant handles GET /api/v1/tenants/:tenantId/orders
func (h *QueryHandler) GetOrdersByTenant(c *gin.Context) {
	tenantID := c.Param("tenantId")

	// Parse pagination parameters
	limit := h.getIntQueryParam(c, "limit", 10)
	offset := h.getIntQueryParam(c, "offset", 0)

	// Get orders
	orders, err := h.projection.GetOrdersByTenant(tenantID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get orders by tenant",
			zap.String("tenant_id", tenantID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "failed_to_get_orders",
			Message: err.Error(),
		})
		return
	}

	// Build response
	response := map[string]interface{}{
		"orders": orders,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"count":  len(orders),
		},
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions

func (h *QueryHandler) getIntQueryParam(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
