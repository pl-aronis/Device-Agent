package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the MDM server configuration
type Config struct {
	// Server settings
	ServerURL  string // Public HTTPS URL (e.g., https://mdm.example.com)
	ListenAddr string // Address to listen on (e.g., :8080)

	// Database
	DatabasePath string // SQLite file path

	// TLS certificates for HTTPS
	TLSCertFile string
	TLSKeyFile  string

	// SCEP CA (optional - can be per-tenant)
	CAKeyFile  string
	CACertFile string

	// Admin settings
	AdminEmail    string
	AdminPassword string // Initial admin password
	JWTSecret     string

	// APNs settings (optional - can be per-tenant)
	APNsCertFile string
	APNsKeyFile  string
	APNsTopic    string

	// Feature flags
	EnableDEP  bool
	EnableSCEP bool
	EnableMTLS bool
	DebugMode  bool
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := &Config{
		// Defaults
		ListenAddr:   getEnv("MDM_LISTEN_ADDR", ":8080"),
		DatabasePath: getEnv("MDM_DATABASE_PATH", "mdm.db"),
		ServerURL:    getEnv("MDM_SERVER_URL", "http://localhost:8080"),
		JWTSecret:    getEnv("MDM_JWT_SECRET", "change-me-in-production"),

		// TLS
		TLSCertFile: getEnv("MDM_TLS_CERT", ""),
		TLSKeyFile:  getEnv("MDM_TLS_KEY", ""),

		// CA
		CAKeyFile:  getEnv("MDM_CA_KEY", ""),
		CACertFile: getEnv("MDM_CA_CERT", ""),

		// Admin
		AdminEmail:    getEnv("MDM_ADMIN_EMAIL", "admin@localhost"),
		AdminPassword: getEnv("MDM_ADMIN_PASSWORD", ""),

		// APNs
		APNsCertFile: getEnv("MDM_APNS_CERT", ""),
		APNsKeyFile:  getEnv("MDM_APNS_KEY", ""),
		APNsTopic:    getEnv("MDM_APNS_TOPIC", ""),

		// Features
		EnableDEP:  getEnvBool("MDM_ENABLE_DEP", true),
		EnableSCEP: getEnvBool("MDM_ENABLE_SCEP", true),
		EnableMTLS: getEnvBool("MDM_ENABLE_MTLS", false),
		DebugMode:  getEnvBool("MDM_DEBUG", false),
	}

	return cfg
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("MDM_SERVER_URL is required")
	}

	if c.DatabasePath == "" {
		return fmt.Errorf("MDM_DATABASE_PATH is required")
	}

	if c.JWTSecret == "change-me-in-production" {
		fmt.Println("WARNING: Using default JWT secret. Set MDM_JWT_SECRET for production!")
	}

	return nil
}

// IsTLSEnabled returns true if TLS certificates are configured
func (c *Config) IsTLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// HasAPNs returns true if APNs is configured at server level
func (c *Config) HasAPNs() bool {
	return c.APNsCertFile != "" && c.APNsTopic != ""
}

// HasCA returns true if CA is configured at server level
func (c *Config) HasCA() bool {
	return c.CAKeyFile != "" && c.CACertFile != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}
