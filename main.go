package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Churma16/portfolio-monitor/internal/api"
	"github.com/Churma16/portfolio-monitor/internal/monitor"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	authToken := os.Getenv("MONITOR_AUTH_TOKEN")

	store := monitor.NewStore(100)
	server := api.NewServer(store, authToken)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server run context
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] portfolio-monitor listening on port %s", port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Graceful shutdown listener
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("[FATAL] Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("[INFO] Received signal %v, initiating graceful shutdown...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("[WARN] Error during graceful shutdown: %v", err)
			_ = httpServer.Close()
		}
		log.Println("[INFO] Server stopped gracefully.")
	}
}
