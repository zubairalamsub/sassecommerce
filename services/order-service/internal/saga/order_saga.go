package saga

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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

// sideEffectError marks a step failure where an external side effect may
// already exist — the payment was captured, or stock is held — even though the
// step did not complete. It covers three cases:
//
//   - recording the side effect on the order failed after the call succeeded
//   - a later item failed after earlier items were already reserved
//   - the dependency returned success but an unusable response, leaving the
//     outcome genuinely unknown
//
// Such a failure is never optional. "Optional" exists for a dependency that is
// unavailable or not applicable (no stock records yet, no payment needed for
// COD), where nothing was acquired and skipping is harmless. Once a side effect
// may exist, skipping would strand it, so a sideEffectError always fails the
// saga and triggers compensation.
type sideEffectError struct {
	err error
}

func (e *sideEffectError) Error() string { return e.err.Error() }
func (e *sideEffectError) Unwrap() error { return e.err }

// OrderSaga orchestrates a distributed transaction for order processing
type OrderSaga struct {
	orderID        string
	order          *aggregates.Order
	commandHandler *commands.CommandHandler
	logger         *zap.Logger
	inventoryURL   string
	paymentURL     string
	authToken      string
	steps          []SagaStep
	// attemptedSteps holds every step whose Execute was entered, in order. A
	// step can acquire an external side effect and then fail, so compensation
	// must consider attempted steps rather than only successful ones.
	// Compensate() is a no-op for a step that acquired nothing, which is what
	// makes tracking attempts safe.
	attemptedSteps []SagaStep
	optionalSteps  map[string]bool
}

// NewOrderSaga creates a new order saga
func NewOrderSaga(
	orderID string,
	order *aggregates.Order,
	commandHandler *commands.CommandHandler,
	logger *zap.Logger,
	inventoryURL string,
	paymentURL string,
	authToken string,
) *OrderSaga {
	return &OrderSaga{
		orderID:        orderID,
		order:          order,
		commandHandler: commandHandler,
		logger:         logger,
		inventoryURL:   inventoryURL,
		paymentURL:     paymentURL,
		authToken:      authToken,
		steps:          make([]SagaStep, 0),
		attemptedSteps: make([]SagaStep, 0),
	}
}

// Execute runs the saga
func (s *OrderSaga) Execute() error {
	s.logger.Info("Starting order saga", zap.String("order_id", s.orderID))

	// Define saga steps
	// Inventory reservation and payment are optional — skip if services return errors
	// (e.g., no stock records exist yet, or payment not required for COD orders)
	s.steps = []SagaStep{
		s.NewReserveInventoryStep(),
		s.NewProcessPaymentStep(),
		s.NewConfirmOrderStep(),
	}
	s.optionalSteps = map[string]bool{
		"ReserveInventory": true,
		"ProcessPayment":   true,
	}

	// Execute steps
	for _, step := range s.steps {
		s.logger.Info("Executing saga step",
			zap.String("order_id", s.orderID),
			zap.String("step", step.GetName()),
		)

		// Register the step for compensation before running it: it may capture a
		// payment or hold stock and then fail, and that side effect still has to
		// be unwound.
		s.attemptedSteps = append(s.attemptedSteps, step)

		if err := step.Execute(); err != nil {
			// An optional step may be skipped only when nothing was acquired.
			// A sideEffectError means an external side effect may already exist,
			// so it must be compensated rather than skipped.
			var sideErr *sideEffectError
			if s.optionalSteps[step.GetName()] && !errors.As(err, &sideErr) {
				s.logger.Warn("Optional saga step failed, skipping",
					zap.String("order_id", s.orderID),
					zap.String("step", step.GetName()),
					zap.Error(err),
				)
				continue
			}

			s.logger.Error("Saga step failed",
				zap.String("order_id", s.orderID),
				zap.String("step", step.GetName()),
				zap.Error(err),
			)

			// Compensate attempted steps
			if compErr := s.compensate(); compErr != nil {
				s.logger.Error("Compensation failed",
					zap.String("order_id", s.orderID),
					zap.Error(compErr),
				)
			}

			return fmt.Errorf("saga step %s failed: %w", step.GetName(), err)
		}
	}

	s.logger.Info("Order saga completed successfully", zap.String("order_id", s.orderID))
	return nil
}

