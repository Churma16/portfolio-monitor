package monitor

import (
	"errors"
	"strings"
	"time"
)

// Event represents a telemetry or business event dispatched by any portfolio backend or frontend.
type Event struct {
	Source     string            `json:"source"`               // e.g. "go", "express", "spring", "laravel", "frontend"
	Type       string            `json:"type"`                 // e.g. "http_request", "contact_form", "cv_download", "toggle_backend"
	Endpoint   string            `json:"endpoint,omitempty"`   // e.g. "/api/projects", "/api/messages"
	StatusCode int               `json:"status_code,omitempty"`// e.g. 200, 201, 400, 429, 500
	DurationMs float64           `json:"duration_ms,omitempty"`// in milliseconds
	Timestamp  time.Time         `json:"timestamp,omitempty"`  // event timestamp
	Metadata   map[string]string `json:"metadata,omitempty"`   // optional contextual tags
}

var (
	ErrEmptySource      = errors.New("event source cannot be empty")
	ErrEmptyType        = errors.New("event type cannot be empty")
	ErrNegativeDuration = errors.New("duration cannot be negative")
	ErrInvalidStatus    = errors.New("status code must be between 100 and 599 when specified")
)

// Validate ensures required fields are set and values conform to specification.
func (e *Event) Validate() error {
	e.Source = strings.TrimSpace(strings.ToLower(e.Source))
	e.Type = strings.TrimSpace(strings.ToLower(e.Type))
	e.Endpoint = strings.TrimSpace(e.Endpoint)

	if e.Source == "" {
		return ErrEmptySource
	}
	if e.Type == "" {
		return ErrEmptyType
	}
	if e.DurationMs < 0 {
		return ErrNegativeDuration
	}
	if e.StatusCode != 0 && (e.StatusCode < 100 || e.StatusCode > 599) {
		return ErrInvalidStatus
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	return nil
}
