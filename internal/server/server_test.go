package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/youmna-rabie/claude-pod/internal/agent"
	"github.com/youmna-rabie/claude-pod/internal/config"
	"github.com/youmna-rabie/claude-pod/internal/event"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

// testSetup creates a Server with a dummy channel, stub agent, and in-memory store.
func testSetup(t *testing.T) *Server {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 0},
	}

	store, err := event.NewMemoryStore(100)
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.Default()

	channels := map[string]types.Channel{
		"dummy": &dummyTestChannel{name: "dummy"},
	}

	agentClient := &agent.StubClient{Logger: logger}

	skills := []types.Skill{
		{Name: "echo", Description: "echoes input", Path: "/skills/echo"},
	}

	return NewServer(cfg, store, channels, agentClient, skills, logger)
}

// dummyTestChannel is a minimal Channel for testing that accepts any POST with a body.
type dummyTestChannel struct {
	name string
}

func (d *dummyTestChannel) Name() string { return d.name }

func (d *dummyTestChannel) ValidateRequest(r *http.Request) error {
	if r.Method != http.MethodPost {
		return &validationError{msg: "method not allowed"}
	}
	return nil
}

func (d *dummyTestChannel) ParseRequest(r *http.Request) (*types.Event, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		return nil, err
	}

	return &types.Event{
		ID:        [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ChannelID: d.name,
		RawBody:   json.RawMessage(buf.Bytes()),
		Headers:   map[string]string{},
		Status:    types.EventStatusReceived,
	}, nil
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

// --- Health endpoint ---

func TestHealthEndpoint(t *testing.T) {
	srv := testSetup(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

// --- Webhook pipeline ---

func TestWebhookPipeline(t *testing.T) {
	srv := testSetup(t)

	payload := `{"alert":"cpu_high"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/dummy", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify response is agent response JSON
	var resp agent.Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected agent status ok, got %q", resp.Status)
	}

	// Verify event was stored
	events, err := srv.store.List(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(events))
	}
	if events[0].Status != types.EventStatusForwarded {
		t.Fatalf("expected status forwarded, got %s", events[0].Status)
	}
}

func TestWebhookUnknownChannel(t *testing.T) {
	srv := testSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/unknown", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestWebhookValidationFailure(t *testing.T) {
	srv := testSetup(t)

	// GET instead of POST triggers validation error
	req := httptest.NewRequest(http.MethodGet, "/webhooks/dummy", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	// chi returns 405 for wrong method on a route that only has Post
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// --- Admin endpoints ---

func TestAdminEventsEmpty(t *testing.T) {
	srv := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"].(float64) != 0 {
		t.Fatalf("expected 0 events, got %v", body["count"])
	}
}

func TestAdminEventsAfterWebhook(t *testing.T) {
	srv := testSetup(t)

	// Post a webhook first
	payload := `{"test":"data"}`
	webhookReq := httptest.NewRequest(http.MethodPost, "/webhooks/dummy", bytes.NewBufferString(payload))
	webhookRec := httptest.NewRecorder()
	srv.ServeHTTP(webhookRec, webhookReq)

	if webhookRec.Code != http.StatusOK {
		t.Fatalf("webhook failed: %d %s", webhookRec.Code, webhookRec.Body.String())
	}

	// Now check admin events
	req := httptest.NewRequest(http.MethodGet, "/admin/events", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"].(float64) != 1 {
		t.Fatalf("expected 1 event, got %v", body["count"])
	}
}

func TestAdminChannels(t *testing.T) {
	srv := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/channels", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"].(float64) != 1 {
		t.Fatalf("expected 1 channel, got %v", body["count"])
	}
	channels := body["channels"].([]any)
	if channels[0].(string) != "dummy" {
		t.Fatalf("expected channel dummy, got %v", channels[0])
	}
}

func TestAdminSkills(t *testing.T) {
	srv := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["count"].(float64) != 1 {
		t.Fatalf("expected 1 skill, got %v", body["count"])
	}
}

// --- Middleware ---

func TestRequestIDMiddleware(t *testing.T) {
	srv := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	rid := rec.Header().Get("X-Request-ID")
	if rid == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
}

func TestRequestIDPassthrough(t *testing.T) {
	srv := testSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "test-id-123")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	rid := rec.Header().Get("X-Request-ID")
	if rid != "test-id-123" {
		t.Fatalf("expected X-Request-ID test-id-123, got %q", rid)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.Default()
	handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Add request ID context so recovery middleware can log it
	ctx := context.WithValue(req.Context(), requestIDKey, "test-recovery")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "internal server error" {
		t.Fatalf("expected internal server error, got %q", body["error"])
	}
}

// --- Response format ---

func TestResponsesAreJSON(t *testing.T) {
	srv := testSetup(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodGet, "/admin/events"},
		{http.MethodGet, "/admin/channels"},
		{http.MethodGet, "/admin/skills"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Fatalf("expected Content-Type application/json, got %q", ct)
			}
		})
	}
}
