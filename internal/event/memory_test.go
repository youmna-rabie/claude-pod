package event

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

func makeEvent(channelID string) types.Event {
	return types.Event{
		ID:        uuid.New(),
		ChannelID: channelID,
		RawBody:   json.RawMessage(`{"test":true}`),
		Headers:   map[string]string{"X-Test": "1"},
		Timestamp: time.Now(),
		Status:    types.EventStatusReceived,
	}
}

func TestNewMemoryStore_InvalidCapacity(t *testing.T) {
	for _, cap := range []int{0, -1, -100} {
		_, err := NewMemoryStore(cap)
		if err != ErrInvalidCapacity {
			t.Errorf("NewMemoryStore(%d) error = %v, want ErrInvalidCapacity", cap, err)
		}
	}
}

func TestSaveAndGet(t *testing.T) {
	store, err := NewMemoryStore(10)
	if err != nil {
		t.Fatal(err)
	}

	ev := makeEvent("slack")
	if err := store.Save(ev); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ev.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != ev.ID {
		t.Errorf("ID = %v, want %v", got.ID, ev.ID)
	}
	if got.ChannelID != ev.ChannelID {
		t.Errorf("ChannelID = %q, want %q", got.ChannelID, ev.ChannelID)
	}
	if got.Status != ev.Status {
		t.Errorf("Status = %q, want %q", got.Status, ev.Status)
	}
}

func TestGetNotFound(t *testing.T) {
	store, _ := NewMemoryStore(10)
	_, err := store.Get(uuid.New())
	if err != ErrNotFound {
		t.Errorf("Get unknown ID: error = %v, want ErrNotFound", err)
	}
}

func TestEvictionAtCapacity(t *testing.T) {
	store, _ := NewMemoryStore(3)

	events := make([]types.Event, 5)
	for i := range events {
		events[i] = makeEvent("ch")
		if err := store.Save(events[i]); err != nil {
			t.Fatalf("Save[%d]: %v", i, err)
		}
	}

	// Count should be capped at capacity.
	if c := store.Count(); c != 3 {
		t.Errorf("Count = %d, want 3", c)
	}

	// Oldest two (events[0], events[1]) should be evicted.
	for _, ev := range events[:2] {
		_, err := store.Get(ev.ID)
		if err != ErrNotFound {
			t.Errorf("Get evicted event %v: error = %v, want ErrNotFound", ev.ID, err)
		}
	}

	// Newest three should still be present.
	for _, ev := range events[2:] {
		got, err := store.Get(ev.ID)
		if err != nil {
			t.Errorf("Get retained event %v: %v", ev.ID, err)
		}
		if got.ID != ev.ID {
			t.Errorf("retained event ID = %v, want %v", got.ID, ev.ID)
		}
	}
}

func TestListNewestFirst(t *testing.T) {
	store, _ := NewMemoryStore(10)

	events := make([]types.Event, 5)
	for i := range events {
		events[i] = makeEvent("ch")
		store.Save(events[i])
	}

	listed, err := store.List(10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 5 {
		t.Fatalf("List len = %d, want 5", len(listed))
	}

	// Should be newest-first: events[4], events[3], events[2], events[1], events[0].
	for i, ev := range listed {
		want := events[len(events)-1-i]
		if ev.ID != want.ID {
			t.Errorf("List[%d].ID = %v, want %v", i, ev.ID, want.ID)
		}
	}
}

func TestListPagination(t *testing.T) {
	store, _ := NewMemoryStore(10)

	events := make([]types.Event, 5)
	for i := range events {
		events[i] = makeEvent("ch")
		store.Save(events[i])
	}

	// Page 1: limit=2, offset=0 → newest two.
	page1, _ := store.List(2, 0)
	if len(page1) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(page1))
	}
	if page1[0].ID != events[4].ID || page1[1].ID != events[3].ID {
		t.Error("page1 has wrong events")
	}

	// Page 2: limit=2, offset=2 → next two.
	page2, _ := store.List(2, 2)
	if len(page2) != 2 {
		t.Fatalf("page2 len = %d, want 2", len(page2))
	}
	if page2[0].ID != events[2].ID || page2[1].ID != events[1].ID {
		t.Error("page2 has wrong events")
	}

	// Page 3: limit=2, offset=4 → last one.
	page3, _ := store.List(2, 4)
	if len(page3) != 1 {
		t.Fatalf("page3 len = %d, want 1", len(page3))
	}

	// Beyond range: limit=2, offset=10 → empty.
	page4, _ := store.List(2, 10)
	if len(page4) != 0 {
		t.Fatalf("page4 len = %d, want 0", len(page4))
	}
}

