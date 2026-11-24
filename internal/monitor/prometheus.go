package monitor

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// FormatPrometheus generates standard Prometheus text format metrics from Store.
func FormatPrometheus(s *Store) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var b strings.Builder

	// 1. Process info & uptime
	b.WriteString("# HELP portfolio_monitor_uptime_seconds Total runtime of portfolio-monitor in seconds\n")
	b.WriteString("# TYPE portfolio_monitor_uptime_seconds gauge\n")
	b.WriteString(fmt.Sprintf("portfolio_monitor_uptime_seconds %.2f\n\n", time.Since(s.startTime).Seconds()))

	b.WriteString("# HELP portfolio_monitor_events_received_total Total telemetry events received by portfolio-monitor\n")
	b.WriteString("# TYPE portfolio_monitor_events_received_total counter\n")
	b.WriteString(fmt.Sprintf("portfolio_monitor_events_received_total %d\n\n", s.eventsReceived))

	// 2. Events by type counter
	b.WriteString("# HELP portfolio_events_total Total portfolio events by source and type\n")
	b.WriteString("# TYPE portfolio_events_total counter\n")
	
	typeKeys := make([]string, 0, len(s.eventsByType))
	for k := range s.eventsByType {
		typeKeys = append(typeKeys, k)
	}
	sort.Strings(typeKeys)

	for _, k := range typeKeys {
		parts := strings.Split(k, "|")
		if len(parts) == 2 {
			source := escapeLabel(parts[0])
			eventType := escapeLabel(parts[1])
			count := s.eventsByType[k]
			b.WriteString(fmt.Sprintf("portfolio_events_total{source=\"%s\",type=\"%s\"} %d\n", source, eventType, count))
		}
	}
	b.WriteString("\n")

	// 3. HTTP requests total counter
	b.WriteString("# HELP portfolio_http_requests_total Total number of HTTP requests processed by portfolio backends\n")
	b.WriteString("# TYPE portfolio_http_requests_total counter\n")

	reqKeys := make([]string, 0, len(s.requestsTotal))
	for k := range s.requestsTotal {
		reqKeys = append(reqKeys, k)
	}
	sort.Strings(reqKeys)

	for _, k := range reqKeys {
		parts := strings.Split(k, "|")
		if len(parts) == 3 {
			source := escapeLabel(parts[0])
			endpoint := escapeLabel(parts[1])
			status := escapeLabel(parts[2])
			count := s.requestsTotal[k]
			b.WriteString(fmt.Sprintf("portfolio_http_requests_total{source=\"%s\",endpoint=\"%s\",status=\"%s\"} %d\n", source, endpoint, status, count))
		}
	}
	b.WriteString("\n")

	// 4. Request duration histogram & summary
	b.WriteString("# HELP portfolio_http_request_duration_seconds HTTP request duration in seconds\n")
	b.WriteString("# TYPE portfolio_http_request_duration_seconds histogram\n")

	durKeys := make([]string, 0, len(s.durationSumMs))
	for k := range s.durationSumMs {
		durKeys = append(durKeys, k)
	}
	sort.Strings(durKeys)

	for _, k := range durKeys {
		parts := strings.Split(k, "|")
		if len(parts) == 2 {
			source := escapeLabel(parts[0])
			endpoint := escapeLabel(parts[1])
			sumSeconds := s.durationSumMs[k] / 1000.0
			count := s.durationCount[k]

			buckets := s.durationBucket[k]
			for _, le := range DefaultDurationBuckets {
				bucketCount := uint64(0)
				if buckets != nil {
					bucketCount = buckets[le]
				}
				leSeconds := le / 1000.0
				b.WriteString(fmt.Sprintf("portfolio_http_request_duration_seconds_bucket{source=\"%s\",endpoint=\"%s\",le=\"%.3f\"} %d\n", source, endpoint, leSeconds, bucketCount))
			}
			b.WriteString(fmt.Sprintf("portfolio_http_request_duration_seconds_bucket{source=\"%s\",endpoint=\"%s\",le=\"+Inf\"} %d\n", source, endpoint, count))
			b.WriteString(fmt.Sprintf("portfolio_http_request_duration_seconds_sum{source=\"%s\",endpoint=\"%s\"} %.6f\n", source, endpoint, sumSeconds))
			b.WriteString(fmt.Sprintf("portfolio_http_request_duration_seconds_count{source=\"%s\",endpoint=\"%s\"} %d\n", source, endpoint, count))
		}
	}

	return b.String()
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
