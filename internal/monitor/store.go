package monitor

import (
	"fmt"
	"sync"
	"time"
)

// Standard duration histogram buckets in milliseconds (5ms up to 5000ms)
var DefaultDurationBuckets = []float64{
	5.0, 10.0, 25.0, 50.0, 100.0, 250.0, 500.0, 1000.0, 2500.0, 5000.0,
}

// EventKey identifies an aggregated event by source, type, and optional breakdown labels.
type EventKey struct {
	Source  string
	Type    string
	Target  string
	Project string
}

// RequestKey identifies aggregated HTTP requests by source, endpoint, and status.
type RequestKey struct {
	Source     string
	Endpoint   string
	StatusCode int
}

// DurationKey identifies duration metrics by source and endpoint.
type DurationKey struct {
	Source   string
	Endpoint string
}

// Store maintains thread-safe aggregated telemetry and Prometheus metrics in memory.
type Store struct {
	mu sync.RWMutex

	startTime      time.Time
	eventsReceived uint64

	// Metrics aggregates using native hashable struct keys (immune to delimiter collision)
	eventsByType   map[EventKey]uint64
	requestsTotal  map[RequestKey]uint64
	durationSumMs  map[DurationKey]float64
	durationCount  map[DurationKey]uint64
	durationBucket map[DurationKey]map[float64]uint64

	// Ring buffer of recent events for human/API inspection
	recentEvents   []Event
	maxRecent      int
}

// NewStore initializes a new in-memory metrics store.
func NewStore(maxRecentEvents int) *Store {
	if maxRecentEvents <= 0 {
		maxRecentEvents = 100
	}
	return &Store{
		startTime:      time.Now().UTC(),
		eventsByType:   make(map[EventKey]uint64),
		requestsTotal:  make(map[RequestKey]uint64),
		durationSumMs:  make(map[DurationKey]float64),
		durationCount:  make(map[DurationKey]uint64),
		durationBucket: make(map[DurationKey]map[float64]uint64),
		recentEvents:   make([]Event, 0, maxRecentEvents),
		maxRecent:      maxRecentEvents,
	}
}

// Record updates metrics counters and histograms safely from an Event.
func (s *Store) Record(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventsReceived++

	// 1. Event count by source, type, and optional metadata breakdown
	var target, project string
	if event.Metadata != nil {
		target = event.Metadata["target"]
		project = event.Metadata["project"]
	}
	typeKey := EventKey{Source: event.Source, Type: event.Type, Target: target, Project: project}
	s.eventsByType[typeKey]++

	// 2. HTTP Request metrics if applicable
	if event.Type == "http_request" || event.Endpoint != "" || event.StatusCode > 0 {
		endpoint := event.Endpoint
		if endpoint == "" {
			endpoint = "/"
		}
		status := event.StatusCode
		if status == 0 {
			status = 200
		}

		reqKey := RequestKey{Source: event.Source, Endpoint: endpoint, StatusCode: status}
		s.requestsTotal[reqKey]++

		if event.DurationMs > 0 {
			durKey := DurationKey{Source: event.Source, Endpoint: endpoint}
			s.durationSumMs[durKey] += event.DurationMs
			s.durationCount[durKey]++

			if _, ok := s.durationBucket[durKey]; !ok {
				s.durationBucket[durKey] = make(map[float64]uint64)
			}
			for _, le := range DefaultDurationBuckets {
				if event.DurationMs <= le {
					s.durationBucket[durKey][le]++
				}
			}
		}
	}

	// 3. Append to ring buffer using in-place shift to avoid slice reallocation churn
	if len(s.recentEvents) >= s.maxRecent {
		copy(s.recentEvents, s.recentEvents[1:])
		s.recentEvents[len(s.recentEvents)-1] = event
	} else {
		s.recentEvents = append(s.recentEvents, event)
	}
}

// StatsSummary represents high-level telemetry data for JSON inspection.
type StatsSummary struct {
	UptimeSeconds  float64           `json:"uptime_seconds"`
	TotalEvents    uint64            `json:"total_events"`
	EventsByType   map[string]uint64 `json:"events_by_type"`
	RequestsTotal  map[string]uint64 `json:"requests_total"`
	RecentEvents   []Event           `json:"recent_events"`
}

// Stats returns a thread-safe snapshot of aggregated telemetry data.
func (s *Store) Stats() StatsSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	eventsCopy := make(map[string]uint64, len(s.eventsByType))
	for k, v := range s.eventsByType {
		keyStr := fmt.Sprintf("%s|%s", k.Source, k.Type)
		if k.Target != "" {
			keyStr += "|" + k.Target
		}
		if k.Project != "" {
			keyStr += "|" + k.Project
		}
		eventsCopy[keyStr] = v
	}

	reqsCopy := make(map[string]uint64, len(s.requestsTotal))
	for k, v := range s.requestsTotal {
		reqsCopy[fmt.Sprintf("%s|%s|%d", k.Source, k.Endpoint, k.StatusCode)] = v
	}

	recentCopy := make([]Event, len(s.recentEvents))
	copy(recentCopy, s.recentEvents)

	return StatsSummary{
		UptimeSeconds: time.Since(s.startTime).Seconds(),
		TotalEvents:   s.eventsReceived,
		EventsByType:  eventsCopy,
		RequestsTotal: reqsCopy,
		RecentEvents:  recentCopy,
	}
}

// Reset clears all recorded metrics (useful for testing).
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startTime = time.Now().UTC()
	s.eventsReceived = 0
	s.eventsByType = make(map[EventKey]uint64)
	s.requestsTotal = make(map[RequestKey]uint64)
	s.durationSumMs = make(map[DurationKey]float64)
	s.durationCount = make(map[DurationKey]uint64)
	s.durationBucket = make(map[DurationKey]map[float64]uint64)
	s.recentEvents = make([]Event, 0, s.maxRecent)
}
