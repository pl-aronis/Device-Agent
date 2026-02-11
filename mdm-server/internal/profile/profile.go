package profile

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"text/template"

	"github.com/google/uuid"
	"howett.net/plist"
)

// EnrollmentConfig holds the configuration for generating enrollment profiles
type EnrollmentConfig struct {
	TenantID     string
	TenantName   string
	ServerURL    string // Base URL of the MDM server (e.g., https://mdm.example.com)
	APNsTopic    string // Push notification topic
	CACertBase64 string // Base64-encoded CA certificate
	Challenge    string // Optional SCEP challenge password
}

// EnrollmentProfile represents a .mobileconfig enrollment profile
type EnrollmentProfile struct {
	PayloadContent      []interface{} `plist:"PayloadContent"`
	PayloadDisplayName  string        `plist:"PayloadDisplayName"`
	PayloadIdentifier   string        `plist:"PayloadIdentifier"`
	PayloadOrganization string        `plist:"PayloadOrganization"`
	PayloadType         string        `plist:"PayloadType"`
	PayloadUUID         string        `plist:"PayloadUUID"`
	PayloadVersion      int           `plist:"PayloadVersion"`
	PayloadDescription  string        `plist:"PayloadDescription,omitempty"`
}

// MDMPayload represents the MDM configuration payload
type MDMPayload struct {
	PayloadType             string `plist:"PayloadType"`
	PayloadIdentifier       string `plist:"PayloadIdentifier"`
	PayloadUUID             string `plist:"PayloadUUID"`
	PayloadVersion          int    `plist:"PayloadVersion"`
	PayloadDisplayName      string `plist:"PayloadDisplayName"`
	ServerURL               string `plist:"ServerURL"`
	CheckInURL              string `plist:"CheckInURL"`
	Topic                   string `plist:"Topic"`
	AccessRights            int    `plist:"AccessRights"`
	SignMessage             bool   `plist:"SignMessage"`
	CheckOutWhenRemoved     bool   `plist:"CheckOutWhenRemoved"`
	IdentityCertificateUUID string `plist:"IdentityCertificateUUID,omitempty"`
}

// SCEPPayload represents the SCEP configuration payload
type SCEPPayload struct {
	PayloadType        string             `plist:"PayloadType"`
	PayloadIdentifier  string             `plist:"PayloadIdentifier"`
	PayloadUUID        string             `plist:"PayloadUUID"`
	PayloadVersion     int                `plist:"PayloadVersion"`
	PayloadDisplayName string             `plist:"PayloadDisplayName"`
	PayloadContent     SCEPPayloadContent `plist:"PayloadContent"`
}

// SCEPPayloadContent contains SCEP configuration details
type SCEPPayloadContent struct {
	URL           string     `plist:"URL"`
	Name          string     `plist:"Name,omitempty"`
	Subject       [][]string `plist:"Subject,omitempty"`
	Challenge     string     `plist:"Challenge,omitempty"`
	KeySize       int        `plist:"Keysize"`
	KeyType       string     `plist:"Key Type"`
	KeyUsage      int        `plist:"Key Usage"` // 1=signing, 4=encryption, 5=both
	CAFingerprint []byte     `plist:"CAFingerprint,omitempty"`
}

// CACertPayload represents a CA certificate payload
type CACertPayload struct {
	PayloadType        string `plist:"PayloadType"`
	PayloadIdentifier  string `plist:"PayloadIdentifier"`
	PayloadUUID        string `plist:"PayloadUUID"`
	PayloadVersion     int    `plist:"PayloadVersion"`
	PayloadDisplayName string `plist:"PayloadDisplayName"`
	PayloadContent     []byte `plist:"PayloadContent"`
}

// Generator creates enrollment profiles
type Generator struct{}

