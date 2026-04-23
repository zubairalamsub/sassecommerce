package eventstore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/yourusername/ecommerce/order-service/internal/domain/events"
)

// EventStore is responsible for persisting and retrieving events
type EventStore interface {
	Save(aggregateID string, events []events.Event, expectedVersion int) error
	GetEvents(aggregateID string) ([]events.Event, error)
	GetEventsByType(eventType events.EventType, limit int) ([]events.Event, error)
	GetAllEvents(offset, limit int) ([]events.Event, error)
}

// PostgresEventStore implements EventStore using PostgreSQL
type PostgresEventStore struct {
	db *sql.DB
}

// StoredEvent represents an event as stored in the database
type StoredEvent struct {
	ID          string
	AggregateID string
	EventType   events.EventType
	EventData   json.RawMessage
	Version     int
	Timestamp   string
}

// NewPostgresEventStore creates a new PostgreSQL event store
func NewPostgresEventStore(db *sql.DB) (*PostgresEventStore, error) {
	store := &PostgresEventStore{db: db}

	// Create events table if not exists
	if err := store.createTable(); err != nil {
		return nil, err
	}

	return store, nil
}

// createTable creates the events table
func (es *PostgresEventStore) createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		id VARCHAR(36) PRIMARY KEY,
		aggregate_id VARCHAR(36) NOT NULL,
		event_type VARCHAR(50) NOT NULL,
		event_data JSONB NOT NULL,
		version INTEGER NOT NULL,
		timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

		CONSTRAINT unique_aggregate_version UNIQUE (aggregate_id, version)
	);

	CREATE INDEX IF NOT EXISTS idx_events_aggregate_id ON events(aggregate_id);
	CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	`

	_, err := es.db.Exec(query)
	return err
}

// Save persists events to the store with optimistic concurrency control
func (es *PostgresEventStore) Save(aggregateID string, eventsToSave []events.Event, expectedVersion int) error {
	if len(eventsToSave) == 0 {
		return nil
	}

	// Start transaction
	tx, err := es.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check current version for optimistic concurrency control
	var currentVersion int
	err = tx.QueryRow(`
		SELECT COALESCE(MAX(version), -1)
		FROM events
		WHERE aggregate_id = $1
	`, aggregateID).Scan(&currentVersion)

	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion != expectedVersion {
		return errors.New("concurrency conflict: aggregate version mismatch")
	}

	// Insert events
	stmt, err := tx.Prepare(`
		INSERT INTO events (id, aggregate_id, event_type, event_data, version, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range eventsToSave {
		eventData, err := events.Serialize(event)
		if err != nil {
			return fmt.Errorf("failed to serialize event: %w", err)
		}

		_, err = stmt.Exec(
			event.GetID(),
			event.GetAggregateID(),
			event.GetEventType(),
			eventData,
			event.GetVersion(),
			event.GetTimestamp(),
		)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetEvents retrieves all events for a specific aggregate
func (es *PostgresEventStore) GetEvents(aggregateID string) ([]events.Event, error) {
	rows, err := es.db.Query(`
		SELECT id, aggregate_id, event_type, event_data, version, timestamp
		FROM events
		WHERE aggregate_id = $1
		ORDER BY version ASC
	`, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	return es.scanEvents(rows)
}

// GetEventsByType retrieves events of a specific type
func (es *PostgresEventStore) GetEventsByType(eventType events.EventType, limit int) ([]events.Event, error) {
	rows, err := es.db.Query(`
		SELECT id, aggregate_id, event_type, event_data, version, timestamp
		FROM events
		WHERE event_type = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, eventType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by type: %w", err)
	}
	defer rows.Close()

	return es.scanEvents(rows)
}

// GetAllEvents retrieves all events with pagination
func (es *PostgresEventStore) GetAllEvents(offset, limit int) ([]events.Event, error) {
	rows, err := es.db.Query(`
		SELECT id, aggregate_id, event_type, event_data, version, timestamp
		FROM events
		ORDER BY timestamp ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query all events: %w", err)
	}
	defer rows.Close()

	return es.scanEvents(rows)
}

// scanEvents scans database rows into events
func (es *PostgresEventStore) scanEvents(rows *sql.Rows) ([]events.Event, error) {
	var eventList []events.Event

	for rows.Next() {
		var stored StoredEvent
		err := rows.Scan(
			&stored.ID,
			&stored.AggregateID,
			&stored.EventType,
			&stored.EventData,
			&stored.Version,
			&stored.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Deserialize event
		event, err := events.Deserialize(stored.EventType, stored.EventData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}

		eventList = append(eventList, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return eventList, nil
}
