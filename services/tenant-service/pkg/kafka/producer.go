package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Producer struct {
	writer *kafka.Writer
	logger *logrus.Logger
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string, logger *logrus.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}

	logger.WithField("brokers", brokers).Info("Kafka producer initialized")

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

// Publish publishes a message to a Kafka topic
func (p *Producer) Publish(ctx context.Context, topic, key string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"topic": topic,
			"key":   key,
		}).Error("Failed to publish message to Kafka")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"topic": topic,
		"key":   key,
	}).Debug("Message published to Kafka")

	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		p.logger.WithError(err).Error("Failed to close Kafka producer")
		return err
	}
	p.logger.Info("Kafka producer closed")
	return nil
}
