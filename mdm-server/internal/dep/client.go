package dep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DEP Client for Apple Business Manager integration
// Documentation: https://developer.apple.com/documentation/devicemanagement/device_assignment/

const DefaultDEPBaseURL = "https://mdmenrollment.apple.com"

// ClientConfig holds configuration for the DEP client
type ClientConfig struct {
	// BaseURL overrides the DEP API URL (for mock server)
	BaseURL string

	// UseMock enables built-in mock responses without hitting any server
	UseMock bool

	// OAuth 1.0a credentials for DEP API
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string

	// HTTPTimeout for API requests (default: 30s)
	HTTPTimeout time.Duration
}

// Client is the DEP API client
type Client struct {
	baseURL    string
	useMock    bool
	httpClient *http.Client

	// OAuth credentials
	consumerKey    string
	consumerSecret string
	accessToken    string
	accessSecret   string

	// Session token (obtained from /session endpoint)
	sessionToken string
	sessionMu    sync.RWMutex
}

// Device represents a device in Apple Business Manager
type Device struct {
	SerialNumber       string `json:"serial_number"`
	Model              string `json:"model"`
	Description        string `json:"description"`
	Color              string `json:"color"`
	ProfileStatus      string `json:"profile_status"` // empty, assigned, pushed, removed
	ProfileUUID        string `json:"profile_uuid,omitempty"`
	ProfileAssignTime  string `json:"profile_assign_time,omitempty"`
	DeviceAssignedDate string `json:"device_assigned_date"`
	DeviceAssignedBy   string `json:"device_assigned_by,omitempty"`
	OpType             string `json:"op_type,omitempty"` // added, modified, deleted
	OpDate             string `json:"op_date,omitempty"`
	DeviceFamily       string `json:"device_family,omitempty"`
	OS                 string `json:"os,omitempty"`
	ResponseStatus     string `json:"response_status,omitempty"`
}

// SyncResponse is the response from the device sync endpoint
type SyncResponse struct {
	Devices      []Device `json:"devices"`
	Cursor       string   `json:"cursor"`
	MoreToFollow bool     `json:"more_to_follow"`
	FetchedUntil string   `json:"fetched_until,omitempty"`
}

// Profile represents a DEP enrollment profile
type Profile struct {
	ProfileName           string   `json:"profile_name"`
	ProfileUUID           string   `json:"profile_uuid,omitempty"`
	URL                   string   `json:"url"` // MDM enrollment URL
	AllowPairing          bool     `json:"allow_pairing"`
	AutoAdvanceSetup      bool     `json:"auto_advance_setup"`
	AwaitDeviceConfigured bool     `json:"await_device_configured"`
	Department            string   `json:"department,omitempty"`
	IsSupervised          bool     `json:"is_supervised"`
	IsMandatory           bool     `json:"is_mandatory"`
	IsMDMRemovable        bool     `json:"is_mdm_removable"`
	Language              string   `json:"language,omitempty"`
	OrgMagic              string   `json:"org_magic,omitempty"`
	Region                string   `json:"region,omitempty"`
	SkipSetupItems        []string `json:"skip_setup_items,omitempty"`
	SupportEmailAddress   string   `json:"support_email_address,omitempty"`
	SupportPhoneNumber    string   `json:"support_phone_number,omitempty"`
	AnchorCerts           []string `json:"anchor_certs,omitempty"`
	SupervisingHostCerts  []string `json:"supervising_host_certs,omitempty"`
	ConfigurationWebURL   string   `json:"configuration_web_url,omitempty"`
}

// ProfileAssignRequest is the request body for assigning profiles to devices
type ProfileAssignRequest struct {
	ProfileUUID string   `json:"profile_uuid"`
	Devices     []string `json:"devices"` // Serial numbers
}

// ProfileAssignResponse is the response from profile assignment
type ProfileAssignResponse struct {
	Devices map[string]string `json:"devices"` // serial -> status (SUCCESS, NOT_ACCESSIBLE, FAILED)
}

