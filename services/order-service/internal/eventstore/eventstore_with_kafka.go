package eventstore

import (
	"context"
	"fmt"

	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
	"github.com/yourusername/ecommerce/order-service/internal/messaging"
	"go.uber.org/zap"
)

// EventStoreWithKafka wraps an event store and publishes events to Kafka
type EventStoreWithKafka struct {
	eventStore EventStore
	publisher  messaging.EventPublisher
	logger     *zap.Logger
}

// NewEventStoreWithKafka creates a new event store with Kafka publishing
func NewEventStoreWithKafka(
	eventStore EventStore,
	publisher messaging.EventPublisher,
	logger *zap.Logger,
) *EventStoreWithKafka {
	return &EventStoreWithKafka{
		eventStore: eventStore,
		publisher:  publisher,
		logger:     logger,
	}
}

// Save saves events to the event store and publishes them to Kafka
func (es *EventStoreWithKafka) Save(aggregateID string, eventsToSave []events.Event, expectedVersion int) error {
	// First save to event store (source of truth)
	if err := es.eventStore.Save(aggregateID, eventsToSave, expectedVersion); err != nil {
		return err
	}

	// Then publish to Kafka for event-driven architecture
	ctx := context.Background()
	if err := es.publisher.PublishBatch(ctx, eventsToSave); err != nil {
		// Log error but don't fail - events are persisted in event store
		// Consumers can replay from event store if needed
		es.logger.Error("Failed to publish events to Kafka",
			zap.String("aggregate_id", aggregateID),
			zap.Int("event_count", len(eventsToSave)),
			zap.Error(err),
		)
		// Return error to signal publishing failure
		return fmt.Errorf("failed to publish events to Kafka: %w", err)
	}

	es.logger.Debug("Events saved and published",
		zap.String("aggregate_id", aggregateID),
		zap.Int("event_count", len(eventsToSave)),
	)

	return nil
}

// GetEvents retrieves events from the event store
func (es *EventStoreWithKafka) GetEvents(aggregateID string) ([]events.Event, error) {
	return es.eventStore.GetEvents(aggregateID)
}

// GetEventsByType retrieves events by type from the event store
func (es *EventStoreWithKafka) GetEventsByType(eventType events.EventType, limit int) ([]events.Event, error) {
	return es.eventStore.GetEventsByType(eventType, limit)
}

// GetAllEvents retrieves all events from the event store
func (es *EventStoreWithKafka) GetAllEvents(offset, limit int) ([]events.Event, error) {
	return es.eventStore.GetAllEvents(offset, limit)
}

// Close closes the underlying event store and Kafka publisher
func (es *EventStoreWithKafka) Close() error {
	var errs []error

	// Close publisher
	if err := es.publisher.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close publisher: %w", err))
	}

	// Close event store if it implements Close
	if closer, ok := es.eventStore.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close event store: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing event store with kafka: %v", errs)
	}

	return nil
}
