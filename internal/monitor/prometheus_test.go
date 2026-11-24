package monitor

import (
	"strings"
	"testing"
)

func TestFormatPrometheus(t *testing.T) {
	store := NewStore(10)

	store.Record(Event{
		Source:     "go",
		Type:       "http_request",
		Endpoint:   "/api/messages",
		StatusCode: 200,
		DurationMs: 15.0,
	})
	store.Record(Event{
		Source:     "laravel",
		Type:       "contact_form",
	})

	output := FormatPrometheus(store)

	expectedMetrics := []string{
		"portfolio_monitor_uptime_seconds",
		"portfolio_monitor_events_received_total 2",
		"portfolio_events_total{source=\"go\",type=\"http_request\"} 1",
		"portfolio_events_total{source=\"laravel\",type=\"contact_form\"} 1",
		"portfolio_http_requests_total{source=\"go\",endpoint=\"/api/messages\",status=\"200\"} 1",
		"portfolio_http_request_duration_seconds_bucket{source=\"go\",endpoint=\"/api/messages\",le=\"0.025\"} 1",
		"portfolio_http_request_duration_seconds_sum{source=\"go\",endpoint=\"/api/messages\"} 0.015000",
		"portfolio_http_request_duration_seconds_count{source=\"go\",endpoint=\"/api/messages\"} 1",
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(output, m) {
			t.Errorf("expected prometheus output to contain %q, but missing.\nOutput:\n%s", m, output)
		}
	}
}
