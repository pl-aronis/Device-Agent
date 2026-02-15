package scep

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"mdm-server/internal/store"

	pkcs7 "go.mozilla.org/pkcs7"
)

var (
	oidTransactionID  = asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, 7}
	oidMessageType    = asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, 2}
	oidPKIStatus      = asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, 3}
	oidFailInfo       = asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, 4}
	oidSenderNonce    = asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, 5}
	oidRecipientNonce = asn1.ObjectIdentifier{2, 16, 840, 1, 113733, 1, 9, 6}
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

	log.Printf("SCEP request: tenant=%s, operation=%s, method=%s, url=%s, remote=%s",
		tenantID, operation, r.Method, r.URL.String(), r.RemoteAddr)

	// Get or load tenant CA
	ca, err := h.getCA(tenantID)
	if err != nil {
		log.Printf("SCEP error: failed to get CA for tenant %s: %v", tenantID, err)
		http.Error(w, fmt.Sprintf("Tenant CA not configured: %v", err), http.StatusInternalServerError)
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
		return nil, fmt.Errorf("database error: %w", err)
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

// handleGetCACert returns the CA certificate(s)
func (h *Handler) handleGetCACert(w http.ResponseWriter, r *http.Request, ca *CA) {
	// Return the CA cert as a degenerate PKCS#7 (Application/x-x509-ca-ra-cert)
	// This allows including the RA certificate if we had one, and is often required by SCEP clients
	// even if there is only one root CA.

	degenerate, err := pkcs7.DegenerateCertificate(ca.Certificate.Raw)
	if err != nil {
		log.Printf("SCEP: failed to create degenerate cert for GetCACert: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("SCEP GetCACert: returning CA certificate (size=%d bytes, subject=%s)",
		len(degenerate), ca.Certificate.Subject.CommonName)

	w.Header().Set("Content-Type", "application/x-x509-ca-ra-cert")
	w.Write(degenerate)
}

// handleGetCACaps returns SCEP capabilities
func (h *Handler) handleGetCACaps(w http.ResponseWriter, r *http.Request) {
	caps := []string{
		"POSTPKIOperation",
		"SHA-256",
		"AES",
		"SCEPStandard",
	}

	log.Printf("SCEP GetCACaps: returning capabilities: %v", caps)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(caps, "\n")))
}

// handlePKIOperation handles certificate signing requests via PKCS#7
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

	log.Printf("SCEP PKIOperation: received %d bytes from tenant %s", len(message), tenantID)

	// Parse the outer PKCS#7 SignedData envelope
	p7, err := pkcs7.Parse(message)
	if err != nil {
		log.Printf("SCEP: failed to parse PKCS#7 SignedData: %v", err)
		// Try as raw CSR (fallback for simple clients)
		h.handleRawCSR(w, message, ca, tenantID)
		return
	}

	// The content inside the SignedData is an EnvelopedData (encrypted for our CA)
	envelopedData := p7.Content

	// Parse the inner PKCS#7 EnvelopedData
	p7env, err := pkcs7.Parse(envelopedData)
	if err != nil {
		log.Printf("SCEP: inner content is not EnvelopedData (%v), trying as CSR", err)
		// Maybe the content is already a CSR
		h.handleRawCSR(w, envelopedData, ca, tenantID)
		return
	}

	// Decrypt the EnvelopedData using our CA certificate and key
	decryptedContent, err := p7env.Decrypt(ca.Certificate, ca.PrivateKey)
	if err != nil {
		log.Printf("SCEP: failed to decrypt EnvelopedData: %v", err)
		h.sendSCEPFailure(w, ca, p7)
		return
	}

	log.Printf("SCEP: decrypted %d bytes of CSR data", len(decryptedContent))

	// Parse the CSR from decrypted content
	csr, err := x509.ParseCertificateRequest(decryptedContent)
	if err != nil {
		log.Printf("SCEP: failed to parse CSR from decrypted content: %v", err)
		h.sendSCEPFailure(w, ca, p7)
		return
	}

	// Validate CSR
	if err := csr.CheckSignature(); err != nil {
		log.Printf("SCEP: CSR signature invalid: %v", err)
		h.sendSCEPFailure(w, ca, p7)
		return
	}

	// Issue certificate
	cert, _, err := ca.IssueCertificate(csr, 365) // 1 year validity
	if err != nil {
		log.Printf("SCEP: failed to issue certificate: %v", err)
		h.sendSCEPFailure(w, ca, p7)
		return
	}

	log.Printf("SCEP: issued certificate for %s (serial: %s)", csr.Subject.CommonName, cert.SerialNumber)

	// Build SCEP success response: SignedData containing the issued cert
	h.sendSCEPSuccess(w, ca, cert, p7)
}

