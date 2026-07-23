package messaging

import (
	"context"
	"fmt"

	sharedkafka "github.com/ecommerce/shared/go/pkg/kafka"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// eventSigner HMAC-signs outgoing Kafka events when EVENT_SIGNING_KEY is set.
var eventSigner = sharedkafka.NewEventSignerFromEnv()

type Producer struct {
	writer *kafka.Writer
	logger *logrus.Logger
}

func NewProducer(brokers []string, logger *logrus.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
		// Async so WriteMessages returns immediately instead of blocking the
		// request handler on the broker round-trip. Publish failures are only
		// logged (events are best-effort), so surface them via Completion.
		Async: true,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				logger.WithError(err).WithField("message_count", len(messages)).
					Error("Async Kafka publish failed")
			}
		},
	}

	logger.WithField("brokers", brokers).Info("Kafka producer initialized")

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

func (p *Producer) Publish(ctx context.Context, topic, key string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	eventSigner.Sign(&msg)

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

func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		p.logger.WithError(err).Error("Failed to close Kafka producer")
		return err
	}
	p.logger.Info("Kafka producer closed")
	return nil
}
