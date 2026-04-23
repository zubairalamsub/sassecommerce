package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// ProducerConfig holds Kafka producer configuration
type ProducerConfig struct {
	Brokers      []string
	Topic        string
	RequiredAcks int
	Async        bool
	Compression  kafka.Compression
}

// DefaultProducerConfig returns default producer configuration
func DefaultProducerConfig(brokers []string, topic string) ProducerConfig {
	return ProducerConfig{
		Brokers:      brokers,
		Topic:        topic,
		RequiredAcks: 1,
		Async:        false,
		Compression:  kafka.Snappy,
	}
}

// Producer wraps kafka.Writer
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(config ProducerConfig) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		RequiredAcks: kafka.RequiredAcks(config.RequiredAcks),
		Async:        config.Async,
		Compression:  config.Compression,
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	}

	return &Producer{writer: writer}
}

// Publish sends a message to Kafka
func (p *Producer) Publish(ctx context.Context, key string, value interface{}) error {
	// Marshal value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Create message
	msg := kafka.Message{
		Key:   []byte(key),
		Value: valueBytes,
		Time:  time.Now(),
	}

	// Write message
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// PublishWithHeaders sends a message with headers to Kafka
func (p *Producer) PublishWithHeaders(ctx context.Context, key string, value interface{}, headers map[string]string) error {
	// Marshal value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Create headers
	kafkaHeaders := make([]kafka.Header, 0, len(headers))
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Create message
	msg := kafka.Message{
		Key:     []byte(key),
		Value:   valueBytes,
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}

	// Write message
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// PublishBatch sends multiple messages to Kafka
func (p *Producer) PublishBatch(ctx context.Context, messages []kafka.Message) error {
	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("failed to write messages: %w", err)
	}
	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// Stats returns producer statistics
func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}