// handleRawCSR handles the case where the message is a raw CSR (not PKCS#7 wrapped)
func (h *Handler) handleRawCSR(w http.ResponseWriter, message []byte, ca *CA, tenantID string) {
	csr, err := x509.ParseCertificateRequest(message)
	if err != nil {
		log.Printf("SCEP: not a valid CSR either: %v", err)
		http.Error(w, "Invalid SCEP message", http.StatusBadRequest)
		return
	}

	cert, _, err := ca.IssueCertificate(csr, 365)
	if err != nil {
		log.Printf("SCEP: failed to issue certificate: %v", err)
		http.Error(w, "Failed to issue certificate", http.StatusInternalServerError)
		return
	}

	log.Printf("SCEP: issued certificate via raw CSR for %s (serial: %s)", csr.Subject.CommonName, cert.SerialNumber)

	// Return as degenerate PKCS#7
	degenerateCerts, err := pkcs7.DegenerateCertificate(cert.Raw)
	if err != nil {
		log.Printf("SCEP: failed to create degenerate cert: %v", err)
		w.Header().Set("Content-Type", "application/x-pki-message")
		w.Write(cert.Raw)
		return
	}

	w.Header().Set("Content-Type", "application/x-pki-message")
	w.Write(degenerateCerts)
}

// sendSCEPSuccess builds a CertRep SUCCESS response following the SCEP protocol spec.
// Reference: smallstep/scep Success() method
func (h *Handler) sendSCEPSuccess(w http.ResponseWriter, ca *CA, issuedCert *x509.Certificate, requestP7 *pkcs7.PKCS7) {
	// Step 1: Create a degenerate PKCS#7 containing the issued certificate
	degenerateCerts, err := pkcs7.DegenerateCertificate(issuedCert.Raw)
	if err != nil {
		log.Printf("SCEP: failed to create degenerate certificate: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("SCEP SUCCESS: degenerate cert created (%d bytes)", len(degenerateCerts))

	// Step 2: Encrypt the degenerate cert for the requesting client
	// The client's certificate is in the PKCS#7 SignedData from the request.
	// This is CRITICAL - macOS expects the CertRep content to be encrypted.
	encryptedContent, err := pkcs7.Encrypt(degenerateCerts, requestP7.Certificates)
	if err != nil {
		log.Printf("SCEP: failed to encrypt cert for recipient: %v (num recipient certs: %d)", err, len(requestP7.Certificates))
		// Fallback: try without encryption (some clients may accept this)
		encryptedContent = degenerateCerts
	} else {
		log.Printf("SCEP SUCCESS: encrypted cert for %d recipient(s)", len(requestP7.Certificates))
	}

	// Step 3: Extract attributes from the request
	transactionID, senderNonce := getSCEPAttributes(requestP7)
	log.Printf("SCEP SUCCESS: transactionID=%s, senderNonce=%x", transactionID, senderNonce)

	// Step 4: Generate a new Sender Nonce for the response
	respSenderNonce := make([]byte, 16)
	rand.Read(respSenderNonce)

	// Step 5: Prepare signed attributes with STRING types (SCEP spec requirement)
	// MessageType and PKIStatus MUST be PrintableString, NOT integers
	attrs := []pkcs7.Attribute{
		{Type: oidTransactionID, Value: transactionID},
		{Type: oidMessageType, Value: "3"}, // CertRep as string
		{Type: oidPKIStatus, Value: "0"},   // SUCCESS as string
		{Type: oidSenderNonce, Value: respSenderNonce},
		{Type: oidRecipientNonce, Value: senderNonce},
	}

	// Step 6: Create SignedData wrapping the encrypted content
	signedData, err := pkcs7.NewSignedData(encryptedContent)
	if err != nil {
		log.Printf("SCEP: failed to create signed data: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Step 7: Add the issued certificate to the signed data
	// The recipient expects this as the first certificate in the array
	signedData.AddCertificate(issuedCert)

	// Step 8: Sign with CA certificate and include SCEP attributes
	signerConfig := pkcs7.SignerInfoConfig{
		ExtraSignedAttributes: attrs,
	}
	if err := signedData.AddSigner(ca.Certificate, ca.PrivateKey, signerConfig); err != nil {
		log.Printf("SCEP: failed to add signer: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Step 9: Finalize
	signedResponse, err := signedData.Finish()
	if err != nil {
		log.Printf("SCEP: failed to finish signing: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	log.Printf("SCEP SUCCESS: sending CertRep response (%d bytes)", len(signedResponse))

	w.Header().Set("Content-Type", "application/x-pki-message")
	w.Write(signedResponse)
}

// sendSCEPFailure sends a SCEP failure response
func (h *Handler) sendSCEPFailure(w http.ResponseWriter, ca *CA, requestP7 *pkcs7.PKCS7) {
	// Proper failure response should be signed and include failure status
	// For now we just return 500, but in future we should replicate sendSCEPSuccess
	// structure with pkiStatus=FAILURE (2)
	http.Error(w, "SCEP enrollment failed", http.StatusInternalServerError)
}

// signSCEPResponse signs data with the CA certificate
func signSCEPResponse(content []byte, signerCert *x509.Certificate, signerKey crypto.PrivateKey, attrs []pkcs7.Attribute) ([]byte, error) {
	toBeSigned, err := pkcs7.NewSignedData(content)
	if err != nil {
		return nil, fmt.Errorf("create signed data: %w", err)
	}

	config := pkcs7.SignerInfoConfig{
		ExtraSignedAttributes: attrs,
	}

	if err := toBeSigned.AddSigner(signerCert, signerKey, config); err != nil {
		return nil, fmt.Errorf("add signer: %w", err)
	}

	signed, err := toBeSigned.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish signing: %w", err)
	}

	return signed, nil
}

// getSCEPAttributes extracts transactionID and senderNonce from a PKCS#7 envelope
func getSCEPAttributes(p7 *pkcs7.PKCS7) (transactionID string, senderNonce []byte) {
	// Extract Transaction ID
	if err := p7.UnmarshalSignedAttribute(oidTransactionID, &transactionID); err != nil {
		// Try extracting as bytes in case it was encoded differently
		var paramBytes []byte
		if err := p7.UnmarshalSignedAttribute(oidTransactionID, &paramBytes); err == nil {
			transactionID = string(paramBytes)
		}
	}

	// Extract Sender Nonce
	if err := p7.UnmarshalSignedAttribute(oidSenderNonce, &senderNonce); err != nil {
		// Log error or ignore? SCEP requires senderNonce.
		log.Printf("SCEP: failed to extract senderNonce: %v", err)
	}

	return
}

// InvalidateCache removes a tenant's CA from the cache
func (h *Handler) InvalidateCache(tenantID string) {
	delete(h.caCache, tenantID)
}

// issueSelfSignedCert creates a simple self-signed certificate (for testing when CSR parsing fails)
func issueSelfSignedCert(ca *CA) (*x509.Certificate, error) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Certificate, ca.Certificate.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(certDER)
}
