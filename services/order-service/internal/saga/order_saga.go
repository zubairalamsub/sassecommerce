package saga

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/ecommerce/order-service/internal/domain/aggregates"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"go.uber.org/zap"
)

// SagaStep represents a step in the saga
type SagaStep interface {
	Execute() error
	Compensate() error
	GetName() string
}

// OrderSaga orchestrates a distributed transaction for order processing
type OrderSaga struct {
	orderID        string
	order          *aggregates.Order
	commandHandler *commands.CommandHandler
	logger         *zap.Logger
	inventoryURL   string
	paymentURL     string
	steps          []SagaStep
	completedSteps []SagaStep
}

// NewOrderSaga creates a new order saga
func NewOrderSaga(
	orderID string,
	order *aggregates.Order,
	commandHandler *commands.CommandHandler,
	logger *zap.Logger,
	inventoryURL string,
	paymentURL string,
) *OrderSaga {
	return &OrderSaga{
		orderID:        orderID,
		order:          order,
		commandHandler: commandHandler,
		logger:         logger,
		inventoryURL:   inventoryURL,
		paymentURL:     paymentURL,
		steps:          make([]SagaStep, 0),
		completedSteps: make([]SagaStep, 0),
	}
}

// Execute runs the saga
func (s *OrderSaga) Execute() error {
	s.logger.Info("Starting order saga", zap.String("order_id", s.orderID))

	// Define saga steps
	s.steps = []SagaStep{
		s.NewReserveInventoryStep(),
		s.NewProcessPaymentStep(),
		s.NewConfirmOrderStep(),
	}

	// Execute steps
	for _, step := range s.steps {
		s.logger.Info("Executing saga step",
			zap.String("order_id", s.orderID),
			zap.String("step", step.GetName()),
		)

		if err := step.Execute(); err != nil {
			s.logger.Error("Saga step failed",
				zap.String("order_id", s.orderID),
				zap.String("step", step.GetName()),
				zap.Error(err),
			)

			// Compensate completed steps
			if compErr := s.compensate(); compErr != nil {
				s.logger.Error("Compensation failed",
					zap.String("order_id", s.orderID),
					zap.Error(compErr),
				)
			}

			return fmt.Errorf("saga step %s failed: %w", step.GetName(), err)
		}

		s.completedSteps = append(s.completedSteps, step)
	}

	s.logger.Info("Order saga completed successfully", zap.String("order_id", s.orderID))
	return nil
}

// compensate runs compensation for completed steps in reverse order
func (s *OrderSaga) compensate() error {
	s.logger.Warn("Starting saga compensation", zap.String("order_id", s.orderID))

	// Compensate in reverse order
	for i := len(s.completedSteps) - 1; i >= 0; i-- {
		step := s.completedSteps[i]

		s.logger.Info("Compensating saga step",
			zap.String("order_id", s.orderID),
			zap.String("step", step.GetName()),
		)

		if err := step.Compensate(); err != nil {
			s.logger.Error("Compensation step failed",
				zap.String("order_id", s.orderID),
				zap.String("step", step.GetName()),
				zap.Error(err),
			)
			return err
		}
	}

	// Cancel the order
	cancelCmd := commands.CancelOrderCommand{
		OrderID:     s.orderID,
		Reason:      "Saga compensation",
		CancelledBy: "system",
	}

	if err := s.commandHandler.Handle(cancelCmd); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	s.logger.Info("Saga compensation completed", zap.String("order_id", s.orderID))
	return nil
}

// Reserve Inventory Step
type ReserveInventoryStep struct {
	saga          *OrderSaga
	reservationID string
}

func (s *OrderSaga) NewReserveInventoryStep() *ReserveInventoryStep {
	return &ReserveInventoryStep{saga: s}
}

func (step *ReserveInventoryStep) GetName() string {
	return "ReserveInventory"
}

func (step *ReserveInventoryStep) Execute() error {
	// Build reservation request
	items := make([]map[string]interface{}, 0, len(step.saga.order.Items))
	for _, item := range step.saga.order.Items {
		items = append(items, map[string]interface{}{
			"product_id": item.ProductID,
			"variant_id": item.VariantID,
			"quantity":   item.Quantity,
		})
	}

	request := map[string]interface{}{
		"tenant_id": step.saga.order.TenantID,
		"order_id":  step.saga.orderID,
		"items":     items,
	}

	// Call Inventory Service
	response, err := step.callInventoryService("/api/v1/inventory/reservations", request)
	if err != nil {
		return fmt.Errorf("failed to reserve inventory: %w", err)
	}

	reservationID, ok := response["id"].(string)
	if !ok {
		return fmt.Errorf("invalid reservation response")
	}

	step.reservationID = reservationID

	// Record reservation in order
	reservedItems := make([]events.ReservedItem, 0, len(step.saga.order.Items))
	for _, item := range step.saga.order.Items {
		reservedItems = append(reservedItems, events.ReservedItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		})
	}

	if err := step.saga.order.RecordInventoryReservation(reservationID, reservedItems); err != nil {
		return err
	}

	// Save events
	uncommittedEvents := step.saga.order.GetUncommittedEvents()
	if len(uncommittedEvents) > 0 {
		// Note: In production, this would go through command handler
		step.saga.order.MarkEventsAsCommitted()
	}

	return nil
}