// AccountDetail contains information about the DEP account
type AccountDetail struct {
	ServerName    string `json:"server_name"`
	ServerUUID    string `json:"server_uuid"`
	FacilitatorID string `json:"facilitator_id,omitempty"`
	AdminID       string `json:"admin_id"`
	OrgName       string `json:"org_name"`
	OrgEmail      string `json:"org_email"`
	OrgPhone      string `json:"org_phone"`
	OrgAddress    string `json:"org_address"`
	OrgType       string `json:"org_type"` // edu, org
	OrgVersion    string `json:"org_version"`
	OrgID         string `json:"org_id"`
	OrgIDHash     string `json:"org_id_hash"`
}

// NewClient creates a new DEP client with default settings
// Deprecated: Use NewClientWithConfig for better configurability
func NewClient(consumerKey, consumerSecret, accessToken, accessSecret string) *Client {
	return NewClientWithConfig(ClientConfig{
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		AccessToken:    accessToken,
		AccessSecret:   accessSecret,
	})
}

// NewClientWithConfig creates a new DEP client with the given configuration
func NewClientWithConfig(cfg ClientConfig) *Client {
	baseURL := DefaultDEPBaseURL
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:        baseURL,
		useMock:        cfg.UseMock,
		consumerKey:    cfg.ConsumerKey,
		consumerSecret: cfg.ConsumerSecret,
		accessToken:    cfg.AccessToken,
		accessSecret:   cfg.AccessSecret,
		httpClient:     &http.Client{Timeout: timeout},
	}
}

// NewClientFromEnv creates a DEP client from environment variables
// Environment variables:
//   - DEP_MOCK=true: Use mock responses
//   - DEP_MOCK_URL: URL of mock DEP server
//   - DEP_CONSUMER_KEY, DEP_CONSUMER_SECRET: OAuth consumer credentials
//   - DEP_ACCESS_TOKEN, DEP_ACCESS_SECRET: OAuth access credentials
func NewClientFromEnv() *Client {
	if os.Getenv("DEP_MOCK") == "true" {
		return NewClientWithConfig(ClientConfig{
			UseMock: true,
		})
	}

	if mockURL := os.Getenv("DEP_MOCK_URL"); mockURL != "" {
		return NewClientWithConfig(ClientConfig{
			BaseURL: mockURL,
		})
	}

	return NewClientWithConfig(ClientConfig{
		ConsumerKey:    os.Getenv("DEP_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("DEP_CONSUMER_SECRET"),
		AccessToken:    os.Getenv("DEP_ACCESS_TOKEN"),
		AccessSecret:   os.Getenv("DEP_ACCESS_SECRET"),
	})
}

// doRequest performs an authenticated request to the DEP API
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("X-Server-Protocol-Version", "3")

	// Add session token if we have one
	c.sessionMu.RLock()
	if c.sessionToken != "" {
		req.Header.Set("X-ADM-Auth-Session", c.sessionToken)
	}
	c.sessionMu.RUnlock()

	// TODO: Add OAuth 1.0a signature headers
	// For production, use a library like github.com/dghubble/oauth1
	// Authorization: OAuth oauth_consumer_key="...", oauth_token="...", ...

	return c.httpClient.Do(req)
}

