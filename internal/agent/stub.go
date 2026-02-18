package agent

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

// StubClient is a Client implementation that logs the envelope and returns
// a canned response. Useful for testing and development.
type StubClient struct {
	Logger *slog.Logger
}

// Forward logs the incoming envelope and returns a successful stub response.
func (s *StubClient) Forward(_ context.Context, envelope types.EventEnvelope) (Response, error) {
	s.Logger.Info("forwarding event",
		"event_id", envelope.Event.ID,
		"channel", envelope.Channel,
		"skills", len(envelope.Skills),
	)

	return Response{
		Status:  "ok",
		EventID: uuid.New().String(),
	}, nil
}
