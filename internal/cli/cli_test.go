package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/youmna-rabie/claude-pod/internal/config"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildChannels(t *testing.T) {
	cfgs := []config.ChannelConfig{
		{Name: "grafana-alerts", Type: "grafana", Auth: "secret"},
		{Name: "generic", Type: "dummy"},
		{Name: "unknown-type", Type: "whatever"},
	}

	channels := buildChannels(cfgs)

	if len(channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(channels))
	}

	// Grafana type should produce GrafanaChannel.
	if ch, ok := channels["grafana-alerts"]; !ok {
		t.Error("missing grafana-alerts channel")
	} else if ch.Name() != "grafana-alerts" {
		t.Errorf("expected name grafana-alerts, got %s", ch.Name())
	}

	// Dummy type should produce DummyChannel.
	if ch, ok := channels["generic"]; !ok {
		t.Error("missing generic channel")
	} else if ch.Name() != "generic" {
		t.Errorf("expected name generic, got %s", ch.Name())
	}

	// Unknown types fall back to DummyChannel.
	if ch, ok := channels["unknown-type"]; !ok {
		t.Error("missing unknown-type channel")
	} else if ch.Name() != "unknown-type" {
		t.Errorf("expected name unknown-type, got %s", ch.Name())
	}
}

func TestBuildChannelsEmpty(t *testing.T) {
	channels := buildChannels(nil)
	if len(channels) != 0 {
		t.Fatalf("expected 0 channels, got %d", len(channels))
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.LoggingConfig
	}{
		{"json format", config.LoggingConfig{Level: "info", Format: "json"}},
		{"text format", config.LoggingConfig{Level: "debug", Format: "text"}},
		{"warn level", config.LoggingConfig{Level: "warn", Format: "json"}},
		{"error level", config.LoggingConfig{Level: "error", Format: "text"}},
		{"default level", config.LoggingConfig{Level: "unknown", Format: "json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := newLogger(tt.cfg)
			if logger == nil {
				t.Fatal("expected non-nil logger")
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLogLevel(tt.input)
			if got.String() != tt.want {
				t.Errorf("parseLogLevel(%q) = %s, want %s", tt.input, got.String(), tt.want)
			}
		})
	}
}

func TestListChannelsCommand(t *testing.T) {
	cfg := writeTestConfig(t, `
server:
  port: 9090
channels:
  - name: test-chan
    type: dummy
`)

	// Temporarily set configPath for the command.
	old := configPath
	configPath = cfg
	defer func() { configPath = old }()

	err := listChannels(nil, nil)
	if err != nil {
		t.Fatalf("listChannels returned error: %v", err)
	}
}

func TestListChannelsCommandNoChannels(t *testing.T) {
	cfg := writeTestConfig(t, `
server:
  port: 9090
`)

	old := configPath
	configPath = cfg
	defer func() { configPath = old }()

	err := listChannels(nil, nil)
	if err != nil {
		t.Fatalf("listChannels returned error: %v", err)
	}
}

func TestListEventsCommand(t *testing.T) {
	eventsLimit = 10
	err := listEvents(nil, nil)
	if err != nil {
		t.Fatalf("listEvents returned error: %v", err)
	}
}

func TestListSkillsCommand(t *testing.T) {
	cfg := writeTestConfig(t, `
server:
  port: 9090
`)

	old := configPath
	configPath = cfg
	defer func() { configPath = old }()

	err := listSkills(nil, nil)
	if err != nil {
		t.Fatalf("listSkills returned error: %v", err)
	}
}

// Verify Channel interface conformance for buildChannels results.
func TestBuildChannelsInterfaceConformance(t *testing.T) {
	cfgs := []config.ChannelConfig{
		{Name: "g", Type: "grafana", Auth: "tok"},
		{Name: "d", Type: "dummy"},
	}

	channels := buildChannels(cfgs)
	for name, ch := range channels {
		// Each must satisfy types.Channel.
		var _ types.Channel = ch
		if ch.Name() != name {
			t.Errorf("channel name mismatch: key=%s, Name()=%s", name, ch.Name())
		}
	}
}
