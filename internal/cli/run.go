package cli

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/youmna-rabie/claude-pod/internal/agent"
	"github.com/youmna-rabie/claude-pod/internal/channel"
	"github.com/youmna-rabie/claude-pod/internal/config"
	"github.com/youmna-rabie/claude-pod/internal/event"
	"github.com/youmna-rabie/claude-pod/internal/server"
	"github.com/youmna-rabie/claude-pod/internal/skill"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the gateway HTTP server",
	RunE:  runGateway,
}

func runGateway(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg.Logging)

	store, err := event.NewMemoryStore(cfg.Store.Capacity)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	channels := buildChannels(cfg.Channels)

	agentClient := &agent.StubClient{Logger: logger}

	reg := &skill.Registry{}
	if len(cfg.Skills.Dirs) > 0 {
		if err := reg.Scan(cfg.Skills.Dirs); err != nil {
			logger.Warn("skill scan error", "error", err)
		}
	}
	skills := reg.Filter(cfg.Skills.Allowlist)

	srv := server.NewServer(cfg, store, channels, agentClient, skills, logger)

	addr := net.JoinHostPort(cfg.Server.Host, fmt.Sprintf("%d", cfg.Server.Port))
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", addr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		logger.Info("shutting down gracefully")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	logger.Info("server stopped")
	return nil
}

func buildChannels(cfgs []config.ChannelConfig) map[string]types.Channel {
	channels := make(map[string]types.Channel, len(cfgs))
	for _, ch := range cfgs {
		switch ch.Type {
		case "grafana":
			channels[ch.Name] = channel.NewGrafanaChannel(ch.Name, ch.Auth)
		default:
			channels[ch.Name] = channel.NewDummyChannel(ch.Name)
		}
	}
	return channels
}

func newLogger(cfg config.LoggingConfig) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: parseLogLevel(cfg.Level)}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
