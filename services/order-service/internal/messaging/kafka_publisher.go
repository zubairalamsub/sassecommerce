package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"go.uber.org/zap"
)

// EventPublisher publishes domain events to Kafka
type EventPublisher interface {
	Publish(ctx context.Context, event events.Event) error
	PublishBatch(ctx context.Context, events []events.Event) error
	Close() error
}

// KafkaEventPublisher publishes events to Kafka
type KafkaEventPublisher struct {
	writer *kafka.Writer
	logger *zap.Logger
	topic  string
}

// NewKafkaEventPublisher creates a new Kafka event publisher
func NewKafkaEventPublisher(brokers []string, topic string, logger *zap.Logger) *KafkaEventPublisher {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Compression:  kafka.Snappy,
		RequiredAcks: kafka.RequireOne,
		MaxAttempts:  3,
		Logger:       kafka.LoggerFunc(func(msg string, args ...interface{}) {}), // Suppress kafka logs
	}

	return &KafkaEventPublisher{
		writer: writer,
		logger: logger,
		topic:  topic,
	}
}

// Publish publishes a single event to Kafka
func (p *KafkaEventPublisher) Publish(ctx context.Context, event events.Event) error {
	return p.PublishBatch(ctx, []events.Event{event})
}

// PublishBatch publishes multiple events to Kafka
func (p *KafkaEventPublisher) PublishBatch(ctx context.Context, eventsToPublish []events.Event) error {
	if len(eventsToPublish) == 0 {
		return nil
	}

	messages := make([]kafka.Message, 0, len(eventsToPublish))

	for _, event := range eventsToPublish {
		// Create event envelope with metadata
		envelope := EventEnvelope{
			EventID:       event.GetID(),
			EventType:     string(event.GetEventType()),
			AggregateID:   event.GetAggregateID(),
			AggregateType: "Order",
			Version:       event.GetVersion(),
			Timestamp:     event.GetTimestamp(),
			Data:          event.GetData(),
		}

		// Serialize to JSON
		value, err := json.Marshal(envelope)
		if err != nil {
			p.logger.Error("Failed to marshal event",
				zap.String("event_id", event.GetID()),
				zap.Error(err),
			)
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		// Create Kafka message
		// Use aggregate ID as key for ordering guarantees
		message := kafka.Message{
			Key:   []byte(event.GetAggregateID()),
			Value: value,
			Headers: []kafka.Header{
				{Key: "event-type", Value: []byte(event.GetEventType())},
				{Key: "aggregate-id", Value: []byte(event.GetAggregateID())},
				{Key: "version", Value: []byte(fmt.Sprintf("%d", event.GetVersion()))},
			},
			Time: event.GetTimestamp(),
		}

		messages = append(messages, message)
	}

	// Publish to Kafka
	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
		p.logger.Error("Failed to publish events to Kafka",
			zap.Int("event_count", len(eventsToPublish)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish events: %w", err)
	}

	p.logger.Info("Published events to Kafka",
		zap.Int("event_count", len(eventsToPublish)),
		zap.String("topic", p.topic),
	)

	return nil
}

// Close closes the Kafka writer
func (p *KafkaEventPublisher) Close() error {
	if err := p.writer.Close(); err != nil {
		p.logger.Error("Failed to close Kafka writer", zap.Error(err))
		return err
	}
	return nil
}

// EventEnvelope wraps an event with metadata for Kafka
type EventEnvelope struct {
	EventID       string      `json:"event_id"`
	EventType     string      `json:"event_type"`
	AggregateID   string      `json:"aggregate_id"`
	AggregateType string      `json:"aggregate_type"`
	Version       int         `json:"version"`
	Timestamp     time.Time   `json:"timestamp"`
	Data          interface{} `json:"data"`
}
