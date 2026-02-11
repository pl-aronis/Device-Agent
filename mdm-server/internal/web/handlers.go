package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"mdm-server/internal/dep"
	"mdm-server/internal/profile"
	"mdm-server/internal/scep"
	"mdm-server/internal/store"

	"github.com/golang-jwt/jwt/v5"
)

// Handler handles admin web UI requests
type Handler struct {
	tenantStore  *store.TenantStore
	deviceStore  *store.DeviceStore
	commandStore *store.CommandStore
	depClient    *dep.Client
	scepHandler  *scep.Handler
	profileGen   *profile.Generator
	serverURL    string
	jwtSecret    []byte
	templates    *template.Template
}

// Config for the web handler
type Config struct {
	TenantStore  *store.TenantStore
	DeviceStore  *store.DeviceStore
	CommandStore *store.CommandStore
	DEPClient    *dep.Client
	SCEPHandler  *scep.Handler
	ServerURL    string
	JWTSecret    string
}

// NewHandler creates a new web handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		tenantStore:  cfg.TenantStore,
		deviceStore:  cfg.DeviceStore,
		commandStore: cfg.CommandStore,
		depClient:    cfg.DEPClient,
		scepHandler:  cfg.SCEPHandler,
		profileGen:   profile.NewGenerator(),
		serverURL:    cfg.ServerURL,
		jwtSecret:    []byte(cfg.JWTSecret),
	}
}

// RegisterRoutes registers all web routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Admin UI
	mux.HandleFunc("/admin/", h.handleAdmin)
	mux.HandleFunc("/admin/login", h.handleLogin)
	mux.HandleFunc("/admin/tenants", h.handleTenants)
	mux.HandleFunc("/admin/tenants/", h.handleTenantDetail)

	// Enrollment endpoints
	mux.HandleFunc("/enroll/", h.handleEnroll)

	// API endpoints (JSON)
	mux.HandleFunc("/api/tenants", h.handleAPITenants)
	mux.HandleFunc("/api/tenants/", h.handleAPITenantOperations)
}

// handleAdmin serves the admin dashboard
func (h *Handler) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/" && r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}

	tenants, _ := h.tenantStore.List()

	// Get device counts
	type tenantStats struct {
		*store.Tenant
		DeviceCount int
	}
	stats := make([]tenantStats, len(tenants))
	for i, t := range tenants {
		count, _ := h.tenantStore.GetDeviceCount(t.ID)
		stats[i] = tenantStats{t, count}
	}

	h.renderHTML(w, "dashboard", map[string]interface{}{
		"Title":   "MDM Dashboard",
		"Tenants": stats,
	})
}

// handleLogin handles admin authentication
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.renderHTML(w, "login", map[string]interface{}{
			"Title": "Login",
		})
		return
	}

	// POST: Process login
	email := r.FormValue("email")
	password := r.FormValue("password")

	// TODO: Validate against database
	// For now, accept any login for development
	if email == "" || password == "" {
		http.Error(w, "Email and password required", 400)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", 500)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "mdm_auth",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   86400, // 24 hours
	})

	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

// handleTenants handles tenant list and creation
func (h *Handler) handleTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tenants, _ := h.tenantStore.List()
		h.renderHTML(w, "tenants", map[string]interface{}{
			"Title":   "Tenants",
			"Tenants": tenants,
		})
		return
	}

	// POST: Create tenant
	name := r.FormValue("name")
	domain := r.FormValue("domain")

	if name == "" {
		http.Error(w, "Name is required", 400)
		return
	}

	tenant, err := h.tenantStore.Create(name, domain)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/admin/tenants/"+tenant.ID, http.StatusSeeOther)
}

