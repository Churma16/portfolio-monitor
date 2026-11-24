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

// Store maintains thread-safe aggregated telemetry and Prometheus metrics in memory.
type Store struct {
	mu sync.RWMutex

	startTime      time.Time
	eventsReceived uint64

	// Metrics aggregates
	eventsByType   map[string]uint64 // key: "source|type"
	requestsTotal  map[string]uint64 // key: "source|endpoint|status"
	durationSumMs  map[string]float64// key: "source|endpoint"
	durationCount  map[string]uint64 // key: "source|endpoint"
	durationBucket map[string]map[float64]uint64 // key: "source|endpoint" -> bucket_le -> count

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
		eventsByType:   make(map[string]uint64),
		requestsTotal:  make(map[string]uint64),
		durationSumMs:  make(map[string]float64),
		durationCount:  make(map[string]uint64),
		durationBucket: make(map[string]map[float64]uint64),
		recentEvents:   make([]Event, 0, maxRecentEvents),
		maxRecent:      maxRecentEvents,
	}
}

// Record updates metrics counters and histograms safely from an Event.
func (s *Store) Record(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventsReceived++

	// 1. Event count by source and type
	typeKey := fmt.Sprintf("%s|%s", event.Source, event.Type)
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

		reqKey := fmt.Sprintf("%s|%s|%d", event.Source, endpoint, status)
		s.requestsTotal[reqKey]++

		if event.DurationMs > 0 {
			durKey := fmt.Sprintf("%s|%s", event.Source, endpoint)
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

	// 3. Append to ring buffer
	if len(s.recentEvents) >= s.maxRecent {
		s.recentEvents = append(s.recentEvents[1:], event)
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
		eventsCopy[k] = v
	}

	reqsCopy := make(map[string]uint64, len(s.requestsTotal))
	for k, v := range s.requestsTotal {
		reqsCopy[k] = v
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
	s.eventsByType = make(map[string]uint64)
	s.requestsTotal = make(map[string]uint64)
	s.durationSumMs = make(map[string]float64)
	s.durationCount = make(map[string]uint64)
	s.durationBucket = make(map[string]map[float64]uint64)
	s.recentEvents = make([]Event, 0, s.maxRecent)
}
