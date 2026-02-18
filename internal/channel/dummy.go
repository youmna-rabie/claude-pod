package channel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

// DummyChannel accepts any POST request and wraps the body in an Event.
type DummyChannel struct {
	name string
}

// NewDummyChannel creates a DummyChannel with the given name.
func NewDummyChannel(name string) *DummyChannel {
	return &DummyChannel{name: name}
}

func (d *DummyChannel) Name() string {
	return d.name
}

func (d *DummyChannel) ValidateRequest(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method %s not allowed, expected POST", r.Method)
	}
	return nil
}

func (d *DummyChannel) ParseRequest(r *http.Request) (*types.Event, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	return &types.Event{
		ID:        uuid.New(),
		ChannelID: d.name,
		RawBody:   json.RawMessage(body),
		Headers:   extractHeaders(r),
		Timestamp: time.Now(),
		Status:    types.EventStatusReceived,
	}, nil
}

func extractHeaders(r *http.Request) map[string]string {
	h := make(map[string]string, len(r.Header))
	for k := range r.Header {
		h[k] = r.Header.Get(k)
	}
	return h
}
