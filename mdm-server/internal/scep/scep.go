package scep

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"mdm-server/internal/store"
)

// Handler handles SCEP protocol requests
type Handler struct {
	tenantStore *store.TenantStore
	caCache     map[string]*CA // tenantID -> CA
}

// NewHandler creates a new SCEP handler
func NewHandler(tenantStore *store.TenantStore) *Handler {
	return &Handler{
		tenantStore: tenantStore,
		caCache:     make(map[string]*CA),
	}
}

// ServeHTTP handles SCEP requests
// SCEP endpoint format: /scep/{tenantID}?operation=...
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid SCEP URL", http.StatusBadRequest)
		return
	}
	tenantID := parts[2]

	operation := r.URL.Query().Get("operation")
	if operation == "" {
		http.Error(w, "Missing operation parameter", http.StatusBadRequest)
		return
	}

	log.Printf("SCEP request: tenant=%s, operation=%s, method=%s", tenantID, operation, r.Method)

	// Get or load tenant CA
	ca, err := h.getCA(tenantID)
	if err != nil {
		log.Printf("SCEP error: failed to get CA for tenant %s: %v", tenantID, err)
		http.Error(w, "Tenant CA not configured", http.StatusInternalServerError)
		return
	}

	switch operation {
	case "GetCACert":
		h.handleGetCACert(w, r, ca)
	case "GetCACaps":
		h.handleGetCACaps(w, r)
	case "PKIOperation":
		h.handlePKIOperation(w, r, ca, tenantID)
	default:
		http.Error(w, "Unknown operation", http.StatusBadRequest)
	}
}

// getCA retrieves or loads the CA for a tenant
func (h *Handler) getCA(tenantID string) (*CA, error) {
	// Check cache
	if ca, ok := h.caCache[tenantID]; ok {
		return ca, nil
	}

	// Load from database
	tenant, err := h.tenantStore.GetByID(tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// If tenant has CA configured, load it
	if tenant.CACertPEM != "" && tenant.CAKeyPEM != "" {
		ca, err := LoadCA(tenant.CACertPEM, tenant.CAKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to load tenant CA: %w", err)
		}
		h.caCache[tenantID] = ca
		return ca, nil
	}

	// Generate new CA for tenant
	log.Printf("Generating new CA for tenant %s", tenantID)
	ca, err := NewCA(tenant.Name, 10) // 10 year validity
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	// Save CA to tenant
	if err := h.tenantStore.UpdateCA(tenantID, ca.CertPEM, ca.KeyPEM); err != nil {
		return nil, fmt.Errorf("failed to save CA: %w", err)
	}

	h.caCache[tenantID] = ca
	return ca, nil
}

// handleGetCACert returns the CA certificate
func (h *Handler) handleGetCACert(w http.ResponseWriter, r *http.Request, ca *CA) {
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Write(ca.Certificate.Raw)
}

// handleGetCACaps returns SCEP capabilities
func (h *Handler) handleGetCACaps(w http.ResponseWriter, r *http.Request) {
	caps := []string{
		"POSTPKIOperation",
		"SHA-256",
		"SHA-512",
		"AES",
		"SCEPStandard",
		"Renewal",
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(caps, "\n")))
}

// handlePKIOperation handles certificate signing requests
func (h *Handler) handlePKIOperation(w http.ResponseWriter, r *http.Request, ca *CA, tenantID string) {
	var message []byte
	var err error

	if r.Method == http.MethodGet {
		// GET: Base64-encoded message in 'message' parameter
		messageB64 := r.URL.Query().Get("message")
		if messageB64 == "" {
			http.Error(w, "Missing message parameter", http.StatusBadRequest)
			return
		}
		message, err = base64.StdEncoding.DecodeString(messageB64)
		if err != nil {
			http.Error(w, "Invalid base64 message", http.StatusBadRequest)
			return
		}
	} else if r.Method == http.MethodPost {
		// POST: Binary message in body
		message, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Process SCEP PKI message
	// This is a simplified implementation - a full implementation would use the scep library
	// to properly parse PKCS#7 SignedData and EnvelopedData

	// For now, we'll handle the common case where the message contains a CSR
	// In production, you'd use github.com/smallstep/scep/scep package

	log.Printf("SCEP PKIOperation: received %d bytes from tenant %s", len(message), tenantID)

	// Try to parse as a simple CSR (for testing/development)
	// Production would use full PKCS#7 parsing
	csr, err := h.extractCSRFromMessage(message)
	if err != nil {
		log.Printf("SCEP: failed to extract CSR: %v", err)
		// For now, just return the CA cert as a response
		// This allows devices to at least get the CA
		w.Header().Set("Content-Type", "application/x-pki-message")
		w.Write(ca.Certificate.Raw)
		return
	}

	// Issue certificate
	cert, certPEM, err := ca.IssueCertificate(csr, 365) // 1 year validity
	if err != nil {
		log.Printf("SCEP: failed to issue certificate: %v", err)
		http.Error(w, "Failed to issue certificate", http.StatusInternalServerError)
		return
	}

	log.Printf("SCEP: issued certificate for %s (serial: %s)", csr.Subject.CommonName, cert.SerialNumber)
	_ = certPEM // Would be stored in database

	// Return signed certificate
	// In production, this would be wrapped in PKCS#7 SignedData
	w.Header().Set("Content-Type", "application/x-pki-message")
	w.Write(cert.Raw)
}

// extractCSRFromMessage attempts to extract a CSR from a SCEP message
// This is simplified - production would use full PKCS#7 parsing
func (h *Handler) extractCSRFromMessage(message []byte) (*x509.CertificateRequest, error) {
	// Try to parse directly as CSR (for testing)
	csr, err := x509.ParseCertificateRequest(message)
	if err == nil {
		return csr, nil
	}

	// SCEP messages are typically PKCS#7 wrapped
	// A full implementation would:
	// 1. Parse outer PKCS#7 SignedData
	// 2. Verify signature
	// 3. Decrypt inner PKCS#7 EnvelopedData
	// 4. Extract CSR from decrypted content

	return nil, fmt.Errorf("could not extract CSR from SCEP message: %w", err)
}

// InvalidateCache removes a tenant's CA from the cache
func (h *Handler) InvalidateCache(tenantID string) {
	delete(h.caCache, tenantID)
}