// GetSessionToken obtains a session token from the DEP API
func (c *Client) GetSessionToken() (string, error) {
	if c.useMock {
		token := "mock-session-token-" + uuid.New().String()[:8]
		c.sessionMu.Lock()
		c.sessionToken = token
		c.sessionMu.Unlock()
		return token, nil
	}

	resp, err := c.doRequest("GET", "/session", nil)
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("session request failed: %d", resp.StatusCode)
	}

	var result struct {
		AuthSessionToken string `json:"auth_session_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode session response: %w", err)
	}

	c.sessionMu.Lock()
	c.sessionToken = result.AuthSessionToken
	c.sessionMu.Unlock()

	return result.AuthSessionToken, nil
}

// GetAccountDetail retrieves account information from DEP
func (c *Client) GetAccountDetail() (*AccountDetail, error) {
	if c.useMock {
		return &AccountDetail{
			ServerName: "Mock MDM Server",
			ServerUUID: uuid.New().String(),
			AdminID:    "admin@example.com",
			OrgName:    "Mock Organization",
			OrgEmail:   "org@example.com",
			OrgPhone:   "+1-555-0100",
			OrgAddress: "123 Mock Street, Test City",
			OrgType:    "org",
			OrgVersion: "v2",
			OrgID:      "MOCK123",
		}, nil
	}

	resp, err := c.doRequest("GET", "/account", nil)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("account request failed: %d", resp.StatusCode)
	}

	var account AccountDetail
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("decode account response: %w", err)
	}

	return &account, nil
}

// FetchDevices retrieves the list of devices assigned to this MDM server in ABM
func (c *Client) FetchDevices(cursor string) ([]Device, string, error) {
	if c.useMock {
		return c.mockFetchDevices(cursor)
	}

	reqBody := map[string]interface{}{
		"limit": 1000,
	}
	if cursor != "" {
		reqBody["cursor"] = cursor
	}

	resp, err := c.doRequest("POST", "/server/devices", reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("fetch devices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch devices failed: %d", resp.StatusCode)
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, "", fmt.Errorf("decode sync response: %w", err)
	}

	return syncResp.Devices, syncResp.Cursor, nil
}

// SyncDevices performs a sync to get device changes since the last cursor
func (c *Client) SyncDevices(cursor string) (*SyncResponse, error) {
	if c.useMock {
		devices, newCursor, err := c.mockFetchDevices(cursor)
		if err != nil {
			return nil, err
		}
		return &SyncResponse{
			Devices:      devices,
			Cursor:       newCursor,
			MoreToFollow: false,
		}, nil
	}

	reqBody := map[string]string{}
	if cursor != "" {
		reqBody["cursor"] = cursor
	}

	resp, err := c.doRequest("POST", "/devices/sync", reqBody)
	if err != nil {
		return nil, fmt.Errorf("sync devices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync devices failed: %d", resp.StatusCode)
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("decode sync response: %w", err)
	}

	return &syncResp, nil
}

// DefineProfile creates or updates an enrollment profile in ABM
func (c *Client) DefineProfile(profile Profile) (string, error) {
	if c.useMock {
		return c.mockDefineProfile(profile)
	}

	resp, err := c.doRequest("POST", "/profile", profile)
	if err != nil {
		return "", fmt.Errorf("define profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("define profile failed: %d", resp.StatusCode)
	}

	var result struct {
		ProfileUUID string `json:"profile_uuid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode profile response: %w", err)
	}

	return result.ProfileUUID, nil
}

// GetProfile retrieves a profile by UUID
func (c *Client) GetProfile(profileUUID string) (*Profile, error) {
	if c.useMock {
		return &Profile{
			ProfileUUID:  profileUUID,
			ProfileName:  "Mock Profile",
			URL:          "https://mdm.example.com/enroll",
			IsSupervised: true,
			IsMandatory:  true,
		}, nil
	}

	resp, err := c.doRequest("GET", "/profile?profile_uuid="+profileUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get profile failed: %d", resp.StatusCode)
	}

	var profile Profile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}

	return &profile, nil
}

// AssignProfile assigns a profile to one or more devices
func (c *Client) AssignProfile(profileUUID string, serialNumbers []string) (*ProfileAssignResponse, error) {
	if c.useMock {
		return c.mockAssignProfile(profileUUID, serialNumbers)
	}

	req := ProfileAssignRequest{
		ProfileUUID: profileUUID,
		Devices:     serialNumbers,
	}

	resp, err := c.doRequest("PUT", "/profile/devices", req)
	if err != nil {
		return nil, fmt.Errorf("assign profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("assign profile failed: %d", resp.StatusCode)
	}

	var result ProfileAssignResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode assign response: %w", err)
	}

	return &result, nil
}

// RemoveProfile removes a profile from devices
func (c *Client) RemoveProfile(serialNumbers []string) (*ProfileAssignResponse, error) {
	if c.useMock {
		result := make(map[string]string)
		for _, serial := range serialNumbers {
			result[serial] = "SUCCESS"
		}
		return &ProfileAssignResponse{Devices: result}, nil
	}

	req := map[string]interface{}{
		"devices": serialNumbers,
	}

	resp, err := c.doRequest("DELETE", "/profile/devices", req)
	if err != nil {
		return nil, fmt.Errorf("remove profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remove profile failed: %d", resp.StatusCode)
	}

	var result ProfileAssignResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode remove response: %w", err)
	}

	return &result, nil
}

