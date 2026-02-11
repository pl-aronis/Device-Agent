package dep

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockDEPServer simulates Apple's DEP API for testing purposes
// This can be run as a standalone server for integration testing
type MockDEPServer struct {
	devices     map[string]Device  // serial -> device
	profiles    map[string]Profile // uuid -> profile
	assignments map[string]string  // serial -> profile_uuid
	mu          sync.RWMutex

	// Configuration
	simulateLatency   time.Duration
	simulateErrors    bool
	errorRate         float32 // 0.0 to 1.0
	requireAuth       bool
	validSessionToken string
}

// MockServerOption is a functional option for configuring MockDEPServer
type MockServerOption func(*MockDEPServer)

// WithLatency adds simulated network latency
func WithLatency(d time.Duration) MockServerOption {
	return func(s *MockDEPServer) {
		s.simulateLatency = d
	}
}

// WithErrorSimulation enables random error simulation
func WithErrorSimulation(rate float32) MockServerOption {
	return func(s *MockDEPServer) {
		s.simulateErrors = true
		s.errorRate = rate
	}
}

// WithAuth requires authentication for requests
func WithAuth(token string) MockServerOption {
	return func(s *MockDEPServer) {
		s.requireAuth = true
		s.validSessionToken = token
	}
}

// NewMockDEPServer creates a new mock DEP server
func NewMockDEPServer(opts ...MockServerOption) *MockDEPServer {
	s := &MockDEPServer{
		devices:           make(map[string]Device),
		profiles:          make(map[string]Profile),
		assignments:       make(map[string]string),
		validSessionToken: "mock-session-" + uuid.New().String()[:8],
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// AddDevice adds a test device to the mock server
func (s *MockDEPServer) AddDevice(device Device) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if device.DeviceAssignedDate == "" {
		device.DeviceAssignedDate = time.Now().Format(time.RFC3339)
	}
	if device.ProfileStatus == "" {
		device.ProfileStatus = "empty"
	}
	s.devices[device.SerialNumber] = device
}

// AddDevices adds multiple test devices
func (s *MockDEPServer) AddDevices(devices []Device) {
	for _, d := range devices {
		s.AddDevice(d)
	}
}

// GetDevice returns a device by serial number
func (s *MockDEPServer) GetDevice(serial string) (Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.devices[serial]
	return d, ok
}

// GetAllDevices returns all devices
func (s *MockDEPServer) GetAllDevices() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	devices := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}
	return devices
}

// Clear removes all data from the mock server
func (s *MockDEPServer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.devices = make(map[string]Device)
	s.profiles = make(map[string]Profile)
	s.assignments = make(map[string]string)
}

// Handler returns an http.Handler for the mock DEP server
func (s *MockDEPServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// Session endpoint
	mux.HandleFunc("/session", s.handleSession)

	// Account endpoint
	mux.HandleFunc("/account", s.handleAccount)

	// Device endpoints
	mux.HandleFunc("/server/devices", s.handleServerDevices)
	mux.HandleFunc("/devices/sync", s.handleDevicesSync)
	mux.HandleFunc("/devices/disown", s.handleDevicesDisown)

	// Profile endpoints
	mux.HandleFunc("/profile", s.handleProfile)
	mux.HandleFunc("/profile/devices", s.handleProfileDevices)

	// Activation lock endpoints
	mux.HandleFunc("/device/activationlock", s.handleActivationLock)

	return s.middleware(mux)
}

// middleware adds common functionality to all requests
func (s *MockDEPServer) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate latency
		if s.simulateLatency > 0 {
			time.Sleep(s.simulateLatency)
		}

		// Check authentication
		if s.requireAuth && r.URL.Path != "/session" {
			token := r.Header.Get("X-ADM-Auth-Session")
			if token != s.validSessionToken {
				s.writeError(w, http.StatusUnauthorized, "Invalid session token")
				return
			}
		}

		// Set content type
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")

		// Log request
		log.Printf("[MockDEP] %s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func (s *MockDEPServer) writeError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *MockDEPServer) writeJSON(w http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleSession handles GET /session
func (s *MockDEPServer) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.writeJSON(w, map[string]string{
		"auth_session_token": s.validSessionToken,
	})
}

