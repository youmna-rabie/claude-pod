package event

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

var (
	ErrNotFound        = errors.New("event not found")
	ErrInvalidCapacity = errors.New("capacity must be greater than zero")
)

// MemoryStore is an in-memory event store backed by a ring buffer.
// It provides O(1) lookups by ID via a map index and is safe for concurrent use.
type MemoryStore struct {
	mu    sync.RWMutex
	buf   []types.Event // ring buffer
	index map[uuid.UUID]int // event ID → position in buf
	cap   int               // maximum capacity
	count int               // current number of stored events
	head  int               // next write position
}

// NewMemoryStore creates a MemoryStore with the given capacity.
func NewMemoryStore(capacity int) (*MemoryStore, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &MemoryStore{
		buf:   make([]types.Event, capacity),
		index: make(map[uuid.UUID]int, capacity),
		cap:   capacity,
	}, nil
}

// Save adds an event to the store. If the store is at capacity, the oldest event is evicted.
func (s *MemoryStore) Save(event types.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If overwriting an existing slot, remove the old event from the index.
	if s.count == s.cap {
		old := s.buf[s.head]
		delete(s.index, old.ID)
	}

	s.buf[s.head] = event
	s.index[event.ID] = s.head

	s.head = (s.head + 1) % s.cap
	if s.count < s.cap {
		s.count++
	}

	return nil
}

// Get retrieves an event by ID in O(1) time.
func (s *MemoryStore) Get(id uuid.UUID) (types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pos, ok := s.index[id]
	if !ok {
		return types.Event{}, ErrNotFound
	}
	return s.buf[pos], nil
}

// List returns up to limit events ordered newest-first, skipping the first offset results.
func (s *MemoryStore) List(limit, offset int) ([]types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		return nil, nil
	}
	if offset < 0 {
		offset = 0
	}

	// Walk backwards from the most recently written slot.
	result := make([]types.Event, 0, min(limit, s.count))
	skipped := 0
	for i := 0; i < s.count; i++ {
		pos := (s.head - 1 - i + s.cap) % s.cap
		if skipped < offset {
			skipped++
			continue
		}
		result = append(result, s.buf[pos])
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

// UpdateStatus changes the status of an event identified by ID.
func (s *MemoryStore) UpdateStatus(id uuid.UUID, status types.EventStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos, ok := s.index[id]
	if !ok {
		return ErrNotFound
	}
	s.buf[pos].Status = status
	return nil
}

// Count returns the number of events currently stored.
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}
