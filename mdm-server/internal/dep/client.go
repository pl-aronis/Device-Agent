package dep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// DEP Client for Apple Business Manager integration
// Documentation: https://developer.apple.com/documentation/devicemanagement/device_assignment/

const depBaseURL = "https://mdmenrollment.apple.com"

type Client struct {
	oauthToken  string // OAuth token for DEP API
	accessToken string // Session token
	httpClient  *http.Client
}

type Device struct {
	SerialNumber       string `json:"serial_number"`
	Model              string `json:"model"`
	Description        string `json:"description"`
	Color              string `json:"color"`
	ProfileStatus      string `json:"profile_status"` // empty, assigned, pushed, removed
	DeviceAssignedDate string `json:"device_assigned_date"`
}

type SyncResponse struct {
	Devices      []Device `json:"devices"`
	Cursor       string   `json:"cursor"`
	MoreToFollow bool     `json:"more_to_follow"`
}

func NewClient(consumerKey, consumerSecret, accessToken, accessSecret string) *Client {
	// In reality, DEP auth uses OAuth 1.0a.
	// For simplicity in this mock, we assume we have a valid session token
	// or use a library like gomobile/oauth1.
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchDevices retrieves the list of devices assigned to this MDM server in ABM
func (c *Client) FetchDevices(cursor string) ([]Device, string, error) {
	url := fmt.Sprintf("%s/server/devices", depBaseURL)

	reqBody := map[string]string{}
	if cursor != "" {
		reqBody["cursor"] = cursor
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-Protocol-Version", "3")
	// Add OAuth headers here

	// Mock response for testing
	if true {
		return []Device{
			{
				SerialNumber:  "C02XXXXX",
				Model:         "MacBook Pro",
				Description:   "Device Agent Test Device",
				ProfileStatus: "assigned",
			},
		}, "next_cursor_123", nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("DEP API error: %d", resp.StatusCode)
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, "", err
	}

	return syncResp.Devices, syncResp.Cursor, nil
}

// DefineProfile defines an enrollment profile in ABM
func (c *Client) DefineProfile(profile Profile) (string, error) {
	// POST /profile
	// Returns profile_uuid
	return uuid.New().String(), nil
}

type Profile struct {
	ProfileName       string   `json:"profile_name"`
	URL               string   `json:"url"` // URL to your MDM enrollment endpoint
	AwaitDeviceConfig bool     `json:"await_device_configured"`
	IsMDMRemovable    bool     `json:"is_mdm_removable"`
	OrgMagic          string   `json:"org_magic"`
	SkipSetupItems    []string `json:"skip_setup_items"`
}