// handleTenantDetail handles individual tenant operations
func (h *Handler) handleTenantDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}

	tenantID := parts[3]
	tenant, err := h.tenantStore.GetByID(tenantID)
	if err != nil || tenant == nil {
		http.NotFound(w, r)
		return
	}

	// Handle sub-routes
	if len(parts) >= 5 {
		switch parts[4] {
		case "devices":
			h.handleTenantDevices(w, r, tenant)
			return
		case "abm":
			h.handleTenantABM(w, r, tenant)
			return
		case "apns":
			h.handleTenantAPNs(w, r, tenant)
			return
		}
	}

	// Get devices for this tenant
	devices, _ := h.deviceStore.ListByTenant(tenantID)
	deviceCount, _ := h.tenantStore.GetDeviceCount(tenantID)

	h.renderHTML(w, "tenant_detail", map[string]interface{}{
		"Title":       tenant.Name,
		"Tenant":      tenant,
		"Devices":     devices,
		"DeviceCount": deviceCount,
		"HasCA":       tenant.CACertPEM != "",
		"HasAPNs":     tenant.APNsTopic != "",
	})
}

// handleTenantDevices handles device list for a tenant
func (h *Handler) handleTenantDevices(w http.ResponseWriter, r *http.Request, tenant *store.Tenant) {
	devices, _ := h.deviceStore.ListByTenant(tenant.ID)

	// JSON response for API
	if r.Header.Get("Accept") == "application/json" {
		json.NewEncoder(w).Encode(devices)
		return
	}

	h.renderHTML(w, "devices", map[string]interface{}{
		"Title":   fmt.Sprintf("%s - Devices", tenant.Name),
		"Tenant":  tenant,
		"Devices": devices,
	})
}

// handleTenantABM handles ABM configuration for a tenant
func (h *Handler) handleTenantABM(w http.ResponseWriter, r *http.Request, tenant *store.Tenant) {
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) >= 6 {
		switch parts[5] {
		case "sync":
			// Trigger ABM sync
			if h.depClient != nil {
				devices, _, err := h.depClient.FetchDevices("")
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "ok",
					"devices": len(devices),
				})
				return
			}
		}
	}

	h.renderHTML(w, "abm_connect", map[string]interface{}{
		"Title":  "Connect ABM",
		"Tenant": tenant,
	})
}

// handleTenantAPNs handles APNs certificate upload
func (h *Handler) handleTenantAPNs(w http.ResponseWriter, r *http.Request, tenant *store.Tenant) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Failed to parse form", 400)
		return
	}

	file, _, err := r.FormFile("certificate")
	if err != nil {
		http.Error(w, "Certificate file required", 400)
		return
	}
	defer file.Close()

	certData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read certificate", 500)
		return
	}

	topic := r.FormValue("topic")
	if topic == "" {
		http.Error(w, "APNs topic required", 400)
		return
	}

	// Calculate expiry (TODO: parse from certificate)
	expiresAt := time.Now().AddDate(1, 0, 0) // 1 year default

	if err := h.tenantStore.UpdateAPNs(tenant.ID, certData, nil, topic, expiresAt); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/admin/tenants/"+tenant.ID, http.StatusSeeOther)
}

// handleEnroll serves enrollment pages and profiles
func (h *Handler) handleEnroll(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}

	tenantID := parts[2]
	tenant, err := h.tenantStore.GetByID(tenantID)
	if err != nil || tenant == nil {
		http.NotFound(w, r)
		return
	}

	// Check if requesting profile download
	if len(parts) >= 4 && parts[3] == "profile" {
		h.serveEnrollmentProfile(w, r, tenant)
		return
	}

	// Serve enrollment page
	cfg := profile.EnrollmentConfig{
		TenantID:   tenant.ID,
		TenantName: tenant.Name,
		ServerURL:  h.serverURL,
	}

	page, err := h.profileGen.GenerateEnrollmentPage(cfg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(page)
}

