package event

import (
	"github.com/google/uuid"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

// Store defines the interface for persisting and querying events.
type Store interface {
	// Save persists an event. Returns an error if the store is closed or the event is invalid.
	Save(event types.Event) error

	// Get retrieves an event by ID. Returns an error if not found.
	Get(id uuid.UUID) (types.Event, error)

	// List returns up to limit events, ordered newest-first.
	// offset skips the first N results for pagination.
	List(limit, offset int) ([]types.Event, error)

	// UpdateStatus changes the status of an event identified by ID.
	// Returns an error if the event is not found.
	UpdateStatus(id uuid.UUID, status types.EventStatus) error

	// Count returns the total number of events currently stored.
	Count() int
}
