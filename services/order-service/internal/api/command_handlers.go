package api

import (
	"errors"
	"fmt"
	"net/http"

	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/eventstore"
	"github.com/yourusername/ecommerce/order-service/internal/messaging"
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
	// notificationPublisher is optional — when nil, /send-receipt returns 503.
	// This lets local dev / tests run without Kafka while still wiring the
	// route into the router.
	notificationPublisher messaging.NotificationPublisher
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

// SetNotificationPublisher injects the Kafka notification publisher used by
// SendReceipt. Wired separately from the constructor so the publisher can
// stay optional — tests and Kafka-disabled deployments simply skip it.
func (h *CommandHandler) SetNotificationPublisher(p messaging.NotificationPublisher) {
	h.notificationPublisher = p
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

	// Enforce tenant isolation: the order must belong to the caller's tenant.
	if order := h.authorizeOrderAccess(c, orderID); order == nil {
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

	// Enforce tenant isolation: the order must belong to the caller's tenant.
	if order := h.authorizeOrderAccess(c, orderID); order == nil {
		return
	}

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

	// Load order for saga, enforcing tenant isolation on the loaded aggregate.
	order := h.authorizeOrderAccess(c, orderID)
	if order == nil {
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

	// Enforce tenant isolation: the order must belong to the caller's tenant.
	if order := h.authorizeOrderAccess(c, orderID); order == nil {
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

	// Enforce tenant isolation: the order must belong to the caller's tenant.
	if order := h.authorizeOrderAccess(c, orderID); order == nil {
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

	// Enforce tenant isolation: the order must belong to the caller's tenant.
	if order := h.authorizeOrderAccess(c, orderID); order == nil {
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

// SendReceipt handles POST /api/v1/orders/:id/send-receipt.
//
// The endpoint is small on purpose: it publishes a `ReceiptRequested` event
// to Kafka with the order summary inline. The notification-service consumer
// renders and sends the email asynchronously. We deliberately do NOT block
// on email delivery — the cashier sees a fast acknowledgement.
//
// Request body: { "email": "customer@example.com" }
// The `email` field overrides whatever address is on file; it's required
// because the POS flow often serves walk-in customers without a stored
// profile.
func (h *CommandHandler) SendReceipt(c *gin.Context) {
	orderID := c.Param("id")

	var req SendReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if h.notificationPublisher == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "notifications_disabled",
			Message: "Notification publisher is not configured on this deployment",
		})
		return
	}

	// Load order summary from the event store, enforcing tenant isolation.
	// This is the highest-risk mutation: it emails order contents to a
	// caller-supplied address, so we must confirm the order belongs to the
	// caller's tenant before building the receipt. We replay events on the
	// aggregate so this works even when the read-model projection lags.
	order := h.authorizeOrderAccess(c, orderID)
	if order == nil {
		return
	}

	// Project the aggregate's item map into the wire shape. Iteration order
	// of a map is undefined in Go; for receipt rendering this is acceptable
	// (most POS sales have <10 items, line order is not load-bearing).
	items := make([]messaging.ReceiptRequestedItem, 0, len(order.Items))
	var subtotal float64
	for _, it := range order.Items {
		line := it.UnitPrice * float64(it.Quantity)
		subtotal += line
		items = append(items, messaging.ReceiptRequestedItem{
			Name:       it.Name,
			SKU:        it.SKU,
			Quantity:   it.Quantity,
			UnitPrice:  it.UnitPrice,
			TotalPrice: line,
		})
	}

	payload := messaging.ReceiptRequestedPayload{
		TenantID:      order.TenantID,
		OrderID:       order.ID,
		CustomerID:    order.CustomerID,
		CustomerEmail: req.Email,
		CustomerName:  req.CustomerName,
		StoreName:     req.StoreName,
		PaymentMethod: req.PaymentMethod,
		Currency:      "BDT",
		Subtotal:      subtotal,
		Total:         subtotal, // No discount/tax tracked on the aggregate yet.
		Items:         items,
	}

	if err := h.notificationPublisher.PublishReceiptRequested(c.Request.Context(), payload); err != nil {
		h.logger.Error("Failed to publish ReceiptRequested",
			zap.String("order_id", orderID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "publish_failed",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Receipt email queued",
		zap.String("order_id", orderID),
		zap.String("recipient", req.Email),
	)

	c.JSON(http.StatusOK, SuccessResponse{Message: "Receipt email queued"})
}

// Helper functions

// authorizeOrderAccess loads the order and verifies it belongs to the caller's
// tenant, taken from the verified JWT (never the path/body). It is the single
// tenant-isolation gate for every authenticated state mutation.
//
// On any failure — unauthenticated, order missing, or order owned by a
// different tenant — it writes a 404 (never 403) and returns nil. Returning
// 404 on a tenant mismatch is deliberate: a 403 would confirm the order exists,
// letting an attacker probe another tenant's order IDs.
func (h *CommandHandler) authorizeOrderAccess(c *gin.Context, orderID string) *aggregates.Order {
	tenantID := sharedmiddleware.GetTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "authentication required",
		})
		return nil
	}

	order, err := h.loadOrder(orderID)
	if err != nil || order.TenantID != tenantID {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "order_not_found",
			Message: "order not found",
		})
		return nil
	}

	return order
}

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
