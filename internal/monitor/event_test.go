package monitor

import (
	"testing"
	"time"
)

func TestEvent_Validate(t *testing.T) {
	tests := []struct {
		name      string
		event     Event
		wantErr   error
		checkTime bool
	}{
		{
			name: "valid minimal event",
			event: Event{
				Source: "laravel",
				Type:   "http_request",
			},
			wantErr:   nil,
			checkTime: true,
		},
		{
			name: "valid full event",
			event: Event{
				Source:     "Go",
				Type:       "Http_Request",
				Endpoint:   "/api/projects",
				StatusCode: 200,
				DurationMs: 12.5,
				Timestamp:  time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC),
			},
			wantErr:   nil,
			checkTime: false,
		},
		{
			name: "empty source",
			event: Event{
				Source: "  ",
				Type:   "http_request",
			},
			wantErr: ErrEmptySource,
		},
		{
			name: "empty type",
			event: Event{
				Source: "express",
				Type:   "",
			},
			wantErr: ErrEmptyType,
		},
		{
			name: "negative duration",
			event: Event{
				Source:     "spring",
				Type:       "http_request",
				DurationMs: -1.0,
			},
			wantErr: ErrNegativeDuration,
		},
		{
			name: "status code too low",
			event: Event{
				Source:     "laravel",
				Type:       "http_request",
				StatusCode: 99,
			},
			wantErr: ErrInvalidStatus,
		},
		{
			name: "status code too high",
			event: Event{
				Source:     "laravel",
				Type:       "http_request",
				StatusCode: 600,
			},
			wantErr: ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if err != tt.wantErr {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil {
				if tt.checkTime && tt.event.Timestamp.IsZero() {
					t.Errorf("expected auto-populated timestamp, got zero time")
				}
				if tt.event.Source != "laravel" && tt.event.Source != "go" && tt.event.Source != "express" && tt.event.Source != "spring" {
					t.Errorf("expected source to be normalized to lowercase, got %s", tt.event.Source)
				}
			}
		})
	}
}
