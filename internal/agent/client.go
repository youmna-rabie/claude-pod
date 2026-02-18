package agent

import (
	"context"
	"encoding/json"

	"github.com/youmna-rabie/claude-pod/internal/types"
)

// Response represents the result of forwarding an event to an agent.
type Response struct {
	Status  string          `json:"status"`
	EventID string          `json:"event_id"`
	Body    json.RawMessage `json:"body,omitempty"`
}

// Client defines the interface for forwarding events to an agent backend.
type Client interface {
	Forward(ctx context.Context, envelope types.EventEnvelope) (Response, error)
}
