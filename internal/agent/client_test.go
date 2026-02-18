package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

func newTestEnvelope() types.EventEnvelope {
	return types.EventEnvelope{
		Version: "1.0",
		Event: types.Event{
			ID:        uuid.New(),
			ChannelID: "slack",
			RawBody:   json.RawMessage(`{"text":"hello"}`),
			Headers:   map[string]string{"Content-Type": "application/json"},
			Timestamp: time.Now(),
			Status:    types.EventStatusReceived,
		},
		Channel:   "slack",
		Skills:    []types.Skill{{Name: "echo", Description: "echoes input", Path: "/echo"}},
		Timestamp: time.Now(),
	}
}

func TestStubClient_ReturnsExpectedResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	stub := &StubClient{Logger: logger}
	resp, err := stub.Forward(context.Background(), newTestEnvelope())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.EventID == "" {
		t.Error("event_id should not be empty")
	}
	if _, err := uuid.Parse(resp.EventID); err != nil {
		t.Errorf("event_id %q is not a valid UUID: %v", resp.EventID, err)
	}
}

func TestStubClient_LogsEnvelope(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	stub := &StubClient{Logger: logger}
	envelope := newTestEnvelope()
	_, err := stub.Forward(context.Background(), envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}

	if msg, ok := logEntry["msg"].(string); !ok || msg != "forwarding event" {
		t.Errorf("log msg = %q, want %q", logEntry["msg"], "forwarding event")
	}
	if ch, ok := logEntry["channel"].(string); !ok || ch != "slack" {
		t.Errorf("log channel = %q, want %q", logEntry["channel"], "slack")
	}
}

func TestStubClient_ImplementsClientInterface(t *testing.T) {
	var _ Client = (*StubClient)(nil)
}
