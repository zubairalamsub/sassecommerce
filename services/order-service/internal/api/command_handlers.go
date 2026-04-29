package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/eventstore"
	"github.com/yourusername/ecommerce/order-service/internal/saga"
	"go.uber.org/zap"
)

// CommandHandler handles HTTP requests for commands
type CommandHandler struct {
	commandHandler *commands.CommandHandler
	eventStore     eventstore.EventStore
	logger         *zap.Logger
	inventoryURL   string
	paymentURL     string
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(
	commandHandler *commands.CommandHandler,
	eventStore eventstore.EventStore,
	logger *zap.Logger,
	inventoryURL string,
	paymentURL string,
) *CommandHandler {
	return &CommandHandler{
		commandHandler: commandHandler,
		eventStore:     eventStore,
		logger:         logger,
		inventoryURL:   inventoryURL,
		paymentURL:     paymentURL,
	}
}

// CreateOrder handles POST /api/v1/orders
func (h *CommandHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Generate order ID
	orderID := uuid.New().String()

	// For guest checkout: generate a guest customer ID if none provided
	customerID := req.CustomerID
	if customerID == "" {
		if req.GuestEmail == "" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_request",
				Message: "Either customer_id or guest_email is required",
			})
			return
		}
		customerID = "guest-" + uuid.New().String()
	}

	// Create command
	cmd := commands.CreateOrderCommand{
		OrderID:    orderID,
		TenantID:   req.TenantID,
		CustomerID: customerID,
		GuestEmail: req.GuestEmail,
		GuestName:  req.GuestName,
		GuestPhone: req.GuestPhone,
		ShippingAddress: events.Address{
			Street:     req.ShippingAddress.Street,
			City:       req.ShippingAddress.City,
			State:      req.ShippingAddress.State,
			PostalCode: req.ShippingAddress.PostalCode,
			Country:    req.ShippingAddress.Country,
		},
		BillingAddress: events.Address{
			Street:     req.BillingAddress.Street,
			City:       req.BillingAddress.City,
			State:      req.BillingAddress.State,
			PostalCode: req.BillingAddress.PostalCode,
			Country:    req.BillingAddress.Country,
		},
	}

	// Handle command
	if err := h.commandHandler.Handle(cmd); err != nil {
		h.logger.Error("Failed to create order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "order_creation_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order created successfully", zap.String("order_id", orderID))

	c.JSON(http.StatusCreated, CreateOrderResponse{
		OrderID: orderID,
		Message: "Order created successfully",
	})
}

// AddOrderItem handles POST /api/v1/orders/:id/items
func (h *CommandHandler) AddOrderItem(c *gin.Context) {
	orderID := c.Param("id")

	var req AddOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Create command
	cmd := commands.AddOrderItemCommand{
		OrderID:   orderID,
		ProductID: req.ProductID,
		VariantID: req.VariantID,
		SKU:       req.SKU,
		Name:      req.Name,
		Quantity:  req.Quantity,
		UnitPrice: req.UnitPrice,
	}

	// Handle command
	if err := h.commandHandler.Handle(cmd); err != nil {
		h.logger.Error("Failed to add order item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "add_item_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order item added", zap.String("order_id", orderID))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Order item added successfully",
	})
}

// RemoveOrderItem handles DELETE /api/v1/orders/:id/items/:itemId
func (h *CommandHandler) RemoveOrderItem(c *gin.Context) {
	orderID := c.Param("id")
	itemID := c.Param("itemId")

	// Create command
	cmd := commands.RemoveOrderItemCommand{
		OrderID: orderID,
		ItemID:  itemID,
	}

	// Handle command
	if err := h.commandHandler.Handle(cmd); err != nil {
		h.logger.Error("Failed to remove order item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "remove_item_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order item removed", zap.String("order_id", orderID), zap.String("item_id", itemID))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Order item removed successfully",
	})
}

// ConfirmOrder handles POST /api/v1/orders/:id/confirm
func (h *CommandHandler) ConfirmOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req ConfirmOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Load order for saga
	order, err := h.loadOrder(orderID)
	if err != nil {
		h.logger.Error("Failed to load order", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "order_not_found",
			Message: err.Error(),
		})
		return
	}

	// Extract auth token from incoming request to forward to downstream services
	authToken := ""
	if authHeader := c.GetHeader("Authorization"); len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		authToken = authHeader[7:]
	}

	// Execute saga for order confirmation
	orderSaga := saga.NewOrderSaga(
		orderID,
		order,
		h.commandHandler,
		h.logger,
		h.inventoryURL,
		h.paymentURL,
		authToken,
	)

	if err := orderSaga.Execute(); err != nil {
		h.logger.Error("Order saga failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "order_confirmation_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order confirmed successfully", zap.String("order_id", orderID))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Order confirmed successfully",
	})
}

// CancelOrder handles POST /api/v1/orders/:id/cancel
func (h *CommandHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Create command
	cmd := commands.CancelOrderCommand{
		OrderID:     orderID,
		Reason:      req.Reason,
		CancelledBy: req.CancelledBy,
	}

	// Handle command
	if err := h.commandHandler.Handle(cmd); err != nil {
		h.logger.Error("Failed to cancel order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "cancel_order_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order cancelled", zap.String("order_id", orderID))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Order cancelled successfully",
	})
}

// ShipOrder handles POST /api/v1/orders/:id/ship
func (h *CommandHandler) ShipOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req ShipOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Create command
	cmd := commands.ShipOrderCommand{
		OrderID:        orderID,
		TrackingNumber: req.TrackingNumber,
		Carrier:        req.Carrier,
		ShippedBy:      req.ShippedBy,
	}

	// Handle command
	if err := h.commandHandler.Handle(cmd); err != nil {
		h.logger.Error("Failed to ship order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "ship_order_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order shipped", zap.String("order_id", orderID))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Order shipped successfully",
	})
}

// DeliverOrder handles POST /api/v1/orders/:id/deliver
func (h *CommandHandler) DeliverOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req DeliverOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Create command
	cmd := commands.DeliverOrderCommand{
		OrderID:    orderID,
		ReceivedBy: req.ReceivedBy,
	}

	// Handle command
	if err := h.commandHandler.Handle(cmd); err != nil {
		h.logger.Error("Failed to deliver order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "deliver_order_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Order delivered", zap.String("order_id", orderID))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Order delivered successfully",
	})
}

// Helper functions

func (h *CommandHandler) loadOrder(orderID string) (*aggregates.Order, error) {
	eventsHistory, err := h.eventStore.GetEvents(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if len(eventsHistory) == 0 {
		return nil, errors.New("order not found")
	}

	order := &aggregates.Order{
		ID:    orderID,
		Items: make(map[string]*aggregates.OrderItem),
	}
	order.LoadFromHistory(eventsHistory)

	return order, nil
}
