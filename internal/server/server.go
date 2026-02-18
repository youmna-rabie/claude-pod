package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/youmna-rabie/claude-pod/internal/agent"
	"github.com/youmna-rabie/claude-pod/internal/config"
	"github.com/youmna-rabie/claude-pod/internal/event"
	"github.com/youmna-rabie/claude-pod/internal/types"
)

// Server is the HTTP gateway that receives webhooks, stores events,
// forwards them to an agent, and exposes admin/health endpoints.
type Server struct {
	cfg      *config.Config
	store    event.Store
	channels map[string]types.Channel
	agent    agent.Client
	skills   []types.Skill
	router   chi.Router
	logger   *slog.Logger
}

// NewServer creates a Server wired with the given dependencies.
func NewServer(
	cfg *config.Config,
	store event.Store,
	channels map[string]types.Channel,
	agentClient agent.Client,
	skills []types.Skill,
	logger *slog.Logger,
) *Server {
	s := &Server{
		cfg:      cfg,
		store:    store,
		channels: channels,
		agent:    agentClient,
		skills:   skills,
		logger:   logger,
	}

	r := chi.NewRouter()
	r.Use(RequestID)
	r.Use(Logging(logger))
	r.Use(Recovery(logger))

	r.Post("/webhooks/{channel}", s.handleWebhook)
	r.Get("/health", s.handleHealth)
	r.Get("/admin/events", s.handleAdminEvents)
	r.Get("/admin/channels", s.handleAdminChannels)
	r.Get("/admin/skills", s.handleAdminSkills)

	s.router = r
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server on the configured host:port.
func (s *Server) ListenAndServe() error {
	addr := net.JoinHostPort(s.cfg.Server.Host, fmt.Sprintf("%d", s.cfg.Server.Port))
	s.logger.Info("server starting", "addr", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

// handleWebhook processes POST /webhooks/{channel}.
// Pipeline: validate → parse → store → forward → respond.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	channelName := chi.URLParam(r, "channel")

	ch, ok := s.channels[channelName]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("unknown channel: %s", channelName),
		})
		return
	}

	// Validate
	if err := ch.ValidateRequest(r); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Parse
	evt, err := ch.ParseRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Store
	if err := s.store.Save(*evt); err != nil {
		s.logger.Error("failed to save event", "error", err, "event_id", evt.ID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to store event",
		})
		return
	}

	// Forward
	envelope := types.EventEnvelope{
		Version:   "1",
		Event:     *evt,
		Channel:   channelName,
		Skills:    s.skills,
		Timestamp: time.Now(),
	}

	resp, err := s.agent.Forward(context.Background(), envelope)
	if err != nil {
		_ = s.store.UpdateStatus(evt.ID, types.EventStatusFailed)
		s.logger.Error("agent forward failed", "error", err, "event_id", evt.ID)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "agent forwarding failed",
		})
		return
	}

	_ = s.store.UpdateStatus(evt.ID, types.EventStatusForwarded)

	writeJSON(w, http.StatusOK, resp)
}

// handleHealth responds to GET /health with a simple liveness check.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminEvents responds to GET /admin/events with recent events.
func (s *Server) handleAdminEvents(w http.ResponseWriter, _ *http.Request) {
	events, err := s.store.List(50, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to list events",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"count":  len(events),
	})
}

// handleAdminChannels responds to GET /admin/channels with configured channels.
func (s *Server) handleAdminChannels(w http.ResponseWriter, _ *http.Request) {
	names := make([]string, 0, len(s.channels))
	for name := range s.channels {
		names = append(names, name)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channels": names,
		"count":    len(names),
	})
}

// handleAdminSkills responds to GET /admin/skills with registered skills.
func (s *Server) handleAdminSkills(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"skills": s.skills,
		"count":  len(s.skills),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