func TestListWithEviction(t *testing.T) {
	store, _ := NewMemoryStore(3)

	events := make([]types.Event, 5)
	for i := range events {
		events[i] = makeEvent("ch")
		store.Save(events[i])
	}

	listed, _ := store.List(10, 0)
	if len(listed) != 3 {
		t.Fatalf("List len = %d, want 3", len(listed))
	}
	// Should be newest-first: events[4], events[3], events[2].
	if listed[0].ID != events[4].ID || listed[1].ID != events[3].ID || listed[2].ID != events[2].ID {
		t.Error("List after eviction has wrong order")
	}
}

func TestUpdateStatus(t *testing.T) {
	store, _ := NewMemoryStore(10)

	ev := makeEvent("slack")
	store.Save(ev)

	err := store.UpdateStatus(ev.ID, types.EventStatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, _ := store.Get(ev.ID)
	if got.Status != types.EventStatusCompleted {
		t.Errorf("Status = %q, want %q", got.Status, types.EventStatusCompleted)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	store, _ := NewMemoryStore(10)
	err := store.UpdateStatus(uuid.New(), types.EventStatusFailed)
	if err != ErrNotFound {
		t.Errorf("UpdateStatus unknown ID: error = %v, want ErrNotFound", err)
	}
}

func TestCount(t *testing.T) {
	store, _ := NewMemoryStore(5)

	if c := store.Count(); c != 0 {
		t.Errorf("empty store Count = %d, want 0", c)
	}

	for i := 0; i < 3; i++ {
		store.Save(makeEvent("ch"))
	}
	if c := store.Count(); c != 3 {
		t.Errorf("Count after 3 saves = %d, want 3", c)
	}

	// Fill and overflow.
	for i := 0; i < 5; i++ {
		store.Save(makeEvent("ch"))
	}
	if c := store.Count(); c != 5 {
		t.Errorf("Count after overflow = %d, want 5", c)
	}
}

func TestConcurrentAccess(t *testing.T) {
	store, _ := NewMemoryStore(100)

	var wg sync.WaitGroup
	const goroutines = 10
	const opsPerGoroutine = 50

	// Concurrent writers.
	written := make([]types.Event, goroutines*opsPerGoroutine)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				ev := makeEvent("concurrent")
				written[offset+i] = ev
				store.Save(ev)
			}
		}(g * opsPerGoroutine)
	}
	wg.Wait()

	// Concurrent readers.
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				store.List(10, 0)
				store.Count()
			}
		}()
	}
	wg.Wait()

	// Concurrent mixed reads and writes.
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				ev := makeEvent("mixed")
				store.Save(ev)
				store.Get(ev.ID)
				store.UpdateStatus(ev.ID, types.EventStatusForwarded)
				store.List(5, 0)
				store.Count()
			}
		}()
	}
	wg.Wait()

	// Verify store is in a consistent state.
	count := store.Count()
	if count < 1 || count > 100 {
		t.Errorf("Count after concurrent ops = %d, expected 1-100", count)
	}
}

func TestStoreInterface(t *testing.T) {
	// Compile-time check that MemoryStore implements Store.
	var _ Store = (*MemoryStore)(nil)
}