// handleAccount handles GET /account
func (s *MockDEPServer) handleAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.writeJSON(w, AccountDetail{
		ServerName: "Mock MDM Server",
		ServerUUID: uuid.New().String(),
		AdminID:    "admin@mockcompany.com",
		OrgName:    "Mock Company Inc.",
		OrgEmail:   "it@mockcompany.com",
		OrgPhone:   "+1-555-0123",
		OrgAddress: "123 Test Street, Mock City, MC 12345",
		OrgType:    "org",
		OrgVersion: "v2",
		OrgID:      "MOCKORG123",
		OrgIDHash:  "abc123def456",
	})
}

// handleServerDevices handles POST /server/devices
func (s *MockDEPServer) handleServerDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Cursor string `json:"cursor"`
		Limit  int    `json:"limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	s.mu.RLock()
	devices := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}
	s.mu.RUnlock()

	s.writeJSON(w, SyncResponse{
		Devices:      devices,
		Cursor:       "cursor-" + uuid.New().String()[:8],
		MoreToFollow: false,
		FetchedUntil: time.Now().Format(time.RFC3339),
	})
}

// handleDevicesSync handles POST /devices/sync
func (s *MockDEPServer) handleDevicesSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// For mock, sync behaves the same as fetch
	s.handleServerDevices(w, r)
}

// handleDevicesDisown handles POST /devices/disown
func (s *MockDEPServer) handleDevicesDisown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Devices []string `json:"devices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	result := make(map[string]string)
	for _, serial := range req.Devices {
		if _, exists := s.devices[serial]; exists {
			delete(s.devices, serial)
			delete(s.assignments, serial)
			result[serial] = "SUCCESS"
		} else {
			result[serial] = "NOT_ACCESSIBLE"
		}
	}
	s.mu.Unlock()

	s.writeJSON(w, map[string]interface{}{"devices": result})
}

// handleProfile handles profile CRUD operations
func (s *MockDEPServer) handleProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getProfile(w, r)
	case http.MethodPost:
		s.createProfile(w, r)
	case http.MethodDelete:
		s.deleteProfile(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *MockDEPServer) createProfile(w http.ResponseWriter, r *http.Request) {
	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	profileUUID := uuid.New().String()
	profile.ProfileUUID = profileUUID

	s.mu.Lock()
	s.profiles[profileUUID] = profile
	s.mu.Unlock()

	s.writeJSON(w, map[string]string{
		"profile_uuid": profileUUID,
	})
}

func (s *MockDEPServer) getProfile(w http.ResponseWriter, r *http.Request) {
	profileUUID := r.URL.Query().Get("profile_uuid")
	if profileUUID == "" {
		s.writeError(w, http.StatusBadRequest, "profile_uuid required")
		return
	}

	s.mu.RLock()
	profile, exists := s.profiles[profileUUID]
	s.mu.RUnlock()

	if !exists {
		s.writeError(w, http.StatusNotFound, "Profile not found")
		return
	}

	s.writeJSON(w, profile)
}

func (s *MockDEPServer) deleteProfile(w http.ResponseWriter, r *http.Request) {
	profileUUID := r.URL.Query().Get("profile_uuid")
	if profileUUID == "" {
		s.writeError(w, http.StatusBadRequest, "profile_uuid required")
		return
	}

	s.mu.Lock()
	delete(s.profiles, profileUUID)
	s.mu.Unlock()

	s.writeJSON(w, map[string]string{"status": "OK"})
}

// handleProfileDevices handles profile assignment to devices
func (s *MockDEPServer) handleProfileDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.assignProfile(w, r)
	case http.MethodDelete:
		s.removeProfile(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *MockDEPServer) assignProfile(w http.ResponseWriter, r *http.Request) {
	var req ProfileAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	result := make(map[string]string)
	for _, serial := range req.Devices {
		if device, exists := s.devices[serial]; exists {
			device.ProfileStatus = "assigned"
			device.ProfileUUID = req.ProfileUUID
			device.ProfileAssignTime = time.Now().Format(time.RFC3339)
			s.devices[serial] = device
			s.assignments[serial] = req.ProfileUUID
			result[serial] = "SUCCESS"
		} else {
			result[serial] = "NOT_ACCESSIBLE"
		}
	}
	s.mu.Unlock()

	s.writeJSON(w, map[string]interface{}{"devices": result})
}

