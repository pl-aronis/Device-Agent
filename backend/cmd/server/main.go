package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/api"
	"backend/internal/middleware"
	"backend/internal/storage"
)

const (
	defaultAddr      = "0.0.0.0:8080"
	defaultStorePath = "data/devices.json"
	shutdownTimeout  = 30 * time.Second
)

func main() {
	// Initialize logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("========== BACKEND SERVER STARTUP ==========")

	// Get configuration from environment
	addr := os.Getenv("BACKEND_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	storePath := os.Getenv("BACKEND_STORE_PATH")
	if storePath == "" {
		storePath = defaultStorePath
	}

	log.Printf("[CONFIG] Server address: %s", addr)
	log.Printf("[CONFIG] Storage path: %s", storePath)

	// Initialize storage
	store, err := storage.NewFileStore(storePath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create HTTP router
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/register", api.RegisterHandler(store))
	mux.HandleFunc("/api/heartbeat", api.HeartbeatHandler(store))
	mux.HandleFunc("/api/recovery-key", api.RecoveryKeyHandler(store))
	mux.HandleFunc("/admin/set", api.AdminSetHandler(store))
	mux.HandleFunc("/admin/status", api.AdminStatusHandler(store))
	mux.HandleFunc("/ping", api.HealthHandler())
	mux.HandleFunc("/health", api.HealthHandler())

	// Apply middleware
	handler := middleware.LoggingMiddleware(middleware.CORSMiddleware(mux))

	// Create HTTP server
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("[SERVER] Starting HTTP server on %s", addr)
		errChan <- server.ListenAndServe()
	}()

	// Wait for either server error or shutdown signal
	select {
	case err := <-errChan:
		if err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("[SIGNAL] Received signal: %v", sig)
		log.Println("[SHUTDOWN] Initiating graceful shutdown...")

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("[ERROR] Shutdown error: %v", err)
		}

		log.Println("[SHUTDOWN] Server stopped gracefully")
	}
}
