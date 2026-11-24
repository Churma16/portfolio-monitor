package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Churma16/portfolio-monitor/internal/monitor"
)

// Server encapsulates the HTTP API and metrics store.
type Server struct {
	store     *monitor.Store
	authToken string
	startTime time.Time
}

// NewServer initializes a new API server.
func NewServer(store *monitor.Store, authToken string) *Server {
	return &Server{
		store:     store,
		authToken: strings.TrimSpace(authToken),
		startTime: time.Now().UTC(),
	}
}

// Routes sets up the HTTP mux with all endpoints and middleware.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	mux.HandleFunc("GET /api/stats", s.handleStats)

	mux.HandleFunc("POST /api/events", s.handleEvents)
	mux.HandleFunc("POST /api/metrics", s.handleEvents) // alias

	return s.loggingMiddleware(s.recoverMiddleware(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":         "healthy",
		"uptime_seconds": time.Since(s.startTime).Seconds(),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(monitor.FormatPrometheus(s.store)))
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s.store.Stats())
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if s.authToken != "" {
		authHeader := r.Header.Get("Authorization")
		expected := "Bearer " + s.authToken
		if authHeader != expected {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	// Limit body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"payload too large or unreadable"}`, http.StatusBadRequest)
		return
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		http.Error(w, `{"error":"empty request body"}`, http.StatusBadRequest)
		return
	}

	var events []monitor.Event
	if strings.HasPrefix(trimmed, "[") {
		// Array of events
		if err := json.Unmarshal([]byte(trimmed), &events); err != nil {
			http.Error(w, `{"error":"invalid JSON array"}`, http.StatusBadRequest)
			return
		}
	} else {
		// Single event
		var single monitor.Event
		if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
			http.Error(w, `{"error":"invalid JSON object"}`, http.StatusBadRequest)
			return
		}
		events = []monitor.Event{single}
	}

	if len(events) == 0 {
		http.Error(w, `{"error":"no events provided"}`, http.StatusBadRequest)
		return
	}

	// Validate all events
	for i := range events {
		if err := events[i].Validate(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   "validation failed: " + err.Error(),
				"index":   i,
			})
			return
		}
	}

	// Record all validated events
	for _, ev := range events {
		s.store.Record(ev)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "recorded",
		"count":  len(events),
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Suppress verbose logging for frequent /metrics and /health checks
		if r.URL.Path != "/metrics" && r.URL.Path != "/health" {
			log.Printf("[HTTP] %s %s %d (latency: %v)", r.Method, r.URL.Path, rec.statusCode, time.Since(start))
		}
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