func (s *MockDEPServer) removeProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Devices []string `json:"devices"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	s.mu.Lock()
	result := make(map[string]string)
	for _, serial := range req.Devices {
		if device, exists := s.devices[serial]; exists {
			device.ProfileStatus = "empty"
			device.ProfileUUID = ""
			device.ProfileAssignTime = ""
			s.devices[serial] = device
			delete(s.assignments, serial)
			result[serial] = "SUCCESS"
		} else {
			result[serial] = "NOT_ACCESSIBLE"
		}
	}
	s.mu.Unlock()

	s.writeJSON(w, map[string]interface{}{"devices": result})
}

// handleActivationLock handles activation lock operations
func (s *MockDEPServer) handleActivationLock(w http.ResponseWriter, r *http.Request) {
	// Mock implementation - just return success
	s.writeJSON(w, map[string]string{
		"serial_number":   "MOCK123",
		"response_status": "SUCCESS",
	})
}

// ListenAndServe starts the mock DEP server
func (s *MockDEPServer) ListenAndServe(addr string) error {
	log.Printf("[MockDEP] Starting mock DEP server on %s", addr)
	log.Printf("[MockDEP] Session token: %s", s.validSessionToken)
	return http.ListenAndServe(addr, s.Handler())
}

// CreateTestDevices returns a set of realistic test devices
func CreateTestDevices() []Device {
	return []Device{
		{
			SerialNumber:       "C02X1234ABCD",
			Model:              "MacBook Pro (16-inch, 2023)",
			Description:        "Engineering Team - Dev Machine",
			Color:              "Space Gray",
			ProfileStatus:      "empty",
			DeviceFamily:       "Mac",
			OS:                 "macOS",
			DeviceAssignedDate: time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			SerialNumber:       "C02Y5678EFGH",
			Model:              "MacBook Air (M2, 2023)",
			Description:        "Sales Team - Mobile",
			Color:              "Midnight",
			ProfileStatus:      "empty",
			DeviceFamily:       "Mac",
			OS:                 "macOS",
			DeviceAssignedDate: time.Now().Add(-15 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			SerialNumber:       "C02Z9012IJKL",
			Model:              "Mac mini (M2 Pro, 2023)",
			Description:        "Server Room - Build Server",
			Color:              "Silver",
			ProfileStatus:      "assigned",
			ProfileUUID:        "existing-profile-uuid",
			DeviceFamily:       "Mac",
			OS:                 "macOS",
			DeviceAssignedDate: time.Now().Add(-60 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			SerialNumber:       "DNPVG123ABCD",
			Model:              "iMac (24-inch, M3, 2024)",
			Description:        "Design Team - Primary",
			Color:              "Blue",
			ProfileStatus:      "pushed",
			ProfileUUID:        "design-profile-uuid",
			DeviceFamily:       "Mac",
			OS:                 "macOS",
			DeviceAssignedDate: time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			SerialNumber:       "F2LVWX456789",
			Model:              "Mac Studio (M2 Ultra, 2023)",
			Description:        "Video Production - Editing Suite",
			Color:              "Silver",
			ProfileStatus:      "empty",
			DeviceFamily:       "Mac",
			OS:                 "macOS",
			DeviceAssignedDate: time.Now().Add(-45 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
}

// CreateSkipSetupItems returns common setup items to skip for supervised devices
func CreateSkipSetupItems() []string {
	return []string{
		"AppleID",
		"Biometric",
		"Diagnostics",
		"DisplayTone",
		"FileVault",
		"iCloudDiagnostics",
		"iCloudStorage",
		"Location",
		"Payment",
		"Privacy",
		"Registration",
		"Restore",
		"ScreenTime",
		"Siri",
		"TOS",
		"UnlockWithWatch",
	}
}

// RunMockServer is a convenience function to run a mock DEP server
// with test data pre-populated
func RunMockServer(addr string) error {
	server := NewMockDEPServer()

	// Pre-populate with test devices
	server.AddDevices(CreateTestDevices())

	fmt.Printf(`
╔══════════════════════════════════════════════════════════════╗
║           Mock DEP Server for Testing                        ║
╠══════════════════════════════════════════════════════════════╣
║  Address: %s
║  Session Token: %s
║  
║  Endpoints:
║    GET  /session           - Get session token
║    GET  /account           - Get account details  
║    POST /server/devices    - List devices
║    POST /devices/sync      - Sync devices
║    POST /devices/disown    - Disown devices
║    POST /profile           - Create profile
║    GET  /profile           - Get profile
║    PUT  /profile/devices   - Assign profile
║    DELETE /profile/devices - Remove profile
║
║  Test Devices: %d pre-loaded
╚══════════════════════════════════════════════════════════════╝

`, addr, server.validSessionToken, len(server.devices))

	return server.ListenAndServe(addr)
}
