package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Churma16/portfolio-monitor/internal/api"
	"github.com/Churma16/portfolio-monitor/internal/monitor"
)

func TestMainServerRoutes(t *testing.T) {
	store := monitor.NewStore(10)
	srv := api.NewServer(store, "")
	handler := srv.Routes()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 from health endpoint, got %d", rr.Code)
	}
}
