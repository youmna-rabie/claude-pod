package channel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/youmna-rabie/claude-pod/internal/types"
)

func TestDummyChannel_Name(t *testing.T) {
	ch := NewDummyChannel("test-dummy")
	if ch.Name() != "test-dummy" {
		t.Fatalf("expected name %q, got %q", "test-dummy", ch.Name())
	}
}

func TestDummyChannel_ValidateRequest_POST(t *testing.T) {
	ch := NewDummyChannel("dummy")
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	if err := ch.ValidateRequest(r); err != nil {
		t.Fatalf("POST should be valid: %v", err)
	}
}

func TestDummyChannel_ValidateRequest_GET(t *testing.T) {
	ch := NewDummyChannel("dummy")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if err := ch.ValidateRequest(r); err == nil {
		t.Fatal("GET should be rejected")
	}
}

func TestDummyChannel_ParseRequest_ValidPayload(t *testing.T) {
	ch := NewDummyChannel("dummy")
	body := `{"alert":"firing"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("X-Custom", "value")

	ev, err := ch.ParseRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.ChannelID != "dummy" {
		t.Errorf("expected channel_id %q, got %q", "dummy", ev.ChannelID)
	}
	if string(ev.RawBody) != body {
		t.Errorf("expected raw_body %q, got %q", body, string(ev.RawBody))
	}
	if ev.Status != types.EventStatusReceived {
		t.Errorf("expected status %q, got %q", types.EventStatusReceived, ev.Status)
	}
	if ev.Headers["X-Custom"] != "value" {
		t.Errorf("expected header X-Custom=value, got %q", ev.Headers["X-Custom"])
	}
}

func TestDummyChannel_ParseRequest_EmptyBody(t *testing.T) {
	ch := NewDummyChannel("dummy")
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	ev, err := ch.ParseRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(ev.RawBody) != "" {
		t.Errorf("expected empty raw_body, got %q", string(ev.RawBody))
	}
}

func TestDummyChannel_ImplementsChannel(t *testing.T) {
	var _ types.Channel = (*DummyChannel)(nil)
}
