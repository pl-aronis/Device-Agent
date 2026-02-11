package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mdm-server/internal/api"
	"mdm-server/internal/apns"
	"mdm-server/internal/config"
	"mdm-server/internal/scep"
	"mdm-server/internal/store"
	"mdm-server/internal/web"
)

func main() {
	// Command line flags
	configFile := flag.String("config", "", "Path to config file (optional)")
	initDB := flag.Bool("init", false, "Initialize database and exit")
	flag.Parse()

	log.Println("Starting MDM Server...")

	// Load configuration
	cfg := config.LoadFromEnv()
	if *configFile != "" {
		// TODO: Load additional config from file
		log.Printf("Config file specified: %s", *configFile)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize database
	log.Printf("Opening database: %s", cfg.DatabasePath)
	db, err := store.NewSQLiteDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if *initDB {
		log.Println("Database initialized successfully")
		return
	}

	// Initialize stores
	tenantStore := store.NewTenantStore(db)
	deviceStore := store.NewDeviceStore(db)
	commandStore := store.NewCommandStore(db)

	// Ensure default tenant exists
	ensureDefaultTenant(tenantStore)

	// Initialize APNs client pool
	apnsPool := apns.NewClientPool(func(tenantID string) ([]byte, []byte, string, error) {
		tenant, err := tenantStore.GetByID(tenantID)
		if err != nil || tenant == nil {
			return nil, nil, "", fmt.Errorf("tenant not found")
		}
		return tenant.APNsCertData, tenant.APNsKeyData, tenant.APNsTopic, nil
	})

	// Initialize SCEP handler
	scepHandler := scep.NewHandler(tenantStore)

	// Initialize API handlers
	checkinHandler := api.NewCheckinHandler(deviceStore, commandStore, tenantStore)
	connectHandler := api.NewConnectHandler(commandStore, deviceStore)
	adminHandler := api.NewAdminHandler(deviceStore, commandStore, tenantStore, apnsPool)

	// Initialize web handler
	webHandler := web.NewHandler(web.Config{
		TenantStore:  tenantStore,
		DeviceStore:  deviceStore,
		CommandStore: commandStore,
		SCEPHandler:  scepHandler,
		ServerURL:    cfg.ServerURL,
		JWTSecret:    cfg.JWTSecret,
	})

	// Set up HTTP routes
	mux := http.NewServeMux()

	// MDM Protocol endpoints
	mux.Handle("/mdm/checkin", checkinHandler)
	mux.Handle("/mdm/connect", connectHandler)

	// SCEP endpoint
	mux.Handle("/scep/", scepHandler)

	// Admin API (device endpoints)
	mux.HandleFunc("/api/devices", adminHandler.ListDevices)
	mux.HandleFunc("/api/devices/", adminHandler.DeviceAction)

	// Web UI (includes /api/tenants routes and admin dashboard)
	webHandler.RegisterRoutes(mux)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Create server
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: logMiddleware(mux),
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Close()
	}()

	// Start server
	log.Printf("MDM Server listening on %s", cfg.ListenAddr)
	log.Printf("Server URL: %s", cfg.ServerURL)
	log.Printf("Admin Dashboard: %s/admin/", cfg.ServerURL)

	if cfg.IsTLSEnabled() {
		log.Printf("TLS enabled with cert: %s", cfg.TLSCertFile)
		if err := server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Println("WARNING: TLS not enabled. Use HTTPS in production!")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}

	log.Println("Server stopped")
}

// ensureDefaultTenant creates a default tenant if none exist
func ensureDefaultTenant(ts *store.TenantStore) {
	tenants, err := ts.List()
	if err != nil {
		log.Printf("Warning: failed to list tenants: %v", err)
		return
	}

	if len(tenants) == 0 {
		log.Println("Creating default tenant...")
		tenant, err := ts.Create("Default Organization", "default")
		if err != nil {
			log.Printf("Warning: failed to create default tenant: %v", err)
			return
		}
		log.Printf("Default tenant created: %s", tenant.ID)
	}
}

// logMiddleware logs all HTTP requests
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
