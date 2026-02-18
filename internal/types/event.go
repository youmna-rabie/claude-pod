package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the lifecycle state of an event.
type EventStatus string

const (
	EventStatusReceived  EventStatus = "received"
	EventStatusForwarded EventStatus = "forwarded"
	EventStatusFailed    EventStatus = "failed"
	EventStatusCompleted EventStatus = "completed"
)

// Event represents an incoming request from a channel.
type Event struct {
	ID        uuid.UUID          `json:"id"`
	ChannelID string             `json:"channel_id"`
	RawBody   json.RawMessage    `json:"raw_body"`
	Headers   map[string]string  `json:"headers"`
	Timestamp time.Time          `json:"timestamp"`
	Status    EventStatus        `json:"status"`
}

// EventEnvelope wraps an event with routing metadata.
type EventEnvelope struct {
	Version   string    `json:"version"`
	Event     Event     `json:"event"`
	Channel   string    `json:"channel"`
	Skills    []Skill   `json:"skills"`
	Timestamp time.Time `json:"timestamp"`
}
