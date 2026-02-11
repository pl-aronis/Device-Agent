package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	pkcs7 "go.mozilla.org/pkcs7"
)

const mdmcertURL = "https://mdmcert.download/api/v1/signrequest"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "generate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: apnstool generate <email>")
			fmt.Println("  <email>  The email you registered with at mdmcert.download")
			os.Exit(1)
		}
		email := os.Args[2]
		generateCerts(email)

	case "decrypt":
		if len(os.Args) < 3 {
			fmt.Println("Usage: apnstool decrypt <encrypted_file>")
			fmt.Println("  <encrypted_file>  The .p7 file from mdmcert.download email")
			os.Exit(1)
		}
		decryptResponse(os.Args[2])

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`
MDM APNs Certificate Tool
=========================

This tool helps you get an APNs push certificate from mdmcert.download.

STEP 1: Generate certificates and submit to mdmcert.download
  apnstool generate <your-email@example.com>

  This will:
  - Generate an encryption certificate (encrypt_cert.pem + encrypt_key.pem)
  - Generate a push CSR (push_csr.pem + push_key.pem)
  - Submit both to mdmcert.download API
  
STEP 2: Check your email for the signed CSR (may take a few minutes)
  You'll receive an email with an encrypted attachment.
  Save the attachment as "mdmcert.p7" in this directory.

STEP 3: Decrypt the signed CSR
  apnstool decrypt mdmcert.p7

  This creates "signed_csr.pem" - upload this to Apple.

STEP 4: Upload to Apple
  Go to: https://identity.apple.com/pushcert
  - Sign in with your Apple ID
  - Click "Create a Certificate"
  - Upload signed_csr.pem
  - Download the certificate (MDM_.pem)

STEP 5: Upload to your MDM server
  - Go to your MDM admin dashboard
  - Select your tenant → Upload APNs Certificate
  - Upload the MDM_.pem file from Apple
  - Enter the APNs topic from the certificate
`)
}

func generateCerts(email string) {
	outDir := "apns_certs"
	os.MkdirAll(outDir, 0755)

	fmt.Println("=== MDM APNs Certificate Generator ===")
	fmt.Println()

	// Step 1: Generate encryption key pair and certificate
	fmt.Println("[1/3] Generating encryption certificate...")
	encryptKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("ERROR: Failed to generate encryption key: %v\n", err)
		os.Exit(1)
	}

	encryptCert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "MDM Encryption Certificate",
			Organization: []string{"MDM Server"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		BasicConstraintsValid: true,
	}

	encryptCertDER, err := x509.CreateCertificate(rand.Reader, encryptCert, encryptCert, &encryptKey.PublicKey, encryptKey)
	if err != nil {
		fmt.Printf("ERROR: Failed to create encryption certificate: %v\n", err)
		os.Exit(1)
	}

	// Save encryption cert
	encryptCertPath := filepath.Join(outDir, "encrypt_cert.pem")
	saveToFile(encryptCertPath, "CERTIFICATE", encryptCertDER)

	// Save encryption key
	encryptKeyPath := filepath.Join(outDir, "encrypt_key.pem")
	saveToFile(encryptKeyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(encryptKey))

	fmt.Printf("  ✅ Encryption cert: %s\n", encryptCertPath)
	fmt.Printf("  ✅ Encryption key:  %s\n", encryptKeyPath)

	// Step 2: Generate push CSR
	fmt.Println("[2/3] Generating push certificate CSR...")
	pushKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("ERROR: Failed to generate push key: %v\n", err)
		os.Exit(1)
	}

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "MDM Push Certificate",
			Organization: []string{"MDM Server"},
			Country:      []string{"US"},
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, pushKey)
	if err != nil {
		fmt.Printf("ERROR: Failed to create CSR: %v\n", err)
		os.Exit(1)
	}

	// Save push CSR
	csrPath := filepath.Join(outDir, "push_csr.pem")
	saveToFile(csrPath, "CERTIFICATE REQUEST", csrDER)

	// Save push key (you'll need this later!)
	pushKeyPath := filepath.Join(outDir, "push_key.pem")
	saveToFile(pushKeyPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(pushKey))

	fmt.Printf("  ✅ Push CSR: %s\n", csrPath)
	fmt.Printf("  ✅ Push key: %s (⚠️  KEEP THIS SAFE!)\n", pushKeyPath)

	// Step 3: Submit to mdmcert.download
	fmt.Println("[3/3] Submitting to mdmcert.download...")
	err = submitToMdmcert(email,
		filepath.Join(outDir, "push_csr.pem"),
		filepath.Join(outDir, "encrypt_cert.pem"),
	)
	if err != nil {
		fmt.Printf("  ⚠️  Auto-submit failed: %v\n", err)
		fmt.Println()
		fmt.Println("  You can submit manually instead:")
		fmt.Printf("  1. Go to https://mdmcert.download\n")
		fmt.Printf("  2. Upload %s as the CSR\n", csrPath)
		fmt.Printf("  3. Upload %s as the encryption certificate\n", encryptCertPath)
		fmt.Printf("  4. Use email: %s\n", email)
	} else {
		fmt.Println("  ✅ Submitted successfully!")
	}

	fmt.Println()
	fmt.Println("=== NEXT STEPS ===")
	fmt.Println("1. Check your email for the signed CSR from mdmcert.download")
	fmt.Println("2. Save the email attachment as 'mdmcert.p7' in this folder")
	fmt.Println("3. Run: apnstool decrypt apns_certs/mdmcert.p7")
}