func (step *ReserveInventoryStep) Compensate() error {
	if step.reservationID == "" {
		return nil
	}

	// Cancel reservation
	request := map[string]interface{}{
		"cancelled_by": "system",
		"reason":       "Order saga compensation",
	}

	_, err := step.callInventoryService(
		fmt.Sprintf("/api/v1/inventory/reservations/%s/cancel", step.reservationID),
		request,
	)
	if err != nil {
		return fmt.Errorf("failed to cancel inventory reservation: %w", err)
	}

	// Record release in order
	if err := step.saga.order.RecordInventoryRelease(step.reservationID, "Saga compensation"); err != nil {
		return err
	}

	step.saga.order.MarkEventsAsCommitted()
	return nil
}

func (step *ReserveInventoryStep) callInventoryService(path string, request map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := step.saga.inventoryURL + path
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("inventory service returned status %d", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}

// Process Payment Step
type ProcessPaymentStep struct {
	saga      *OrderSaga
	paymentID string
}

func (s *OrderSaga) NewProcessPaymentStep() *ProcessPaymentStep {
	return &ProcessPaymentStep{saga: s}
}

func (step *ProcessPaymentStep) GetName() string {
	return "ProcessPayment"
}

func (step *ProcessPaymentStep) Execute() error {
	// Build payment request
	request := map[string]interface{}{
		"tenant_id":   step.saga.order.TenantID,
		"customer_id": step.saga.order.CustomerID,
		"order_id":    step.saga.orderID,
		"amount":      step.saga.order.TotalAmount,
		"currency":    step.saga.order.Currency,
		"method":      "credit_card", // This would come from order data
	}

	// Call Payment Service
	response, err := step.callPaymentService("/api/v1/payments", request)
	if err != nil {
		return fmt.Errorf("failed to process payment: %w", err)
	}

	paymentID, ok := response["id"].(string)
	if !ok {
		return fmt.Errorf("invalid payment response")
	}

	step.paymentID = paymentID

	// Record payment in order
	if err := step.saga.order.RecordPayment(
		paymentID,
		"credit_card",
		fmt.Sprintf("txn_%s", paymentID),
		step.saga.order.TotalAmount,
	); err != nil {
		return err
	}

	step.saga.order.MarkEventsAsCommitted()
	return nil
}

func (step *ProcessPaymentStep) Compensate() error {
	if step.paymentID == "" {
		return nil
	}

	// Refund payment
	request := map[string]interface{}{
		"reason": "Order saga compensation",
	}

	_, err := step.callPaymentService(
		fmt.Sprintf("/api/v1/payments/%s/refund", step.paymentID),
		request,
	)
	if err != nil {
		return fmt.Errorf("failed to refund payment: %w", err)
	}

	// Record payment failure in order
	if err := step.saga.order.RecordPaymentFailure(step.paymentID, "Refunded due to saga compensation"); err != nil {
		return err
	}

	step.saga.order.MarkEventsAsCommitted()
	return nil
}

func (step *ProcessPaymentStep) callPaymentService(path string, request map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := step.saga.paymentURL + path
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}

// Confirm Order Step
type ConfirmOrderStep struct {
	saga *OrderSaga
}

func (s *OrderSaga) NewConfirmOrderStep() *ConfirmOrderStep {
	return &ConfirmOrderStep{saga: s}
}

func (step *ConfirmOrderStep) GetName() string {
	return "ConfirmOrder"
}

func (step *ConfirmOrderStep) Execute() error {
	cmd := commands.ConfirmOrderCommand{
		OrderID:     step.saga.orderID,
		ConfirmedBy: "system",
	}

	if err := step.saga.commandHandler.Handle(cmd); err != nil {
		return fmt.Errorf("failed to confirm order: %w", err)
	}

	return nil
}

func (step *ConfirmOrderStep) Compensate() error {
	// Order will be cancelled by the main compensation logic
	return nil
}
