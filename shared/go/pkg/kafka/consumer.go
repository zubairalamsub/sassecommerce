package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// ConsumerConfig holds Kafka consumer configuration
type ConsumerConfig struct {
	Brokers       []string
	Topic         string
	GroupID       string
	StartOffset   int64 // kafka.FirstOffset or kafka.LastOffset
	MinBytes      int
	MaxBytes      int
	CommitInterval time.Duration
}

// DefaultConsumerConfig returns default consumer configuration
func DefaultConsumerConfig(brokers []string, topic, groupID string) ConsumerConfig {
	return ConsumerConfig{
		Brokers:       brokers,
		Topic:         topic,
		GroupID:       groupID,
		StartOffset:   kafka.FirstOffset,
		MinBytes:      10e3, // 10KB
		MaxBytes:      10e6, // 10MB
		CommitInterval: time.Second,
	}
}

// Consumer wraps kafka.Reader
type Consumer struct {
	reader *kafka.Reader
	logger *logrus.Logger
}

// MessageHandler is a function that processes a Kafka message
type MessageHandler func(ctx context.Context, message kafka.Message) error

// NewConsumer creates a new Kafka consumer
func NewConsumer(config ConsumerConfig, logger *logrus.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		StartOffset:    config.StartOffset,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		CommitInterval: config.CommitInterval,
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}
}

// Consume starts consuming messages and calls the handler for each message
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Consumer stopped")
			return ctx.Err()
		default:
			// Read message
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				c.logger.WithError(err).Error("Failed to read message")
				continue
			}

			// Log message
			c.logger.WithFields(logrus.Fields{
				"topic":     msg.Topic,
				"partition": msg.Partition,
				"offset":    msg.Offset,
				"key":       string(msg.Key),
			}).Info("Received message")

			// Handle message
			if err := handler(ctx, msg); err != nil {
				c.logger.WithError(err).Error("Failed to handle message")
				// Continue processing other messages
				continue
			}

			// Commit message (if auto-commit is disabled)
			// The reader will auto-commit based on CommitInterval
		}
	}
}

// FetchMessage fetches a single message
func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.FetchMessage(ctx)
}

// CommitMessages commits messages
func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.reader.CommitMessages(ctx, msgs...)
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Stats returns consumer statistics
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// UnmarshalMessage unmarshals a Kafka message value into the provided interface
func UnmarshalMessage(msg kafka.Message, v interface{}) error {
	if err := json.Unmarshal(msg.Value, v); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return nil
}

// GetMessageHeader gets a header value from a Kafka message
func GetMessageHeader(msg kafka.Message, key string) string {
	for _, header := range msg.Headers {
		if header.Key == key {
			return string(header.Value)
		}
	}
	return ""
}
