package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/projection"
	"go.uber.org/zap"
)

// EventConsumer consumes domain events from Kafka
type EventConsumer interface {
	Start(ctx context.Context) error
	Stop() error
}

// KafkaEventConsumer consumes events from Kafka and applies projections
type KafkaEventConsumer struct {
	reader     *kafka.Reader
	projection *projection.OrderProjection
	logger     *zap.Logger
	stopChan   chan struct{}
}

// NewKafkaEventConsumer creates a new Kafka event consumer
func NewKafkaEventConsumer(
	brokers []string,
	topic string,
	groupID string,
	projection *projection.OrderProjection,
	logger *zap.Logger,
) *KafkaEventConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
		Logger:         kafka.LoggerFunc(func(msg string, args ...interface{}) {}), // Suppress kafka logs
	})

	return &KafkaEventConsumer{
		reader:     reader,
		projection: projection,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

// Start starts consuming events
func (c *KafkaEventConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka event consumer",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)

	go c.consumeLoop(ctx)

	return nil
}

// consumeLoop continuously consumes messages from Kafka
func (c *KafkaEventConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer")
			return
		case <-c.stopChan:
			c.logger.Info("Stop signal received, stopping consumer")
			return
		default:
			// Read message with timeout
			message, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				c.logger.Error("Failed to fetch message", zap.Error(err))
				time.Sleep(time.Second) // Back off on error
				continue
			}

			// Process message
			if err := c.processMessage(ctx, message); err != nil {
				c.logger.Error("Failed to process message",
					zap.String("offset", fmt.Sprintf("%d", message.Offset)),
					zap.Error(err),
				)
				// Continue processing next message even on error
				// In production, consider dead letter queue
			}

			// Commit message
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				c.logger.Error("Failed to commit message", zap.Error(err))
			}
		}
	}
}

// processMessage processes a single Kafka message
func (c *KafkaEventConsumer) processMessage(ctx context.Context, message kafka.Message) error {
	// Unmarshal event envelope
	var envelope EventEnvelope
	if err := json.Unmarshal(message.Value, &envelope); err != nil {
		return fmt.Errorf("failed to unmarshal event envelope: %w", err)
	}

	c.logger.Debug("Processing event",
		zap.String("event_id", envelope.EventID),
		zap.String("event_type", envelope.EventType),
		zap.String("aggregate_id", envelope.AggregateID),
		zap.Int("version", envelope.Version),
	)

	// Deserialize event from envelope
	event, err := c.deserializeEvent(envelope)
	if err != nil {
		return fmt.Errorf("failed to deserialize event: %w", err)
	}

	// Apply projection
	if err := c.projection.Project(event); err != nil {
		return fmt.Errorf("failed to apply projection: %w", err)
	}

	c.logger.Info("Event processed successfully",
		zap.String("event_id", envelope.EventID),
		zap.String("event_type", envelope.EventType),
		zap.String("aggregate_id", envelope.AggregateID),
	)

	return nil
}

// deserializeEvent deserializes an event from envelope
func (c *KafkaEventConsumer) deserializeEvent(envelope EventEnvelope) (events.Event, error) {
	// Marshal data back to JSON for deserialization
	dataJSON, err := json.Marshal(envelope.Data)
	if err != nil {
		return nil, err
	}

	eventType := events.EventType(envelope.EventType)

	// Deserialize based on event type
	switch eventType {
	case events.OrderCreatedEvent:
		var event events.OrderCreated
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.OrderItemAddedEvent:
		var event events.OrderItemAdded
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.OrderItemRemovedEvent:
		var event events.OrderItemRemoved
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.OrderConfirmedEvent:
		var event events.OrderConfirmed
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.OrderCancelledEvent:
		var event events.OrderCancelled
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.OrderShippedEvent:
		var event events.OrderShipped
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.OrderDeliveredEvent:
		var event events.OrderDelivered
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.PaymentProcessedEvent:
		var event events.PaymentProcessed
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.PaymentFailedEvent:
		var event events.PaymentFailed
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.InventoryReservedEvent:
		var event events.InventoryReserved
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	case events.InventoryReleasedEvent:
		var event events.InventoryReleased
		if err := json.Unmarshal(dataJSON, &event); err != nil {
			return nil, err
		}
		return event, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}

// Stop stops the consumer
func (c *KafkaEventConsumer) Stop() error {
	close(c.stopChan)

	if err := c.reader.Close(); err != nil {
		c.logger.Error("Failed to close Kafka reader", zap.Error(err))
		return err
	}

	c.logger.Info("Kafka event consumer stopped")
	return nil
}
