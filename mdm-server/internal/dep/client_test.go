package dep

import (
	"net/http/httptest"
	"testing"
)

func TestClient_FetchDevices_Mock(t *testing.T) {
	// Test with built-in mock
	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	devices, cursor, err := client.FetchDevices("")
	if err != nil {
		t.Fatalf("FetchDevices failed: %v", err)
	}

	if len(devices) == 0 {
		t.Error("Expected at least one device")
	}

	if cursor == "" {
		t.Error("Expected a cursor")
	}

	t.Logf("Fetched %d devices, cursor: %s", len(devices), cursor)
	for _, d := range devices {
		t.Logf("  - %s: %s (%s)", d.SerialNumber, d.Model, d.ProfileStatus)
	}
}

func TestClient_FetchDevices_CustomMockData(t *testing.T) {
	// Clear any existing mock data
	ClearMockData()

	// Add custom mock devices
	AddMockDevice(Device{
		SerialNumber:  "CUSTOM001",
		Model:         "Custom Mac",
		Description:   "Custom Test Device",
		ProfileStatus: "empty",
	})
	AddMockDevice(Device{
		SerialNumber:  "CUSTOM002",
		Model:         "Custom MacBook",
		Description:   "Another Test Device",
		ProfileStatus: "assigned",
	})

	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	devices, _, err := client.FetchDevices("")
	if err != nil {
		t.Fatalf("FetchDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}

	// Verify custom devices are returned
	found := make(map[string]bool)
	for _, d := range devices {
		found[d.SerialNumber] = true
	}

	if !found["CUSTOM001"] {
		t.Error("CUSTOM001 not found")
	}
	if !found["CUSTOM002"] {
		t.Error("CUSTOM002 not found")
	}

	// Cleanup
	ClearMockData()
}

func TestClient_DefineProfile_Mock(t *testing.T) {
	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	profile := Profile{
		ProfileName:           "Test MDM Profile",
		URL:                   "https://mdm.example.com/enroll",
		IsSupervised:          true,
		IsMandatory:           true,
		IsMDMRemovable:        false,
		AwaitDeviceConfigured: true,
		SkipSetupItems:        []string{"AppleID", "Siri", "Location"},
	}

	profileUUID, err := client.DefineProfile(profile)
	if err != nil {
		t.Fatalf("DefineProfile failed: %v", err)
	}

	if profileUUID == "" {
		t.Error("Expected a profile UUID")
	}

	t.Logf("Created profile: %s", profileUUID)
}

func TestClient_AssignProfile_Mock(t *testing.T) {
	ClearMockData()

	// Add devices
	AddMockDevice(Device{
		SerialNumber: "ASSIGN001",
		Model:        "MacBook Pro",
	})
	AddMockDevice(Device{
		SerialNumber: "ASSIGN002",
		Model:        "Mac mini",
	})

	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	// Create a profile
	profileUUID, err := client.DefineProfile(Profile{
		ProfileName: "Assignment Test",
		URL:         "https://mdm.example.com/enroll",
	})
	if err != nil {
		t.Fatalf("DefineProfile failed: %v", err)
	}

	// Assign to devices
	result, err := client.AssignProfile(profileUUID, []string{"ASSIGN001", "ASSIGN002", "NONEXISTENT"})
	if err != nil {
		t.Fatalf("AssignProfile failed: %v", err)
	}

	// Check results
	if result.Devices["ASSIGN001"] != "SUCCESS" {
		t.Errorf("ASSIGN001 should be SUCCESS, got %s", result.Devices["ASSIGN001"])
	}
	if result.Devices["ASSIGN002"] != "SUCCESS" {
		t.Errorf("ASSIGN002 should be SUCCESS, got %s", result.Devices["ASSIGN002"])
	}
	if result.Devices["NONEXISTENT"] != "NOT_ACCESSIBLE" {
		t.Errorf("NONEXISTENT should be NOT_ACCESSIBLE, got %s", result.Devices["NONEXISTENT"])
	}

	// Verify devices are updated
	devices := GetMockDevices()
	for _, d := range devices {
		if d.SerialNumber == "ASSIGN001" || d.SerialNumber == "ASSIGN002" {
			if d.ProfileStatus != "assigned" {
				t.Errorf("%s should be assigned, got %s", d.SerialNumber, d.ProfileStatus)
			}
			if d.ProfileUUID != profileUUID {
				t.Errorf("%s profile UUID mismatch", d.SerialNumber)
			}
		}
	}

	ClearMockData()
}

func TestClient_GetSessionToken_Mock(t *testing.T) {
	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	token, err := client.GetSessionToken()
	if err != nil {
		t.Fatalf("GetSessionToken failed: %v", err)
	}

	if token == "" {
		t.Error("Expected a session token")
	}

	if len(token) < 10 {
		t.Error("Token seems too short")
	}

	t.Logf("Got session token: %s", token)
}

func TestClient_GetAccountDetail_Mock(t *testing.T) {
	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	account, err := client.GetAccountDetail()
	if err != nil {
		t.Fatalf("GetAccountDetail failed: %v", err)
	}

	if account.OrgName == "" {
		t.Error("Expected org name")
	}

	if account.ServerUUID == "" {
		t.Error("Expected server UUID")
	}

	t.Logf("Account: %s (%s)", account.OrgName, account.OrgType)
}

func TestClient_RemoveProfile_Mock(t *testing.T) {
	ClearMockData()

	// Add a device with assigned profile
	AddMockDevice(Device{
		SerialNumber:  "REMOVE001",
		Model:         "MacBook Pro",
		ProfileStatus: "assigned",
		ProfileUUID:   "some-profile-uuid",
	})

	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	result, err := client.RemoveProfile([]string{"REMOVE001"})
	if err != nil {
		t.Fatalf("RemoveProfile failed: %v", err)
	}

	if result.Devices["REMOVE001"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", result.Devices["REMOVE001"])
	}

	ClearMockData()
}

func TestClient_DisownDevices_Mock(t *testing.T) {
	ClearMockData()

	AddMockDevice(Device{
		SerialNumber: "DISOWN001",
		Model:        "Mac mini",
	})

	client := NewClientWithConfig(ClientConfig{
		UseMock: true,
	})

	// Verify device exists
	devices := GetMockDevices()
	if len(devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(devices))
	}

	result, err := client.DisownDevices([]string{"DISOWN001"})
	if err != nil {
		t.Fatalf("DisownDevices failed: %v", err)
	}

	if result["DISOWN001"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", result["DISOWN001"])
	}

	ClearMockData()
}

// === Mock Server Tests ===

func TestMockDEPServer_FetchDevices(t *testing.T) {
	// Create mock server
	server := NewMockDEPServer()
	server.AddDevice(Device{
		SerialNumber: "SERVER001",
		Model:        "MacBook Pro",
		Description:  "Test Device 1",
	})
	server.AddDevice(Device{
		SerialNumber: "SERVER002",
		Model:        "Mac mini",
		Description:  "Test Device 2",
	})

	// Start test server
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// Create client pointing to mock server
	client := NewClientWithConfig(ClientConfig{
		BaseURL: ts.URL,
	})

	// Test fetching devices
	devices, cursor, err := client.FetchDevices("")
	if err != nil {
		t.Fatalf("FetchDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}

	if cursor == "" {
		t.Error("Expected cursor")
	}

	t.Logf("Fetched %d devices from mock server", len(devices))
}

func TestMockDEPServer_AssignProfile(t *testing.T) {
	server := NewMockDEPServer()
	server.AddDevice(Device{
		SerialNumber: "SRVASSIGN001",
		Model:        "MacBook Pro",
	})

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := NewClientWithConfig(ClientConfig{
		BaseURL: ts.URL,
	})

	// Create profile
	profileUUID, err := client.DefineProfile(Profile{
		ProfileName: "Server Test Profile",
		URL:         "https://mdm.example.com/enroll",
	})
	if err != nil {
		t.Fatalf("DefineProfile failed: %v", err)
	}

	// Assign profile
	result, err := client.AssignProfile(profileUUID, []string{"SRVASSIGN001"})
	if err != nil {
		t.Fatalf("AssignProfile failed: %v", err)
	}

	if result.Devices["SRVASSIGN001"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", result.Devices["SRVASSIGN001"])
	}

	// Verify device state
	device, ok := server.GetDevice("SRVASSIGN001")
	if !ok {
		t.Fatal("Device not found")
	}

	if device.ProfileStatus != "assigned" {
		t.Errorf("Expected assigned, got %s", device.ProfileStatus)
	}

	if device.ProfileUUID != profileUUID {
		t.Errorf("Profile UUID mismatch")
	}
}

func TestMockDEPServer_GetAccount(t *testing.T) {
	server := NewMockDEPServer()
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := NewClientWithConfig(ClientConfig{
		BaseURL: ts.URL,
	})

	account, err := client.GetAccountDetail()
	if err != nil {
		t.Fatalf("GetAccountDetail failed: %v", err)
	}

	if account.OrgName == "" {
		t.Error("Expected org name")
	}

	t.Logf("Account: %s", account.OrgName)
}

func TestMockDEPServer_GetSessionToken(t *testing.T) {
	server := NewMockDEPServer()
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := NewClientWithConfig(ClientConfig{
		BaseURL: ts.URL,
	})

	token, err := client.GetSessionToken()
	if err != nil {
		t.Fatalf("GetSessionToken failed: %v", err)
	}

	if token == "" {
		t.Error("Expected token")
	}

	t.Logf("Got token: %s", token)
}

func TestMockDEPServer_RemoveProfile(t *testing.T) {
	server := NewMockDEPServer()
	server.AddDevice(Device{
		SerialNumber:  "SRVREMOVE001",
		Model:         "Mac mini",
		ProfileStatus: "assigned",
		ProfileUUID:   "test-profile",
	})

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	client := NewClientWithConfig(ClientConfig{
		BaseURL: ts.URL,
	})

	result, err := client.RemoveProfile([]string{"SRVREMOVE001"})
	if err != nil {
		t.Fatalf("RemoveProfile failed: %v", err)
	}

	if result.Devices["SRVREMOVE001"] != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", result.Devices["SRVREMOVE001"])
	}

	// Verify device state
	device, _ := server.GetDevice("SRVREMOVE001")
	if device.ProfileStatus != "empty" {
		t.Errorf("Expected empty, got %s", device.ProfileStatus)
	}
}

func TestMockDEPServer_CreateTestDevices(t *testing.T) {
	devices := CreateTestDevices()

	if len(devices) == 0 {
		t.Error("Expected test devices")
	}

	for _, d := range devices {
		if d.SerialNumber == "" {
			t.Error("Device missing serial number")
		}
		if d.Model == "" {
			t.Error("Device missing model")
		}
		t.Logf("Test device: %s - %s", d.SerialNumber, d.Model)
	}
}

func TestMockDEPServer_CreateSkipSetupItems(t *testing.T) {
	items := CreateSkipSetupItems()

	if len(items) == 0 {
		t.Error("Expected skip setup items")
	}

	// Check for common items
	found := make(map[string]bool)
	for _, item := range items {
		found[item] = true
	}

	expectedItems := []string{"AppleID", "Siri", "Location", "Diagnostics"}
	for _, expected := range expectedItems {
		if !found[expected] {
			t.Errorf("Expected %s in skip items", expected)
		}
	}
}

// === Integration Test Example ===

func TestDEPFlow_EndToEnd(t *testing.T) {
	// This test demonstrates a complete DEP workflow

	// 1. Start mock server with test devices
	server := NewMockDEPServer()
	server.AddDevices(CreateTestDevices())

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// 2. Create client
	client := NewClientWithConfig(ClientConfig{
		BaseURL: ts.URL,
	})

	// 3. Get session token
	token, err := client.GetSessionToken()
	if err != nil {
		t.Fatalf("GetSessionToken: %v", err)
	}
	t.Logf("Session token: %s", token)

	// 4. Get account details
	account, err := client.GetAccountDetail()
	if err != nil {
		t.Fatalf("GetAccountDetail: %v", err)
	}
	t.Logf("Organization: %s", account.OrgName)

	// 5. Fetch all devices
	devices, cursor, err := client.FetchDevices("")
	if err != nil {
		t.Fatalf("FetchDevices: %v", err)
	}
	t.Logf("Found %d devices (cursor: %s)", len(devices), cursor)

	// 6. Create enrollment profile
	profile := Profile{
		ProfileName:           "E2E Test Profile",
		URL:                   "https://mdm.example.com/enroll",
		IsSupervised:          true,
		IsMandatory:           true,
		IsMDMRemovable:        false,
		AwaitDeviceConfigured: true,
		SkipSetupItems:        CreateSkipSetupItems(),
		Department:            "IT Department",
		SupportEmailAddress:   "support@example.com",
		SupportPhoneNumber:    "+1-555-0100",
	}

	profileUUID, err := client.DefineProfile(profile)
	if err != nil {
		t.Fatalf("DefineProfile: %v", err)
	}
	t.Logf("Created profile: %s", profileUUID)

	// 7. Find unassigned devices
	var unassignedSerials []string
	for _, d := range devices {
		if d.ProfileStatus == "empty" {
			unassignedSerials = append(unassignedSerials, d.SerialNumber)
		}
	}
	t.Logf("Unassigned devices: %d", len(unassignedSerials))

	// 8. Assign profile to unassigned devices
	if len(unassignedSerials) > 0 {
		result, err := client.AssignProfile(profileUUID, unassignedSerials)
		if err != nil {
			t.Fatalf("AssignProfile: %v", err)
		}

		successCount := 0
		for serial, status := range result.Devices {
			if status == "SUCCESS" {
				successCount++
				t.Logf("Assigned %s: %s", serial, status)
			}
		}
		t.Logf("Successfully assigned %d devices", successCount)
	}

	// 9. Verify assignment
	updatedDevices := server.GetAllDevices()
	assignedCount := 0
	for _, d := range updatedDevices {
		if d.ProfileStatus == "assigned" && d.ProfileUUID == profileUUID {
			assignedCount++
		}
	}
	t.Logf("Total devices with new profile: %d", assignedCount)

	// 10. Get profile details
	retrievedProfile, err := client.GetProfile(profileUUID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	t.Logf("Profile name: %s", retrievedProfile.ProfileName)

	t.Log("✅ End-to-end DEP flow completed successfully!")
}