// compensate runs compensation for attempted steps in reverse order
func (s *OrderSaga) compensate() error {
	s.logger.Warn("Starting saga compensation", zap.String("order_id", s.orderID))

	// Compensate in reverse order
	for i := len(s.attemptedSteps) - 1; i >= 0; i-- {
		step := s.attemptedSteps[i]

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
	saga *OrderSaga
	// reservationIDs holds every reservation created by this step, in creation
	// order — the inventory service issues one per order item. Compensation must
	// release all of them; tracking only the last one leaked the rest.
	reservationIDs []string
}

func (s *OrderSaga) NewReserveInventoryStep() *ReserveInventoryStep {
	return &ReserveInventoryStep{saga: s}
}

func (step *ReserveInventoryStep) GetName() string {
	return "ReserveInventory"
}

func (step *ReserveInventoryStep) Execute() error {
	// Reserve each item individually (inventory service expects single-item requests)
	for _, item := range step.saga.order.Items {
		request := map[string]interface{}{
			"tenantId":          step.saga.order.TenantID,
			"productId":         item.ProductID,
			"variantId":         item.VariantID,
			"quantity":          item.Quantity,
			"orderId":           step.saga.orderID,
			"orderItemId":       item.ProductID,
			"expirationMinutes": 30,
			"createdBy":         "system",
		}

		response, err := step.callInventoryService("/api/v1/inventory/reservations", request)
		if err != nil {
			// If earlier items are already reserved, stock is held and must be
			// released — this is no longer a skippable optional failure.
			if len(step.reservationIDs) > 0 {
				return &sideEffectError{fmt.Errorf(
					"failed to reserve inventory for product %s after %d reservation(s) succeeded: %w",
					item.ProductID, len(step.reservationIDs), err)}
			}
			return fmt.Errorf("failed to reserve inventory: %w", err)
		}

		// Record the id immediately, so a failure on a later item can still
		// release everything reserved so far.
		if id, ok := response["id"].(string); ok && id != "" {
			step.reservationIDs = append(step.reservationIDs, id)
		}
	}

	lastReservationID := ""
	if n := len(step.reservationIDs); n > 0 {
		lastReservationID = step.reservationIDs[n-1]
	}

	// Record reservation in order
	reservedItems := make([]events.ReservedItem, 0, len(step.saga.order.Items))
	for _, item := range step.saga.order.Items {
		reservedItems = append(reservedItems, events.ReservedItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		})
	}

	// Route through the command handler so the event is persisted to the event
	// store, projected to the read model and published — recording it on the
	// local aggregate alone would drop it.
	cmd := commands.RecordInventoryReservationCommand{
		OrderID:       step.saga.orderID,
		ReservationID: lastReservationID,
		Items:         reservedItems,
	}
	if err := step.saga.commandHandler.Handle(cmd); err != nil {
		// The stock is already held at this point, so this must not be skipped
		// as an optional-step failure — it has to be compensated.
		return &sideEffectError{fmt.Errorf("failed to record inventory reservation: %w", err)}
	}

	return nil
}

