package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level gateway configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Agent    AgentConfig    `yaml:"agent"`
	Channels []ChannelConfig `yaml:"channels"`
	Skills   SkillsConfig   `yaml:"skills"`
	Store    StoreConfig    `yaml:"store"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP listener settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AgentConfig holds upstream agent connection settings.
type AgentConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

// ChannelConfig describes a single inbound channel.
type ChannelConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Auth string `yaml:"auth"`
}

// SkillsConfig holds skill discovery settings.
type SkillsConfig struct {
	Dirs      []string `yaml:"dirs"`
	Allowlist []string `yaml:"allowlist"`
}

// StoreConfig holds message/session store settings.
type StoreConfig struct {
	Type     string `yaml:"type"`
	Capacity int    `yaml:"capacity"`
}

// LoggingConfig holds structured logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// defaults applies sane defaults to zero-valued fields.
func (c *Config) defaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Agent.Timeout == 0 {
		c.Agent.Timeout = 30 * time.Second
	}
	if c.Store.Type == "" {
		c.Store.Type = "memory"
	}
	if c.Store.Capacity == 0 {
		c.Store.Capacity = 1000
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

// validate checks required fields and value constraints.
func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Agent.Timeout < 0 {
		return fmt.Errorf("agent.timeout must be non-negative")
	}
	for i, ch := range c.Channels {
		if ch.Name == "" {
			return fmt.Errorf("channels[%d].name is required", i)
		}
		if ch.Type == "" {
			return fmt.Errorf("channels[%d].type is required", i)
		}
	}
	if c.Store.Capacity < 0 {
		return fmt.Errorf("store.capacity must be non-negative")
	}
	return nil
}

// expandEnv replaces ${VAR} references in secret-bearing fields with
// environment variable values. This allows keeping secrets out of YAML.
func (c *Config) expandEnv() {
	c.Agent.URL = os.ExpandEnv(c.Agent.URL)
	for i := range c.Channels {
		c.Channels[i].Auth = os.ExpandEnv(c.Channels[i].Auth)
	}
}

// Load reads a YAML config file, applies defaults, expands env vars, and validates.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	cfg.defaults()
	cfg.expandEnv()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}
