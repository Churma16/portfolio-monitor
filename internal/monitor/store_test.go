package monitor

import (
	"sync"
	"testing"
)

func TestStore_RecordAndStats(t *testing.T) {
	store := NewStore(3)

	ev1 := Event{Source: "go", Type: "http_request", Endpoint: "/api/projects", StatusCode: 200, DurationMs: 15.0}
	ev2 := Event{Source: "go", Type: "http_request", Endpoint: "/api/projects", StatusCode: 500, DurationMs: 30.0}
	ev3 := Event{Source: "laravel", Type: "contact_form"}
	ev4 := Event{Source: "express", Type: "cv_download"}

	store.Record(ev1)
	store.Record(ev2)
	store.Record(ev3)
	// Verify capacity before and after exceeding maxRecent
	initialCap := cap(store.recentEvents)
	store.Record(ev4)
	if cap(store.recentEvents) != initialCap {
		t.Errorf("expected slice capacity to remain stable (%d), but changed to %d", initialCap, cap(store.recentEvents))
	}

	stats := store.Stats()
	if stats.TotalEvents != 4 {
		t.Errorf("expected 4 total events, got %d", stats.TotalEvents)
	}

	// Check ring buffer capacity (maxRecent was 3, so ev1 is pushed out)
	if len(stats.RecentEvents) != 3 {
		t.Errorf("expected recent events length 3, got %d", len(stats.RecentEvents))
	}
	if stats.RecentEvents[0].Source != "go" || stats.RecentEvents[0].StatusCode != 500 {
		t.Errorf("expected first element in ring buffer to be ev2, got %+v", stats.RecentEvents[0])
	}
	if stats.RecentEvents[2].Source != "express" {
		t.Errorf("expected last element in ring buffer to be ev4, got %+v", stats.RecentEvents[2])
	}

	// Check request counts
	if stats.RequestsTotal["go|/api/projects|200"] != 1 {
		t.Errorf("expected 1 200 request, got %d", stats.RequestsTotal["go|/api/projects|200"])
	}
	if stats.RequestsTotal["go|/api/projects|500"] != 1 {
		t.Errorf("expected 1 500 request, got %d", stats.RequestsTotal["go|/api/projects|500"])
	}

	// Check event types
	if stats.EventsByType["laravel|contact_form"] != 1 {
		t.Errorf("expected 1 contact_form event, got %d", stats.EventsByType["laravel|contact_form"])
	}
	if stats.EventsByType["express|cv_download"] != 1 {
		t.Errorf("expected 1 cv_download event, got %d", stats.EventsByType["express|cv_download"])
	}

	// Reset
	store.Reset()
	resetStats := store.Stats()
	if resetStats.TotalEvents != 0 || len(resetStats.RecentEvents) != 0 {
		t.Errorf("expected store to be cleared after Reset")
	}
}

func TestStore_ConcurrentRecording(t *testing.T) {
	store := NewStore(100)
	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				store.Record(Event{
					Source:     "spring",
					Type:       "http_request",
					Endpoint:   "/api/health",
					StatusCode: 200,
					DurationMs: float64(j),
				})
			}
		}(i)
	}

	wg.Wait()
	stats := store.Stats()
	expectedTotal := uint64(workers * iterations)
	if stats.TotalEvents != expectedTotal {
		t.Fatalf("expected total events %d, got %d", expectedTotal, stats.TotalEvents)
	}
}
