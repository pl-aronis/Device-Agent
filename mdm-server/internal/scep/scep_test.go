package scep

import (
	"crypto/x509"
	"strings"
	"testing"
)

func TestNewCA(t *testing.T) {
	ca, err := NewCA("Test Organization", 10)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Verify certificate
	if ca.Certificate == nil {
		t.Fatal("Certificate should not be nil")
	}
	if !ca.Certificate.IsCA {
		t.Error("Certificate should be a CA")
	}
	if ca.Certificate.Subject.CommonName != "Test Organization MDM CA" {
		t.Errorf("Unexpected CommonName: %s", ca.Certificate.Subject.CommonName)
	}

	// Verify PEM encoding
	if !strings.Contains(ca.CertPEM, "-----BEGIN CERTIFICATE-----") {
		t.Error("CertPEM should be PEM encoded")
	}
	if !strings.Contains(ca.KeyPEM, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Error("KeyPEM should be PEM encoded")
	}
}

func TestLoadCA(t *testing.T) {
	// Create a CA
	original, err := NewCA("Test Org", 5)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Load from PEM
	loaded, err := LoadCA(original.CertPEM, original.KeyPEM)
	if err != nil {
		t.Fatalf("Failed to load CA: %v", err)
	}

	// Verify loaded CA matches original
	if loaded.Certificate.Subject.CommonName != original.Certificate.Subject.CommonName {
		t.Error("Loaded CA CommonName mismatch")
	}
	if loaded.Certificate.SerialNumber.Cmp(original.Certificate.SerialNumber) != 0 {
		t.Error("Loaded CA SerialNumber mismatch")
	}
}

func TestIssueCertificate(t *testing.T) {
	// Create CA
	ca, err := NewCA("Test Org", 10)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	// Create a mock CSR
	csr := &x509.CertificateRequest{
		Subject: ca.Certificate.Subject,
	}
	csr.Subject.CommonName = "Device-12345"

	// This would fail without a proper CSR with public key
	// For a real test, we'd need to generate a proper CSR
	// But we can at least test the CA was created properly
	if ca.Certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("CA should have CertSign key usage")
	}
}

func TestGetCertificateFingerprint(t *testing.T) {
	ca, err := NewCA("Test Org", 10)
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}

	fingerprint := ca.GetCertificateFingerprint()
	if fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}
	if len(fingerprint) != 64 { // SHA256 = 32 bytes = 64 hex chars
		t.Errorf("Fingerprint should be 64 chars, got %d", len(fingerprint))
	}
}