// serveEnrollmentProfile generates and serves the .mobileconfig file
func (h *Handler) serveEnrollmentProfile(w http.ResponseWriter, r *http.Request, tenant *store.Tenant) {
	// Get CA certificate
	caCertBase64 := ""
	if tenant.CACertPEM != "" {
		// Extract DER from PEM and base64 encode
		// For simplicity, just base64 the PEM for now
		caCertBase64 = base64.StdEncoding.EncodeToString([]byte(tenant.CACertPEM))
	}

	cfg := profile.EnrollmentConfig{
		TenantID:     tenant.ID,
		TenantName:   tenant.Name,
		ServerURL:    h.serverURL,
		APNsTopic:    tenant.APNsTopic,
		CACertBase64: caCertBase64,
	}

	profileData, err := h.profileGen.GenerateEnrollmentProfile(cfg)
	if err != nil {
		log.Printf("Failed to generate profile: %v", err)
		http.Error(w, "Failed to generate profile", 500)
		return
	}

	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-enroll.mobileconfig\"", tenant.Domain))
	w.Write(profileData)
}

// handleAPITenants handles /api/tenants
func (h *Handler) handleAPITenants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		tenants, err := h.tenantStore.List()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(tenants)

	case http.MethodPost:
		var req struct {
			Name   string `json:"name"`
			Domain string `json:"domain"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", 400)
			return
		}

		tenant, err := h.tenantStore.Create(req.Name, req.Domain)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(tenant)

	default:
		http.Error(w, "Method not allowed", 405)
	}
}

// handleAPITenantOperations handles /api/tenants/{id}/...
func (h *Handler) handleAPITenantOperations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}

	tenantID := parts[3]
	tenant, err := h.tenantStore.GetByID(tenantID)
	if err != nil || tenant == nil {
		http.NotFound(w, r)
		return
	}

	// /api/tenants/{id}
	if len(parts) == 4 {
		json.NewEncoder(w).Encode(tenant)
		return
	}

	// /api/tenants/{id}/devices
	if parts[4] == "devices" {
		devices, err := h.deviceStore.ListByTenant(tenantID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(devices)
		return
	}

	http.NotFound(w, r)
}

// renderHTML renders an HTML template
func (h *Handler) renderHTML(w http.ResponseWriter, name string, data interface{}) {
	// Inline templates for simplicity (production would load from files)
	templates := map[string]string{
		"dashboard": `<!DOCTYPE html>
<html><head><title>{{.Title}}</title>
<style>
body { font-family: -apple-system, sans-serif; margin: 0; padding: 20px; background: #f5f5f7; }
.container { max-width: 1200px; margin: 0 auto; }
h1 { color: #1d1d1f; }
.card { background: white; border-radius: 12px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.btn { display: inline-block; background: #0071e3; color: white; padding: 10px 20px; border-radius: 8px; text-decoration: none; }
.btn:hover { background: #0077ed; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 12px; border-bottom: 1px solid #e0e0e0; }
th { color: #86868b; font-weight: 500; }
</style>
</head><body>
<div class="container">
<h1>MDM Dashboard</h1>
<div class="card">
<h2>Tenants</h2>
<table>
<tr><th>Name</th><th>Domain</th><th>Devices</th><th>Actions</th></tr>
{{range .Tenants}}
<tr>
<td><a href="/admin/tenants/{{.ID}}">{{.Name}}</a></td>
<td>{{.Domain}}</td>
<td>{{.DeviceCount}}</td>
<td><a href="/admin/tenants/{{.ID}}" class="btn">Manage</a></td>
</tr>
{{else}}
<tr><td colspan="4">No tenants. <a href="/admin/tenants?new=1">Create one</a></td></tr>
{{end}}
</table>
</div>
<a href="/admin/tenants?new=1" class="btn">+ New Tenant</a>
</div>
</body></html>`,

		"tenant_detail": `<!DOCTYPE html>
<html><head><title>{{.Title}}</title>
<style>
body { font-family: -apple-system, sans-serif; margin: 0; padding: 20px; background: #f5f5f7; }
.container { max-width: 1200px; margin: 0 auto; }
h1 { color: #1d1d1f; }
.card { background: white; border-radius: 12px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.btn { display: inline-block; background: #0071e3; color: white; padding: 10px 20px; border-radius: 8px; text-decoration: none; margin-right: 10px; }
.btn-secondary { background: #86868b; }
.status { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
.status-ok { background: #d4edda; color: #155724; }
.status-missing { background: #fff3cd; color: #856404; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 12px; border-bottom: 1px solid #e0e0e0; }
</style>
</head><body>
<div class="container">
<p><a href="/admin/">← Dashboard</a></p>
<h1>{{.Tenant.Name}}</h1>
<div class="card">
<h3>Configuration</h3>
<p>Domain: <strong>{{.Tenant.Domain}}</strong></p>
<p>SCEP CA: <span class="status {{if .HasCA}}status-ok">Configured{{else}}status-missing">Not configured{{end}}</span></p>
<p>APNs: <span class="status {{if .HasAPNs}}status-ok">{{.Tenant.APNsTopic}}{{else}}status-missing">Not configured{{end}}</span></p>
</div>
<div class="card">
<h3>Enrollment</h3>
<p>Enrollment URL: <code>/enroll/{{.Tenant.ID}}</code></p>
<a href="/enroll/{{.Tenant.ID}}" target="_blank" class="btn">Open Enrollment Page</a>
<a href="/enroll/{{.Tenant.ID}}/profile" class="btn btn-secondary">Download Profile</a>
</div>
<div class="card">
<h3>Devices ({{.DeviceCount}})</h3>
<table>
<tr><th>UDID</th><th>Name</th><th>Model</th><th>Last Seen</th></tr>
{{range .Devices}}
<tr>
<td>{{.UDID}}</td>
<td>{{.DeviceName}}</td>
<td>{{.Model}}</td>
<td>{{.LastSeenAt.Format "2006-01-02 15:04"}}</td>
</tr>
{{else}}
<tr><td colspan="4">No enrolled devices</td></tr>
{{end}}
</table>
</div>
</div>
</body></html>`,

		"login": `<!DOCTYPE html>
<html><head><title>Login</title>
<style>
body { font-family: -apple-system, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f7; }
.card { background: white; border-radius: 12px; padding: 40px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); width: 300px; }
h1 { text-align: center; margin-bottom: 30px; }
input { width: 100%; padding: 12px; margin-bottom: 15px; border: 1px solid #d2d2d7; border-radius: 8px; box-sizing: border-box; }
button { width: 100%; padding: 12px; background: #0071e3; color: white; border: none; border-radius: 8px; cursor: pointer; }
</style>
</head><body>
<div class="card">
<h1>MDM Admin</h1>
<form method="POST">
<input type="email" name="email" placeholder="Email" required>
<input type="password" name="password" placeholder="Password" required>
<button type="submit">Login</button>
</form>
</div>
</body></html>`,

		"tenants": `<!DOCTYPE html>
<html><head><title>{{.Title}}</title>
<style>
body { font-family: -apple-system, sans-serif; margin: 0; padding: 20px; background: #f5f5f7; }
.container { max-width: 600px; margin: 0 auto; }
h1 { color: #1d1d1f; }
.card { background: white; border-radius: 12px; padding: 30px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
label { display: block; margin-bottom: 5px; font-weight: 500; color: #1d1d1f; }
input { width: 100%; padding: 12px; margin-bottom: 15px; border: 1px solid #d2d2d7; border-radius: 8px; box-sizing: border-box; font-size: 16px; }
.btn { display: inline-block; background: #0071e3; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; border: none; cursor: pointer; font-size: 16px; }
.btn:hover { background: #0077ed; }
.btn-secondary { background: #86868b; margin-left: 10px; }
</style>
</head><body>
<div class="container">
<p><a href="/admin/">← Dashboard</a></p>
<h1>Create New Tenant</h1>
<div class="card">
<form method="POST" action="/admin/tenants">
<label for="name">Organization Name *</label>
<input type="text" id="name" name="name" placeholder="e.g., Acme Corporation" required>

<label for="domain">Domain (optional)</label>
<input type="text" id="domain" name="domain" placeholder="e.g., acme.com">

<button type="submit" class="btn">Create Tenant</button>
<a href="/admin/" class="btn btn-secondary">Cancel</a>
</form>
</div>
</div>
</body></html>`,
	}

	tmplStr, ok := templates[name]
	if !ok {
		http.Error(w, "Template not found", 500)
		return
	}

	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		http.Error(w, "Template error", 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, data)
}
