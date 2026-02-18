package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_ValidFull(t *testing.T) {
	yaml := `
server:
  host: "127.0.0.1"
  port: 9090
agent:
  url: "http://localhost:3000"
  timeout: 10s
channels:
  - name: slack
    type: websocket
    auth: "xoxb-token"
  - name: discord
    type: http
    auth: "bot-token"
skills:
  dirs:
    - "./skills"
    - "/opt/skills"
  allowlist:
    - "summarize"
    - "search"
store:
  type: redis
  capacity: 5000
logging:
  level: debug
  format: text
`
	cfg, err := Load(writeTemp(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Server
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("server.host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("server.port = %d, want %d", cfg.Server.Port, 9090)
	}

	// Agent
	if cfg.Agent.URL != "http://localhost:3000" {
		t.Errorf("agent.url = %q, want %q", cfg.Agent.URL, "http://localhost:3000")
	}
	if cfg.Agent.Timeout != 10*time.Second {
		t.Errorf("agent.timeout = %v, want %v", cfg.Agent.Timeout, 10*time.Second)
	}

	// Channels
	if len(cfg.Channels) != 2 {
		t.Fatalf("channels len = %d, want 2", len(cfg.Channels))
	}
	if cfg.Channels[0].Name != "slack" {
		t.Errorf("channels[0].name = %q, want %q", cfg.Channels[0].Name, "slack")
	}
	if cfg.Channels[1].Type != "http" {
		t.Errorf("channels[1].type = %q, want %q", cfg.Channels[1].Type, "http")
	}

	// Skills
	if len(cfg.Skills.Dirs) != 2 {
		t.Errorf("skills.dirs len = %d, want 2", len(cfg.Skills.Dirs))
	}
	if len(cfg.Skills.Allowlist) != 2 {
		t.Errorf("skills.allowlist len = %d, want 2", len(cfg.Skills.Allowlist))
	}

	// Store
	if cfg.Store.Type != "redis" {
		t.Errorf("store.type = %q, want %q", cfg.Store.Type, "redis")
	}
	if cfg.Store.Capacity != 5000 {
		t.Errorf("store.capacity = %d, want %d", cfg.Store.Capacity, 5000)
	}

	// Logging
	if cfg.Logging.Level != "debug" {
		t.Errorf("logging.level = %q, want %q", cfg.Logging.Level, "debug")
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("logging.format = %q, want %q", cfg.Logging.Format, "text")
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Minimal YAML — everything should get defaults
	cfg, err := Load(writeTemp(t, "{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("default server.host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default server.port = %d, want %d", cfg.Server.Port, 8080)
	}
	if cfg.Agent.Timeout != 30*time.Second {
		t.Errorf("default agent.timeout = %v, want %v", cfg.Agent.Timeout, 30*time.Second)
	}
	if cfg.Store.Type != "memory" {
		t.Errorf("default store.type = %q, want %q", cfg.Store.Type, "memory")
	}
	if cfg.Store.Capacity != 1000 {
		t.Errorf("default store.capacity = %d, want %d", cfg.Store.Capacity, 1000)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("default logging.level = %q, want %q", cfg.Logging.Level, "info")
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("default logging.format = %q, want %q", cfg.Logging.Format, "json")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	_, err := Load(writeTemp(t, "{{{{not yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_ValidationError_BadPort(t *testing.T) {
	yaml := `
server:
  port: 99999
`
	_, err := Load(writeTemp(t, yaml))
	if err == nil {
		t.Fatal("expected validation error for bad port, got nil")
	}
}

func TestLoad_ValidationError_ChannelMissingName(t *testing.T) {
	yaml := `
channels:
  - type: websocket
    auth: "token"
`
	_, err := Load(writeTemp(t, yaml))
	if err == nil {
		t.Fatal("expected validation error for channel missing name, got nil")
	}
}

func TestLoad_ValidationError_ChannelMissingType(t *testing.T) {
	yaml := `
channels:
  - name: slack
    auth: "token"
`
	_, err := Load(writeTemp(t, yaml))
	if err == nil {
		t.Fatal("expected validation error for channel missing type, got nil")
	}
}

func TestLoad_EnvVarExpansion(t *testing.T) {
	t.Setenv("TEST_AGENT_URL", "http://secret-agent:4000")
	t.Setenv("TEST_SLACK_TOKEN", "xoxb-secret-123")

	yaml := `
agent:
  url: "${TEST_AGENT_URL}"
channels:
  - name: slack
    type: websocket
    auth: "${TEST_SLACK_TOKEN}"
`
	cfg, err := Load(writeTemp(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Agent.URL != "http://secret-agent:4000" {
		t.Errorf("agent.url = %q, want %q", cfg.Agent.URL, "http://secret-agent:4000")
	}
	if cfg.Channels[0].Auth != "xoxb-secret-123" {
		t.Errorf("channels[0].auth = %q, want %q", cfg.Channels[0].Auth, "xoxb-secret-123")
	}
}
