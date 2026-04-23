package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ecommerce/review-service/internal/models"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// EligibleOrder tracks orders that have been delivered and are eligible for review
type EligibleOrder struct {
	TenantID   string
	OrderID    string
	CustomerID string
	ProductIDs []string
	DeliveredAt time.Time
}

// EventConsumer consumes OrderDelivered events to enable review eligibility
type EventConsumer struct {
	reader *kafka.Reader
	logger *logrus.Logger
	stop   chan struct{}
	// In production, store eligible orders in a database; here we just log them
}

func NewEventConsumer(brokers []string, groupID string, logger *logrus.Logger) *EventConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          "order-events",
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})

	return &EventConsumer{
		reader: reader,
		logger: logger,
		stop:   make(chan struct{}),
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	go c.consumeLoop(ctx)
	c.logger.Info("Kafka event consumer started for order-events topic")
}

func (c *EventConsumer) Stop() {
	close(c.stop)
	if err := c.reader.Close(); err != nil {
		c.logger.WithError(err).Error("Failed to close Kafka reader")
	}
	c.logger.Info("Kafka event consumer stopped")
}

func (c *EventConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-c.stop:
			return
		case <-ctx.Done():
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.logger.WithError(err).Error("Failed to fetch message")
				time.Sleep(time.Second)
				continue
			}

			c.processMessage(msg)

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				c.logger.WithError(err).Error("Failed to commit message")
			}
		}
	}
}

func (c *EventConsumer) processMessage(msg kafka.Message) {
	var envelope models.EventEnvelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal event")
		return
	}

	if envelope.EventType == "OrderDelivered" {
		payload := envelope.GetPayload()
		if payload == nil {
			return
		}

		orderID, _ := payload["order_id"].(string)
		tenantID, _ := payload["tenant_id"].(string)
		customerID, _ := payload["customer_id"].(string)

		c.logger.WithFields(logrus.Fields{
			"event_type":  "OrderDelivered",
			"order_id":    orderID,
			"tenant_id":   tenantID,
			"customer_id": customerID,
		}).Info("Order delivered - review now eligible")
	}
}
