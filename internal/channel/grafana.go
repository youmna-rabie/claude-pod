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

const maxBodySize = 1 << 20 // 1 MB

// GrafanaChannel validates Content-Type, auth token, parses JSON, and enforces a 1MB body limit.
type GrafanaChannel struct {
	name      string
	authToken string
}

// NewGrafanaChannel creates a GrafanaChannel with the given name and expected auth token.
func NewGrafanaChannel(name, authToken string) *GrafanaChannel {
	return &GrafanaChannel{name: name, authToken: authToken}
}

func (g *GrafanaChannel) Name() string {
	return g.name
}

func (g *GrafanaChannel) ValidateRequest(r *http.Request) error {
	if r.Method != http.MethodPost {
		return fmt.Errorf("method %s not allowed, expected POST", r.Method)
	}

	ct := r.Header.Get("Content-Type")
	if ct != "application/json" {
		return fmt.Errorf("unsupported Content-Type %q, expected application/json", ct)
	}

	if g.authToken != "" {
		tok := r.Header.Get("Authorization")
		if tok != "Bearer "+g.authToken {
			return fmt.Errorf("invalid or missing authorization token")
		}
	}

	return nil
}

func (g *GrafanaChannel) ParseRequest(r *http.Request) (*types.Event, error) {
	limited := io.LimitReader(r.Body, maxBodySize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}
	if len(body) > maxBodySize {
		return nil, fmt.Errorf("request body exceeds 1MB limit")
	}

	if !json.Valid(body) {
		return nil, fmt.Errorf("request body is not valid JSON")
	}

	return &types.Event{
		ID:        uuid.New(),
		ChannelID: g.name,
		RawBody:   json.RawMessage(body),
		Headers:   extractHeaders(r),
		Timestamp: time.Now(),
		Status:    types.EventStatusReceived,
	}, nil
}