func submitToMdmcert(email, csrPath, encryptCertPath string) error {
	csrPEM, err := os.ReadFile(csrPath)
	if err != nil {
		return fmt.Errorf("read CSR: %w", err)
	}

	encryptPEM, err := os.ReadFile(encryptCertPath)
	if err != nil {
		return fmt.Errorf("read encrypt cert: %w", err)
	}

	// mdmcert.download expects base64-encoded PEM data (not DER) + an API key
	// Reference: https://github.com/micromdm/micromdm/blob/main/cmd/mdmctl/mdmcert.download.go
	payload := struct {
		CSR     string `json:"csr"`
		Email   string `json:"email"`
		Key     string `json:"key"`
		Encrypt string `json:"encrypt"`
	}{
		CSR:     base64.StdEncoding.EncodeToString(csrPEM),
		Email:   email,
		Key:     "f847aea2ba06b41264d587b229e2712c89b1490a1208b7ff1aafab5bb40d47bc",
		Encrypt: base64.StdEncoding.EncodeToString(encryptPEM),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	req, err := http.NewRequest("POST", mdmcertURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "micromdm/certhelper")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	fmt.Printf("  Response: %s\n", string(respBody))
	return nil
}

func decryptResponse(encryptedFile string) {
	outDir := "apns_certs"

	fmt.Println("=== Decrypting mdmcert.download Response ===")
	fmt.Println()

	// Read encrypted file
	rawData, err := os.ReadFile(encryptedFile)
	if err != nil {
		fmt.Printf("ERROR: Cannot read %s: %v\n", encryptedFile, err)
		os.Exit(1)
	}

	// mdmcert.download sends the file as hex-encoded text
	// Try hex decoding first (MicroMDM format), then base64, then raw binary
	var pkcsBytes []byte
	trimmed := bytes.TrimSpace(rawData)

	// Try hex decode
	hexDecoded, err := hex.DecodeString(string(trimmed))
	if err == nil {
		fmt.Println("  Detected hex-encoded file")
		pkcsBytes = hexDecoded
	} else {
		// Try base64 decode
		b64Decoded, err := base64.StdEncoding.DecodeString(string(trimmed))
		if err == nil {
			fmt.Println("  Detected base64-encoded file")
			pkcsBytes = b64Decoded
		} else {
			// Try raw binary
			fmt.Println("  Treating as raw binary file")
			pkcsBytes = rawData
		}
	}

	// Read encryption key
	keyPath := filepath.Join(outDir, "encrypt_key.pem")
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("ERROR: Cannot read encryption key %s: %v\n", keyPath, err)
		fmt.Println("Make sure you're in the same directory where you ran 'generate'")
		os.Exit(1)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		fmt.Println("ERROR: Failed to decode encryption key PEM")
		os.Exit(1)
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse encryption key: %v\n", err)
		os.Exit(1)
	}

	// Read encryption cert
	certPath := filepath.Join(outDir, "encrypt_cert.pem")
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		fmt.Printf("ERROR: Cannot read encryption cert %s: %v\n", certPath, err)
		os.Exit(1)
	}

	certBlock, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse encryption cert: %v\n", err)
		os.Exit(1)
	}

	// Parse PKCS7 envelope
	p7, err := pkcs7.Parse(pkcsBytes)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse PKCS7 data: %v\n", err)
		fmt.Println("The file may be corrupted or in an unexpected format.")
		fmt.Printf("File size: %d bytes, first bytes: %x\n", len(rawData), rawData[:min(32, len(rawData))])
		os.Exit(1)
	}

	// Decrypt
	decrypted, err := p7.Decrypt(cert, key)
	if err != nil {
		fmt.Printf("ERROR: Failed to decrypt: %v\n", err)
		os.Exit(1)
	}

	// Save decrypted signed CSR
	outputPath := filepath.Join(outDir, "signed_csr.pem")
	if err := os.WriteFile(outputPath, decrypted, 0644); err != nil {
		fmt.Printf("ERROR: Failed to save: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Decrypted signed CSR saved to: %s\n", outputPath)
	fmt.Println()
	fmt.Println("=== NEXT STEPS ===")
	fmt.Println("1. Go to: https://identity.apple.com/pushcert")
	fmt.Println("2. Sign in with your Apple ID")
	fmt.Println("3. Click 'Create a Certificate'")
	fmt.Printf("4. Upload: %s\n", outputPath)
	fmt.Println("5. Download the resulting MDM_.pem certificate")
	fmt.Println("6. Upload that .pem to your MDM admin dashboard")
	fmt.Printf("   (Also upload %s as the private key)\n", filepath.Join(outDir, "push_key.pem"))
}

func saveToFile(path, blockType string, data []byte) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("ERROR: Cannot create %s: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()

	pem.Encode(f, &pem.Block{
		Type:  blockType,
		Bytes: data,
	})
}
