package channel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/youmna-rabie/claude-pod/internal/types"
)

func TestGrafanaChannel_Name(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	if ch.Name() != "grafana" {
		t.Fatalf("expected name %q, got %q", "grafana", ch.Name())
	}
}

func TestGrafanaChannel_ValidateRequest_Valid(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer secret")

	if err := ch.ValidateRequest(r); err != nil {
		t.Fatalf("valid request should pass: %v", err)
	}
}

func TestGrafanaChannel_ValidateRequest_WrongMethod(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer secret")

	if err := ch.ValidateRequest(r); err == nil {
		t.Fatal("GET should be rejected")
	}
}

func TestGrafanaChannel_ValidateRequest_WrongContentType(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "text/plain")
	r.Header.Set("Authorization", "Bearer secret")

	err := ch.ValidateRequest(r)
	if err == nil {
		t.Fatal("wrong Content-Type should be rejected")
	}
	if !strings.Contains(err.Error(), "Content-Type") {
		t.Errorf("error should mention Content-Type: %v", err)
	}
}

func TestGrafanaChannel_ValidateRequest_MissingAuth(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")

	err := ch.ValidateRequest(r)
	if err == nil {
		t.Fatal("missing auth should be rejected")
	}
	if !strings.Contains(err.Error(), "authorization") {
		t.Errorf("error should mention authorization: %v", err)
	}
}

func TestGrafanaChannel_ValidateRequest_WrongAuth(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer wrong-token")

	err := ch.ValidateRequest(r)
	if err == nil {
		t.Fatal("wrong auth token should be rejected")
	}
}

func TestGrafanaChannel_ValidateRequest_NoAuthRequired(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")

	if err := ch.ValidateRequest(r); err != nil {
		t.Fatalf("no-auth channel should pass without token: %v", err)
	}
}

func TestGrafanaChannel_ParseRequest_ValidJSON(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	body := `{"status":"firing","alerts":[{"labels":{"alertname":"CPU"}}]}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	ev, err := ch.ParseRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.ChannelID != "grafana" {
		t.Errorf("expected channel_id %q, got %q", "grafana", ev.ChannelID)
	}
	if string(ev.RawBody) != body {
		t.Errorf("expected raw_body %q, got %q", body, string(ev.RawBody))
	}
	if ev.Status != types.EventStatusReceived {
		t.Errorf("expected status %q, got %q", types.EventStatusReceived, ev.Status)
	}
}

func TestGrafanaChannel_ParseRequest_InvalidJSON(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))

	_, err := ch.ParseRequest(r)
	if err == nil {
		t.Fatal("invalid JSON should be rejected")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("error should mention invalid JSON: %v", err)
	}
}

func TestGrafanaChannel_ParseRequest_BodyTooLarge(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	// Create body just over 1MB
	big := strings.Repeat("x", maxBodySize+1)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(big))

	_, err := ch.ParseRequest(r)
	if err == nil {
		t.Fatal("body exceeding 1MB should be rejected")
	}
	if !strings.Contains(err.Error(), "1MB") {
		t.Errorf("error should mention 1MB limit: %v", err)
	}
}

func TestGrafanaChannel_ParseRequest_BodyExactlyAtLimit(t *testing.T) {
	ch := NewGrafanaChannel("grafana", "secret")
	// Build a valid JSON body exactly at the limit
	// Use a JSON string with padding
	padding := strings.Repeat("a", maxBodySize-4) // 4 for `"..."` wrapper: {"x":"..."}
	body := `{"x":"` + padding + `"}`
	// This will exceed due to JSON framing; let's just test a body at exactly maxBodySize
	body = `"` + strings.Repeat("a", maxBodySize-2) + `"`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	ev, err := ch.ParseRequest(r)
	if err != nil {
		t.Fatalf("body at exactly 1MB should be accepted: %v", err)
	}
	if len(ev.RawBody) != maxBodySize {
		t.Errorf("expected body length %d, got %d", maxBodySize, len(ev.RawBody))
	}
}

func TestGrafanaChannel_ImplementsChannel(t *testing.T) {
	var _ types.Channel = (*GrafanaChannel)(nil)
}