// NewGenerator creates a new profile generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateEnrollmentProfile creates a complete enrollment profile
func (g *Generator) GenerateEnrollmentProfile(cfg EnrollmentConfig) ([]byte, error) {
	if cfg.APNsTopic == "" {
		return nil, fmt.Errorf("APNs topic is required for enrollment. Upload an APNs push certificate for this tenant first")
	}

	profileUUID := uuid.New().String()
	mdmUUID := uuid.New().String()
	scepUUID := uuid.New().String()
	caUUID := uuid.New().String()

	// Decode CA certificate
	var caCertData []byte
	if cfg.CACertBase64 != "" {
		var err error
		caCertData, err = base64.StdEncoding.DecodeString(cfg.CACertBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CA certificate: %w", err)
		}
	}

	// Build payloads
	payloads := []interface{}{}

	// 1. CA Certificate Payload (if provided)
	if len(caCertData) > 0 {
		caPayload := map[string]interface{}{
			"PayloadType":        "com.apple.security.root",
			"PayloadIdentifier":  fmt.Sprintf("com.%s.mdm.ca", cfg.TenantID),
			"PayloadUUID":        caUUID,
			"PayloadVersion":     1,
			"PayloadDisplayName": fmt.Sprintf("%s CA Certificate", cfg.TenantName),
			"PayloadContent":     caCertData,
		}
		payloads = append(payloads, caPayload)
	}

	// 2. SCEP Payload
	scepPayload := map[string]interface{}{
		"PayloadType":        "com.apple.security.scep",
		"PayloadIdentifier":  fmt.Sprintf("com.%s.mdm.scep", cfg.TenantID),
		"PayloadUUID":        scepUUID,
		"PayloadVersion":     1,
		"PayloadDisplayName": "MDM Identity Certificate",
		"PayloadContent": map[string]interface{}{
			"URL":       fmt.Sprintf("%s/scep/%s", cfg.ServerURL, cfg.TenantID),
			"Name":      "MDM-SCEP",
			"Subject":   [][][]string{{{"CN", "MDM Device Certificate"}}, {{"O", cfg.TenantName}}},
			"Keysize":   2048,
			"Key Type":  "RSA",
			"Key Usage": 5, // Both signing and encryption
		},
	}
	if cfg.Challenge != "" {
		scepPayload["PayloadContent"].(map[string]interface{})["Challenge"] = cfg.Challenge
	}
	payloads = append(payloads, scepPayload)

	// 3. MDM Payload
	mdmPayload := map[string]interface{}{
		"PayloadType":             "com.apple.mdm",
		"PayloadIdentifier":       fmt.Sprintf("com.%s.mdm.profile", cfg.TenantID),
		"PayloadUUID":             mdmUUID,
		"PayloadVersion":          1,
		"PayloadDisplayName":      "MDM Configuration",
		"ServerURL":               fmt.Sprintf("%s/mdm/connect", cfg.ServerURL),
		"CheckInURL":              fmt.Sprintf("%s/mdm/checkin", cfg.ServerURL),
		"Topic":                   cfg.APNsTopic,
		"AccessRights":            8191, // Full access
		"SignMessage":             true,
		"CheckOutWhenRemoved":     true,
		"IdentityCertificateUUID": scepUUID,
		"ServerCapabilities":      []string{"com.apple.mdm.per-user-connections"},
	}
	payloads = append(payloads, mdmPayload)

	// Build profile
	profile := map[string]interface{}{
		"PayloadContent":      payloads,
		"PayloadDisplayName":  fmt.Sprintf("%s MDM Enrollment", cfg.TenantName),
		"PayloadIdentifier":   fmt.Sprintf("com.%s.mdm", cfg.TenantID),
		"PayloadOrganization": cfg.TenantName,
		"PayloadType":         "Configuration",
		"PayloadUUID":         profileUUID,
		"PayloadVersion":      1,
		"PayloadDescription":  fmt.Sprintf("This profile enables device management by %s.", cfg.TenantName),
	}

	// Encode to plist
	var buf bytes.Buffer
	encoder := plist.NewEncoder(&buf)
	encoder.Indent("\t")
	if err := encoder.Encode(profile); err != nil {
		return nil, fmt.Errorf("failed to encode profile: %w", err)
	}

	return buf.Bytes(), nil
}

// ProfileTemplate is the template for generating HTML enrollment pages
const ProfileTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.TenantName}} - Device Enrollment</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f7;
        }
        .card {
            background: white;
            border-radius: 12px;
            padding: 30px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        h1 { color: #1d1d1f; margin-bottom: 10px; }
        p { color: #86868b; line-height: 1.6; }
        .btn {
            display: inline-block;
            background: #0071e3;
            color: white;
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            font-weight: 500;
            margin-top: 20px;
        }
        .btn:hover { background: #0077ed; }
        .steps { margin-top: 30px; }
        .step {
            display: flex;
            align-items: flex-start;
            margin-bottom: 15px;
        }
        .step-num {
            background: #0071e3;
            color: white;
            width: 28px;
            height: 28px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            margin-right: 15px;
            flex-shrink: 0;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>Enroll Your Device</h1>
        <p>Welcome to {{.TenantName}} device management. Click the button below to download and install the enrollment profile.</p>
        
        {{if .APNsTopic}}
        <a href="/enroll/{{.TenantID}}/profile" class="btn">Download Profile</a>
        {{else}}
        <div style="background:#fff3cd;border:1px solid #ffc107;border-radius:8px;padding:15px;margin:15px 0;color:#856404">
            <strong>⚠️ Not Ready:</strong> This tenant does not have an APNs push certificate configured yet.
            An administrator must upload an APNs certificate before devices can enroll.
            <br><br>
            <a href="https://mdmcert.download" target="_blank" style="color:#0071e3">Get a free APNs certificate →</a>
        </div>
        {{end}}
        
        <div class="steps">
            <div class="step">
                <span class="step-num">1</span>
                <span>Click "Download Profile" above</span>
            </div>
            <div class="step">
                <span class="step-num">2</span>
                <span>Open System Settings → Privacy & Security → Profiles</span>
            </div>
            <div class="step">
                <span class="step-num">3</span>
                <span>Select the downloaded profile and click "Install"</span>
            </div>
            <div class="step">
                <span class="step-num">4</span>
                <span>Enter your Mac password when prompted</span>
            </div>
        </div>
    </div>
</body>
</html>`

// GenerateEnrollmentPage creates an HTML enrollment page
func (g *Generator) GenerateEnrollmentPage(cfg EnrollmentConfig) ([]byte, error) {
	tmpl, err := template.New("enroll").Parse(ProfileTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