// DisownDevices removes devices from DEP (cannot be undone)
func (c *Client) DisownDevices(serialNumbers []string) (map[string]string, error) {
	if c.useMock {
		result := make(map[string]string)
		for _, serial := range serialNumbers {
			result[serial] = "SUCCESS"
		}
		return result, nil
	}

	req := map[string]interface{}{
		"devices": serialNumbers,
	}

	resp, err := c.doRequest("POST", "/devices/disown", req)
	if err != nil {
		return nil, fmt.Errorf("disown devices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("disown devices failed: %d", resp.StatusCode)
	}

	var result struct {
		Devices map[string]string `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode disown response: %w", err)
	}

	return result.Devices, nil
}

// === Mock implementations ===

var (
	mockDevices   = make(map[string]Device)
	mockProfiles  = make(map[string]Profile)
	mockDeviceMu  sync.RWMutex
	mockProfileMu sync.RWMutex
)

// AddMockDevice adds a device to the mock data store (for testing)
func AddMockDevice(device Device) {
	mockDeviceMu.Lock()
	defer mockDeviceMu.Unlock()
	if device.DeviceAssignedDate == "" {
		device.DeviceAssignedDate = time.Now().Format(time.RFC3339)
	}
	if device.ProfileStatus == "" {
		device.ProfileStatus = "empty"
	}
	mockDevices[device.SerialNumber] = device
}

// ClearMockData clears all mock data (for testing)
func ClearMockData() {
	mockDeviceMu.Lock()
	mockProfileMu.Lock()
	defer mockDeviceMu.Unlock()
	defer mockProfileMu.Unlock()
	mockDevices = make(map[string]Device)
	mockProfiles = make(map[string]Profile)
}

// GetMockDevices returns all mock devices (for testing)
func GetMockDevices() []Device {
	mockDeviceMu.RLock()
	defer mockDeviceMu.RUnlock()
	devices := make([]Device, 0, len(mockDevices))
	for _, d := range mockDevices {
		devices = append(devices, d)
	}
	return devices
}

func (c *Client) mockFetchDevices(cursor string) ([]Device, string, error) {
	mockDeviceMu.RLock()
	defer mockDeviceMu.RUnlock()

	// If no mock devices configured, return some defaults
	if len(mockDevices) == 0 {
		return []Device{
			{
				SerialNumber:       "C02MOCK001",
				Model:              "MacBook Pro (16-inch, 2023)",
				Description:        "Mock Test Device 1",
				Color:              "Space Gray",
				ProfileStatus:      "empty",
				DeviceAssignedDate: time.Now().Format(time.RFC3339),
				DeviceFamily:       "Mac",
				OS:                 "macOS",
			},
			{
				SerialNumber:       "C02MOCK002",
				Model:              "Mac mini (M2, 2023)",
				Description:        "Mock Test Device 2",
				Color:              "Silver",
				ProfileStatus:      "assigned",
				DeviceAssignedDate: time.Now().Format(time.RFC3339),
				DeviceFamily:       "Mac",
				OS:                 "macOS",
			},
		}, "mock-cursor-" + uuid.New().String()[:8], nil
	}

	devices := make([]Device, 0, len(mockDevices))
	for _, d := range mockDevices {
		devices = append(devices, d)
	}

	return devices, "mock-cursor-" + uuid.New().String()[:8], nil
}

func (c *Client) mockDefineProfile(profile Profile) (string, error) {
	mockProfileMu.Lock()
	defer mockProfileMu.Unlock()

	profileUUID := uuid.New().String()
	profile.ProfileUUID = profileUUID
	mockProfiles[profileUUID] = profile

	return profileUUID, nil
}

func (c *Client) mockAssignProfile(profileUUID string, serialNumbers []string) (*ProfileAssignResponse, error) {
	mockDeviceMu.Lock()
	defer mockDeviceMu.Unlock()

	result := make(map[string]string)
	for _, serial := range serialNumbers {
		if device, exists := mockDevices[serial]; exists {
			device.ProfileStatus = "assigned"
			device.ProfileUUID = profileUUID
			device.ProfileAssignTime = time.Now().Format(time.RFC3339)
			mockDevices[serial] = device
			result[serial] = "SUCCESS"
		} else {
			result[serial] = "NOT_ACCESSIBLE"
		}
	}

	return &ProfileAssignResponse{Devices: result}, nil
}
