package store

import (
	"os"
	"testing"
)

func TestNewSQLiteDB(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-mdm-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Open database
	db, err := NewSQLiteDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify tables exist
	tables := []string{"tenants", "devices", "commands", "profiles", "dep_tokens"}
	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

func TestTenantStore(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-mdm-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewSQLiteDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	db.Migrate()

	store := NewTenantStore(db)

	// Test Create
	tenant, err := store.Create("Test Org", "test.example.com")
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	if tenant.ID == "" {
		t.Error("Tenant ID should not be empty")
	}
	if tenant.Name != "Test Org" {
		t.Errorf("Expected name 'Test Org', got '%s'", tenant.Name)
	}

	// Test GetByID
	retrieved, err := store.GetByID(tenant.ID)
	if err != nil {
		t.Fatalf("Failed to get tenant: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Tenant should not be nil")
	}
	if retrieved.Name != tenant.Name {
		t.Errorf("Name mismatch: expected '%s', got '%s'", tenant.Name, retrieved.Name)
	}

	// Test GetByDomain
	byDomain, err := store.GetByDomain("test.example.com")
	if err != nil {
		t.Fatalf("Failed to get tenant by domain: %v", err)
	}
	if byDomain == nil {
		t.Fatal("Tenant should not be nil")
	}

	// Test List
	tenants, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list tenants: %v", err)
	}
	if len(tenants) != 1 {
		t.Errorf("Expected 1 tenant, got %d", len(tenants))
	}

	// Test Update
	tenant.Name = "Updated Org"
	if err := store.Update(tenant); err != nil {
		t.Fatalf("Failed to update tenant: %v", err)
	}
	retrieved, _ = store.GetByID(tenant.ID)
	if retrieved.Name != "Updated Org" {
		t.Errorf("Update failed: expected 'Updated Org', got '%s'", retrieved.Name)
	}

	// Test Delete
	if err := store.Delete(tenant.ID); err != nil {
		t.Fatalf("Failed to delete tenant: %v", err)
	}
	tenants, _ = store.List()
	if len(tenants) != 0 {
		t.Error("Tenant should be deleted")
	}
}

func TestDeviceStore(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-mdm-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewSQLiteDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	db.Migrate()

	tenantStore := NewTenantStore(db)
	tenant, _ := tenantStore.Create("Test Org", "test.example.com")

	store := NewDeviceStore(db)

	// Test SaveDevice
	token := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	device, err := store.SaveDevice(tenant.ID, "test-udid-123", token, "push-magic-abc")
	if err != nil {
		t.Fatalf("Failed to save device: %v", err)
	}
	if device.ID == "" {
		t.Error("Device ID should not be empty")
	}
	if device.UDID != "test-udid-123" {
		t.Errorf("Expected UDID 'test-udid-123', got '%s'", device.UDID)
	}
	if device.PushMagic != "push-magic-abc" {
		t.Errorf("Expected PushMagic 'push-magic-abc', got '%s'", device.PushMagic)
	}

	// Test GetByUDID
	retrieved, err := store.GetByUDID(tenant.ID, "test-udid-123")
	if err != nil {
		t.Fatalf("Failed to get device: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Device should not be nil")
	}

	// Test GetDevice (legacy)
	legacy, ok := store.GetDevice("test-udid-123")
	if !ok {
		t.Error("GetDevice should return true")
	}
	if legacy == nil {
		t.Error("Device should not be nil")
	}

	// Test ListByTenant
	devices, err := store.ListByTenant(tenant.ID)
	if err != nil {
		t.Fatalf("Failed to list devices: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	// Test UpdateDeviceInfo
	info := map[string]interface{}{
		"DeviceName":   "Test Mac",
		"Model":        "MacBookPro18,1",
		"OSVersion":    "14.0",
		"SerialNumber": "ABC123",
	}
	if err := store.UpdateDeviceInfo(device.ID, info); err != nil {
		t.Fatalf("Failed to update device info: %v", err)
	}
	retrieved, _ = store.GetByUDID(tenant.ID, "test-udid-123")
	if retrieved.DeviceName != "Test Mac" {
		t.Errorf("Expected DeviceName 'Test Mac', got '%s'", retrieved.DeviceName)
	}

	// Test RemoveDevice
	if err := store.RemoveDevice("test-udid-123"); err != nil {
		t.Fatalf("Failed to remove device: %v", err)
	}
	_, ok = store.GetDevice("test-udid-123")
	if ok {
		t.Error("Device should be removed")
	}
}

func TestCommandStore(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test-mdm-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewSQLiteDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	db.Migrate()

	tenantStore := NewTenantStore(db)
	tenant, _ := tenantStore.Create("Test Org", "test.example.com")

	deviceStore := NewDeviceStore(db)
	device, _ := deviceStore.SaveDevice(tenant.ID, "test-udid-456", []byte{1, 2, 3}, "magic")

	store := NewCommandStore(db)

	// Test Enqueue
	cmdUUID, err := store.Enqueue(tenant.ID, device.ID, "DeviceLock", map[string]interface{}{"PIN": "1234"})
	if err != nil {
		t.Fatalf("Failed to enqueue command: %v", err)
	}
	if cmdUUID == "" {
		t.Error("Command UUID should not be empty")
	}

	// Test Next
	cmd, err := store.Next(device.ID)
	if err != nil {
		t.Fatalf("Failed to get next command: %v", err)
	}
	if cmd == nil {
		t.Fatal("Command should not be nil")
	}
	if cmd.RequestType != "DeviceLock" {
		t.Errorf("Expected RequestType 'DeviceLock', got '%s'", cmd.RequestType)
	}

	// Test MarkSent
	if err := store.MarkSent(cmdUUID); err != nil {
		t.Fatalf("Failed to mark sent: %v", err)
	}

	// Test MarkAcknowledged
	if err := store.MarkAcknowledged(cmdUUID); err != nil {
		t.Fatalf("Failed to mark acknowledged: %v", err)
	}

	// Verify status updated
	cmd, _ = store.GetByUUID(cmdUUID)
	if cmd.Status != CommandStatusAcknowledged {
		t.Errorf("Expected status 'acknowledged', got '%s'", cmd.Status)
	}

	// Test ListByDevice
	commands, err := store.ListByDevice(device.ID, 10)
	if err != nil {
		t.Fatalf("Failed to list commands: %v", err)
	}
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}
}
