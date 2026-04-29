package projection

import (
	"database/sql"
	"fmt"

	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/domain/queries"
	"go.uber.org/zap"
)

// OrderProjection maintains the read model for orders
type OrderProjection struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewOrderProjection creates a new order projection
func NewOrderProjection(db *sql.DB, logger *zap.Logger) (*OrderProjection, error) {
	projection := &OrderProjection{
		db:     db,
		logger: logger,
	}

	if err := projection.createTables(); err != nil {
		return nil, err
	}

	return projection, nil
}

// createTables creates read model tables
func (p *OrderProjection) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS order_read_model (
		id VARCHAR(36) PRIMARY KEY,
		tenant_id VARCHAR(36) NOT NULL,
		customer_id VARCHAR(36) NOT NULL,
		status VARCHAR(20) NOT NULL,
		total_amount DECIMAL(15, 2) NOT NULL DEFAULT 0,
		currency VARCHAR(3) NOT NULL DEFAULT 'BDT',
		shipping_street VARCHAR(255),
		shipping_city VARCHAR(100),
		shipping_state VARCHAR(100),
		shipping_postal_code VARCHAR(20),
		shipping_country VARCHAR(100),
		billing_street VARCHAR(255),
		billing_city VARCHAR(100),
		billing_state VARCHAR(100),
		billing_postal_code VARCHAR(20),
		billing_country VARCHAR(100),
		payment_id VARCHAR(36),
		reservation_id VARCHAR(36),
		tracking_number VARCHAR(100),
		carrier VARCHAR(100),
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		version INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_order_tenant ON order_read_model (tenant_id);
	CREATE INDEX IF NOT EXISTS idx_order_customer ON order_read_model (customer_id);
	CREATE INDEX IF NOT EXISTS idx_order_status ON order_read_model (status);
	CREATE INDEX IF NOT EXISTS idx_order_created_at ON order_read_model (created_at);

	CREATE TABLE IF NOT EXISTS order_item_read_model (
		id VARCHAR(100) PRIMARY KEY,
		order_id VARCHAR(36) NOT NULL,
		product_id VARCHAR(36) NOT NULL,
		variant_id VARCHAR(36),
		sku VARCHAR(100) NOT NULL,
		name VARCHAR(255) NOT NULL,
		quantity INTEGER NOT NULL,
		unit_price DECIMAL(15, 2) NOT NULL,
		total_price DECIMAL(15, 2) NOT NULL,
		FOREIGN KEY (order_id) REFERENCES order_read_model(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_order_item_order ON order_item_read_model (order_id);
	CREATE INDEX IF NOT EXISTS idx_order_item_product ON order_item_read_model (product_id);
	`

	_, err := p.db.Exec(query)
	return err
}

// Project applies an event to the read model
func (p *OrderProjection) Project(event events.Event) error {
	switch e := event.(type) {
	case events.OrderCreated:
		return p.projectOrderCreated(e)
	case events.OrderItemAdded:
		return p.projectOrderItemAdded(e)
	case events.OrderItemRemoved:
		return p.projectOrderItemRemoved(e)
	case events.OrderConfirmed:
		return p.projectOrderConfirmed(e)
	case events.OrderCancelled:
		return p.projectOrderCancelled(e)
	case events.OrderShipped:
		return p.projectOrderShipped(e)
	case events.OrderDelivered:
		return p.projectOrderDelivered(e)
	case events.PaymentProcessed:
		return p.projectPaymentProcessed(e)
	case events.InventoryReserved:
		return p.projectInventoryReserved(e)
	case events.InventoryReleased:
		return p.projectInventoryReleased(e)
	default:
		p.logger.Debug("Unhandled event type for projection",
			zap.String("event_type", string(event.GetEventType())),
		)
		return nil
	}
}

func (p *OrderProjection) projectOrderCreated(e events.OrderCreated) error {
	_, err := p.db.Exec(`
		INSERT INTO order_read_model (
			id, tenant_id, customer_id, status, total_amount, currency,
			shipping_street, shipping_city, shipping_state, shipping_postal_code, shipping_country,
			billing_street, billing_city, billing_state, billing_postal_code, billing_country,
			created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (id) DO NOTHING
	`,
		e.AggregateID, e.TenantID, e.CustomerID, "pending", e.TotalAmount, e.Currency,
		e.ShippingAddr.Street, e.ShippingAddr.City, e.ShippingAddr.State, e.ShippingAddr.PostalCode, e.ShippingAddr.Country,
		e.BillingAddr.Street, e.BillingAddr.City, e.BillingAddr.State, e.BillingAddr.PostalCode, e.BillingAddr.Country,
		e.Timestamp, e.Timestamp, e.Version,
	)
	return err
}

func (p *OrderProjection) projectOrderItemAdded(e events.OrderItemAdded) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Upsert order item (idempotent for duplicate event processing)
	_, err = tx.Exec(`
		INSERT INTO order_item_read_model (id, order_id, product_id, variant_id, sku, name, quantity, unit_price, total_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			quantity = EXCLUDED.quantity,
			unit_price = EXCLUDED.unit_price,
			total_price = EXCLUDED.total_price
	`, e.ItemID, e.AggregateID, e.ProductID, e.VariantID, e.SKU, e.Name, e.Quantity, e.UnitPrice, e.TotalPrice)
	if err != nil {
		return err
	}

	// Update order total
	_, err = tx.Exec(`
		UPDATE order_read_model
		SET total_amount = (
			SELECT COALESCE(SUM(total_price), 0)
			FROM order_item_read_model
			WHERE order_id = $1
		),
		updated_at = $2,
		version = $3
		WHERE id = $1
	`, e.AggregateID, e.Timestamp, e.Version)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *OrderProjection) projectOrderItemRemoved(e events.OrderItemRemoved) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete order item
	_, err = tx.Exec(`DELETE FROM order_item_read_model WHERE id = $1 AND order_id = $2`, e.ItemID, e.AggregateID)
	if err != nil {
		return err
	}

	// Update order total
	_, err = tx.Exec(`
		UPDATE order_read_model
		SET total_amount = (
			SELECT COALESCE(SUM(total_price), 0)
			FROM order_item_read_model
			WHERE order_id = $1
		),
		updated_at = $2,
		version = $3
		WHERE id = $1
	`, e.AggregateID, e.Timestamp, e.Version)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *OrderProjection) projectOrderConfirmed(e events.OrderConfirmed) error {
	return p.updateOrderStatus(e.AggregateID, "confirmed", e.Timestamp, e.Version)
}

func (p *OrderProjection) projectOrderCancelled(e events.OrderCancelled) error {
	return p.updateOrderStatus(e.AggregateID, "cancelled", e.Timestamp, e.Version)
}

func (p *OrderProjection) projectOrderShipped(e events.OrderShipped) error {
	_, err := p.db.Exec(`
		UPDATE order_read_model
		SET status = $1, tracking_number = $2, carrier = $3, updated_at = $4, version = $5
		WHERE id = $6
	`, "shipped", e.TrackingNumber, e.Carrier, e.Timestamp, e.Version, e.AggregateID)
	return err
}

func (p *OrderProjection) projectOrderDelivered(e events.OrderDelivered) error {
	return p.updateOrderStatus(e.AggregateID, "delivered", e.Timestamp, e.Version)
}

func (p *OrderProjection) projectPaymentProcessed(e events.PaymentProcessed) error {
	_, err := p.db.Exec(`
		UPDATE order_read_model
		SET payment_id = $1, updated_at = $2, version = $3
		WHERE id = $4
	`, e.PaymentID, e.Timestamp, e.Version, e.AggregateID)
	return err
}

func (p *OrderProjection) projectInventoryReserved(e events.InventoryReserved) error {
	_, err := p.db.Exec(`
		UPDATE order_read_model
		SET reservation_id = $1, updated_at = $2, version = $3
		WHERE id = $4
	`, e.ReservationID, e.Timestamp, e.Version, e.AggregateID)
	return err
}

func (p *OrderProjection) projectInventoryReleased(e events.InventoryReleased) error {
	_, err := p.db.Exec(`
		UPDATE order_read_model
		SET reservation_id = NULL, updated_at = $1, version = $2
		WHERE id = $3
	`, e.Timestamp, e.Version, e.AggregateID)
	return err
}

func (p *OrderProjection) updateOrderStatus(orderID, status string, timestamp interface{}, version int) error {
	_, err := p.db.Exec(`
		UPDATE order_read_model
		SET status = $1, updated_at = $2, version = $3
		WHERE id = $4
	`, status, timestamp, version, orderID)
	return err
}

// GetOrder retrieves an order from the read model
func (p *OrderProjection) GetOrder(orderID string) (*queries.OrderReadModel, error) {
	var order queries.OrderReadModel
	var shippingAddr, billingAddr queries.Address

	err := p.db.QueryRow(`
		SELECT id, tenant_id, customer_id, status, total_amount, currency,
			shipping_street, shipping_city, shipping_state, shipping_postal_code, shipping_country,
			billing_street, billing_city, billing_state, billing_postal_code, billing_country,
			COALESCE(payment_id, ''), COALESCE(reservation_id, ''),
			COALESCE(tracking_number, ''), COALESCE(carrier, ''),
			created_at, updated_at, version
		FROM order_read_model
		WHERE id = $1
	`, orderID).Scan(
		&order.ID, &order.TenantID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.Currency,
		&shippingAddr.Street, &shippingAddr.City, &shippingAddr.State, &shippingAddr.PostalCode, &shippingAddr.Country,
		&billingAddr.Street, &billingAddr.City, &billingAddr.State, &billingAddr.PostalCode, &billingAddr.Country,
		&order.PaymentID, &order.ReservationID,
		&order.TrackingNumber, &order.Carrier,
		&order.CreatedAt, &order.UpdatedAt, &order.Version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, err
	}

	order.ShippingAddress = shippingAddr
	order.BillingAddress = billingAddr

	return &order, nil
}

// GetOrderItems retrieves order items from the read model
func (p *OrderProjection) GetOrderItems(orderID string) ([]*queries.OrderItemReadModel, error) {
	rows, err := p.db.Query(`
		SELECT id, order_id, product_id, COALESCE(variant_id, ''), sku, name, quantity, unit_price, total_price
		FROM order_item_read_model
		WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*queries.OrderItemReadModel, 0)
	for rows.Next() {
		var item queries.OrderItemReadModel
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.VariantID,
			&item.SKU, &item.Name, &item.Quantity, &item.UnitPrice, &item.TotalPrice,
		); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, rows.Err()
}

// GetOrdersByCustomer retrieves orders for a customer
func (p *OrderProjection) GetOrdersByCustomer(customerID string, limit, offset int) ([]*queries.OrderSummary, error) {
	rows, err := p.db.Query(`
		SELECT o.id, o.customer_id, o.status, o.total_amount, o.currency, o.created_at, o.updated_at,
			COALESCE((SELECT COUNT(*) FROM order_item_read_model WHERE order_id = o.id), 0) as item_count
		FROM order_read_model o
		WHERE o.customer_id = $1
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`, customerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return p.scanOrderSummaries(rows)
}

// GetOrdersByTenant retrieves orders for a tenant
func (p *OrderProjection) GetOrdersByTenant(tenantID string, limit, offset int) ([]*queries.OrderSummary, error) {
	rows, err := p.db.Query(`
		SELECT o.id, o.customer_id, o.status, o.total_amount, o.currency, o.created_at, o.updated_at,
			COALESCE((SELECT COUNT(*) FROM order_item_read_model WHERE order_id = o.id), 0) as item_count
		FROM order_read_model o
		WHERE o.tenant_id = $1
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return p.scanOrderSummaries(rows)
}

func (p *OrderProjection) scanOrderSummaries(rows *sql.Rows) ([]*queries.OrderSummary, error) {
	summaries := make([]*queries.OrderSummary, 0)
	for rows.Next() {
		var summary queries.OrderSummary
		if err := rows.Scan(
			&summary.ID, &summary.CustomerID, &summary.Status, &summary.TotalAmount,
			&summary.Currency, &summary.CreatedAt, &summary.UpdatedAt, &summary.ItemCount,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, &summary)
	}

	return summaries, rows.Err()
}
