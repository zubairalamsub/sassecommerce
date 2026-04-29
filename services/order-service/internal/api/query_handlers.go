package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/domain/queries"
	"github.com/yourusername/ecommerce/order-service/internal/eventstore"
	"github.com/yourusername/ecommerce/order-service/internal/projection"
	"go.uber.org/zap"
)

// QueryHandler handles HTTP requests for queries
type QueryHandler struct {
	projection *projection.OrderProjection
	eventStore eventstore.EventStore
	logger     *zap.Logger
}

// NewQueryHandler creates a new query handler
func NewQueryHandler(projection *projection.OrderProjection, eventStore eventstore.EventStore, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		projection: projection,
		eventStore: eventStore,
		logger:     logger,
	}
}

// GetOrder handles GET /api/v1/orders/:id
func (h *QueryHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	// Get order from read model
	order, err := h.projection.GetOrder(orderID)
	if err != nil {
		// Projection miss — try rebuilding from event store
		rebuilt, items := h.rebuildFromEvents(orderID)
		if rebuilt == nil {
			h.logger.Error("Failed to get order", zap.String("order_id", orderID), zap.Error(err))
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "order_not_found",
				Message: err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, map[string]interface{}{"order": rebuilt, "items": items})
		return
	}

	// Get order items
	items, err := h.projection.GetOrderItems(orderID)
	if err != nil {
		h.logger.Error("Failed to get order items", zap.String("order_id", orderID), zap.Error(err))
		items = make([]*queries.OrderItemReadModel, 0)
	}

	// If projection has no items but events might, rebuild from event store
	if len(items) == 0 || order.TotalAmount == 0 {
		rebuilt, rebuiltItems := h.rebuildFromEvents(orderID)
		if rebuilt != nil && (len(rebuiltItems) > len(items) || rebuilt.TotalAmount > order.TotalAmount) {
			order = rebuilt
			items = rebuiltItems
		}
	}

	// Build response
	response := map[string]interface{}{
		"order": order,
		"items": items,
	}

	c.JSON(http.StatusOK, response)
}

// rebuildFromEvents reconstructs order data from the event store and updates the projection
func (h *QueryHandler) rebuildFromEvents(orderID string) (*queries.OrderReadModel, []*queries.OrderItemReadModel) {
	eventHistory, err := h.eventStore.GetEvents(orderID)
	if err != nil || len(eventHistory) == 0 {
		return nil, nil
	}

	// Replay events on aggregate
	order := &aggregates.Order{
		ID:    orderID,
		Items: make(map[string]*aggregates.OrderItem),
	}
	order.LoadFromHistory(eventHistory)

	// Re-project all events to fix the read model
	for _, event := range eventHistory {
		if projErr := h.projection.Project(event); projErr != nil {
			h.logger.Debug("Re-projection event skipped (likely already exists)",
				zap.String("order_id", orderID),
				zap.Error(projErr),
			)
		}
	}

	// Build response from aggregate state
	readModel := &queries.OrderReadModel{
		ID:          order.ID,
		TenantID:    order.TenantID,
		CustomerID:  order.CustomerID,
		Status:      string(order.Status),
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
		ShippingAddress: queries.Address{
			Street:     order.ShippingAddress.Street,
			City:       order.ShippingAddress.City,
			State:      order.ShippingAddress.State,
			PostalCode: order.ShippingAddress.PostalCode,
			Country:    order.ShippingAddress.Country,
		},
		BillingAddress: queries.Address{
			Street:     order.BillingAddress.Street,
			City:       order.BillingAddress.City,
			State:      order.BillingAddress.State,
			PostalCode: order.BillingAddress.PostalCode,
			Country:    order.BillingAddress.Country,
		},
		PaymentID:      order.PaymentID,
		ReservationID:  order.ReservationID,
		TrackingNumber: order.TrackingNumber,
		Carrier:        order.Carrier,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		Version:        order.Version,
	}

	items := make([]*queries.OrderItemReadModel, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, &queries.OrderItemReadModel{
			ID:         item.ID,
			OrderID:    orderID,
			ProductID:  item.ProductID,
			VariantID:  item.VariantID,
			SKU:        item.SKU,
			Name:       item.Name,
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			TotalPrice: item.TotalPrice,
		})
	}

	return readModel, items
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
