package commands

import (
	"errors"
	"fmt"

	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/eventstore"
	"go.uber.org/zap"
)

// CommandHandler handles commands and coordinates with event store
type CommandHandler struct {
	eventStore eventstore.EventStore
	logger     *zap.Logger
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(eventStore eventstore.EventStore, logger *zap.Logger) *CommandHandler {
	return &CommandHandler{
		eventStore: eventStore,
		logger:     logger,
	}
}

// Handle dispatches commands to appropriate handlers
func (h *CommandHandler) Handle(cmd Command) error {
	switch c := cmd.(type) {
	case CreateOrderCommand:
		return h.handleCreateOrder(c)
	case AddOrderItemCommand:
		return h.handleAddOrderItem(c)
	case RemoveOrderItemCommand:
		return h.handleRemoveOrderItem(c)
	case ConfirmOrderCommand:
		return h.handleConfirmOrder(c)
	case CancelOrderCommand:
		return h.handleCancelOrder(c)
	case ShipOrderCommand:
		return h.handleShipOrder(c)
	case DeliverOrderCommand:
		return h.handleDeliverOrder(c)
	default:
		return errors.New("unknown command type")
	}
}

// handleCreateOrder creates a new order
func (h *CommandHandler) handleCreateOrder(cmd CreateOrderCommand) error {
	h.logger.Info("Handling CreateOrderCommand",
		zap.String("order_id", cmd.OrderID),
		zap.String("customer_id", cmd.CustomerID),
	)

	// Create new order aggregate
	order := aggregates.NewOrder(
		cmd.OrderID,
		cmd.TenantID,
		cmd.CustomerID,
		cmd.ShippingAddress,
		cmd.BillingAddress,
	)

	// Save events
	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, -1); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// handleAddOrderItem adds an item to an order
func (h *CommandHandler) handleAddOrderItem(cmd AddOrderItemCommand) error {
	h.logger.Info("Handling AddOrderItemCommand",
		zap.String("order_id", cmd.OrderID),
		zap.String("product_id", cmd.ProductID),
	)

	// Load order from event history
	order, err := h.loadOrder(cmd.OrderID)
	if err != nil {
		return err
	}

	// Execute command on aggregate
	if err := order.AddItem(
		cmd.ProductID,
		cmd.VariantID,
		cmd.SKU,
		cmd.Name,
		cmd.Quantity,
		cmd.UnitPrice,
	); err != nil {
		return err
	}

	// Save new events
	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, order.Version-len(events)); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// handleRemoveOrderItem removes an item from an order
func (h *CommandHandler) handleRemoveOrderItem(cmd RemoveOrderItemCommand) error {
	h.logger.Info("Handling RemoveOrderItemCommand",
		zap.String("order_id", cmd.OrderID),
		zap.String("item_id", cmd.ItemID),
	)

	order, err := h.loadOrder(cmd.OrderID)
	if err != nil {
		return err
	}

	if err := order.RemoveItem(cmd.ItemID); err != nil {
		return err
	}

	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, order.Version-len(events)); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// handleConfirmOrder confirms an order
func (h *CommandHandler) handleConfirmOrder(cmd ConfirmOrderCommand) error {
	h.logger.Info("Handling ConfirmOrderCommand",
		zap.String("order_id", cmd.OrderID),
	)

	order, err := h.loadOrder(cmd.OrderID)
	if err != nil {
		return err
	}

	if err := order.Confirm(cmd.ConfirmedBy); err != nil {
		return err
	}

	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, order.Version-len(events)); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// handleCancelOrder cancels an order
func (h *CommandHandler) handleCancelOrder(cmd CancelOrderCommand) error {
	h.logger.Info("Handling CancelOrderCommand",
		zap.String("order_id", cmd.OrderID),
		zap.String("reason", cmd.Reason),
	)

	order, err := h.loadOrder(cmd.OrderID)
	if err != nil {
		return err
	}

	if err := order.Cancel(cmd.Reason, cmd.CancelledBy); err != nil {
		return err
	}

	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, order.Version-len(events)); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// handleShipOrder marks an order as shipped
func (h *CommandHandler) handleShipOrder(cmd ShipOrderCommand) error {
	h.logger.Info("Handling ShipOrderCommand",
		zap.String("order_id", cmd.OrderID),
		zap.String("tracking_number", cmd.TrackingNumber),
	)

	order, err := h.loadOrder(cmd.OrderID)
	if err != nil {
		return err
	}

	if err := order.Ship(cmd.TrackingNumber, cmd.Carrier, cmd.ShippedBy); err != nil {
		return err
	}

	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, order.Version-len(events)); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// handleDeliverOrder marks an order as delivered
func (h *CommandHandler) handleDeliverOrder(cmd DeliverOrderCommand) error {
	h.logger.Info("Handling DeliverOrderCommand",
		zap.String("order_id", cmd.OrderID),
	)

	order, err := h.loadOrder(cmd.OrderID)
	if err != nil {
		return err
	}

	if err := order.Deliver(cmd.ReceivedBy); err != nil {
		return err
	}

	events := order.GetUncommittedEvents()
	if err := h.eventStore.Save(cmd.OrderID, events, order.Version-len(events)); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}

	order.MarkEventsAsCommitted()
	return nil
}

// loadOrder loads an order aggregate from event history
func (h *CommandHandler) loadOrder(orderID string) (*aggregates.Order, error) {
	events, err := h.eventStore.GetEvents(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 {
		return nil, errors.New("order not found")
	}

	order := &aggregates.Order{
		ID:    orderID,
		Items: make(map[string]*aggregates.OrderItem),
	}
	order.LoadFromHistory(events)

	return order, nil
}