func (step *ReserveInventoryStep) Compensate() error {
	if len(step.reservationIDs) == 0 {
		return nil
	}

	request := map[string]interface{}{
		"cancelled_by": "system",
		"reason":       "Order saga compensation",
	}

	// Release every reservation, newest first. Keep going after a failure so one
	// unreachable reservation cannot strand the others, then report the first
	// error once the rest have been attempted.
	var firstErr error
	released := make([]string, 0, len(step.reservationIDs))
	for i := len(step.reservationIDs) - 1; i >= 0; i-- {
		id := step.reservationIDs[i]

		if _, err := step.callInventoryService(
			fmt.Sprintf("/api/v1/inventory/reservations/%s/cancel", id),
			request,
		); err != nil {
			step.saga.logger.Error("Failed to cancel inventory reservation",
				zap.String("order_id", step.saga.orderID),
				zap.String("reservation_id", id),
				zap.Error(err),
			)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to cancel inventory reservation %s: %w", id, err)
			}
			continue
		}
		released = append(released, id)
	}

	// Record what was actually released, so the event history matches reality
	// even on a partial release.
	for _, id := range released {
		cmd := commands.RecordInventoryReleaseCommand{
			OrderID:       step.saga.orderID,
			ReservationID: id,
			Reason:        "Saga compensation",
		}
		if err := step.saga.commandHandler.Handle(cmd); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to record inventory release %s: %w", id, err)
			}
		}
	}

	return firstErr
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
	if step.saga.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+step.saga.authToken)
	}
	if step.saga.order.TenantID != "" {
		req.Header.Set("X-Tenant-ID", step.saga.order.TenantID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
	// Build payment request (camelCase for .NET service)
	request := map[string]interface{}{
		"tenantId":   step.saga.order.TenantID,
		"customerId": step.saga.order.CustomerID,
		"orderId":    step.saga.orderID,
		"amount":     step.saga.order.TotalAmount,
		"currency":   step.saga.order.Currency,
		"method":     "bkash",
		"createdBy":  "system",
	}

	// Call Payment Service
	response, err := step.callPaymentService("/api/v1/payments", request)
	if err != nil {
		return fmt.Errorf("failed to process payment: %w", err)
	}

	paymentID, ok := response["id"].(string)
	if !ok || paymentID == "" {
		// A 2xx with no usable id means the charge may have gone through while
		// leaving us nothing to refund against. Never skip this: confirming the
		// order on an unknown payment outcome is worse than failing it. Log the
		// response field names (not values — they may carry PII) so the payment
		// can be reconciled by hand.
		fields := make([]string, 0, len(response))
		for k := range response {
			fields = append(fields, k)
		}
		sort.Strings(fields)

		step.saga.logger.Error("Payment service reported success without a payment id; the charge may have been captured and needs manual reconciliation",
			zap.String("order_id", step.saga.orderID),
			zap.String("tenant_id", step.saga.order.TenantID),
			zap.Float64("amount", step.saga.order.TotalAmount),
			zap.String("currency", step.saga.order.Currency),
			zap.Strings("response_fields", fields),
		)

		return &sideEffectError{errors.New("payment service returned no payment id")}
	}

	step.paymentID = paymentID

	// Record payment in order
	cmd := commands.RecordPaymentCommand{
		OrderID:       step.saga.orderID,
		PaymentID:     paymentID,
		PaymentMethod: "credit_card",
		TransactionID: fmt.Sprintf("txn_%s", paymentID),
		Amount:        step.saga.order.TotalAmount,
	}
	if err := step.saga.commandHandler.Handle(cmd); err != nil {
		// The payment is already captured at this point, so this must not be
		// skipped as an optional-step failure — it has to be refunded.
		return &sideEffectError{fmt.Errorf("failed to record payment: %w", err)}
	}

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
	cmd := commands.RecordPaymentFailureCommand{
		OrderID:   step.saga.orderID,
		PaymentID: step.paymentID,
		Reason:    "Refunded due to saga compensation",
	}
	if err := step.saga.commandHandler.Handle(cmd); err != nil {
		return fmt.Errorf("failed to record payment failure: %w", err)
	}

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
	if step.saga.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+step.saga.authToken)
	}
	if step.saga.order.TenantID != "" {
		req.Header.Set("X-Tenant-ID", step.saga.order.TenantID)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

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
