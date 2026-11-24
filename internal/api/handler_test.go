package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Churma16/portfolio-monitor/internal/monitor"
)

func TestServer_HealthAndMetrics(t *testing.T) {
	store := monitor.NewStore(10)
	server := NewServer(store, "")
	handler := server.Routes()

	// 1. Health check
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /health, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"status":"healthy"`) {
		t.Errorf("expected healthy body, got %s", rr.Body.String())
	}

	// 2. Metrics check
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /metrics, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "portfolio_monitor_uptime_seconds") {
		t.Errorf("expected prometheus metrics in response, got %s", rr.Body.String())
	}
}

func TestServer_EventsIngestion(t *testing.T) {
	store := monitor.NewStore(10)
	server := NewServer(store, "secret123")
	handler := server.Routes()

	// 1. Unauthorized when token is missing
	payload := `{"source":"go","type":"http_request","endpoint":"/api/projects","status_code":200,"duration_ms":10}`
	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}

	// 2. Authorized single event
	req = httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(payload))
	req.Header.Set("Authorization", "Bearer secret123")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	var res map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &res)
	if res["count"] != float64(1) {
		t.Errorf("expected count 1, got %v", res["count"])
	}

	// 3. Authorized batch events via /api/metrics alias
	batchPayload := `[
		{"source":"laravel","type":"http_request","endpoint":"/api/tags","status_code":200,"duration_ms":8},
		{"source":"express","type":"contact_form"}
	]`
	req = httptest.NewRequest(http.MethodPost, "/api/metrics", bytes.NewBufferString(batchPayload))
	req.Header.Set("Authorization", "Bearer secret123")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted for batch, got %d", rr.Code)
	}

	// 4. Invalid event validation
	badPayload := `{"source":"","type":"invalid"}`
	req = httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewBufferString(badPayload))
	req.Header.Set("Authorization", "Bearer secret123")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid event, got %d", rr.Code)
	}

	// 5. Stats endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/stats, got %d", rr.Code)
	}
	var stats monitor.StatsSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}
	if stats.TotalEvents != 3 {
		t.Errorf("expected 3 total recorded events, got %d", stats.TotalEvents)
	}
}
