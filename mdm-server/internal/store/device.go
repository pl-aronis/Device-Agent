package store

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Device represents a managed Apple device
type Device struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	UDID         string `json:"udid"`
	SerialNumber string `json:"serial_number,omitempty"`

	// Push notification credentials
	PushToken string `json:"push_token"`
	PushMagic string `json:"push_magic"`

	// Device information
	DeviceName   string `json:"device_name,omitempty"`
	Model        string `json:"model,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	OSVersion    string `json:"os_version,omitempty"`
	BuildVersion string `json:"build_version,omitempty"`
	ProductName  string `json:"product_name,omitempty"`

	// Enrollment status
	EnrolledAt     time.Time `json:"enrolled_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
	EnrollmentType string    `json:"enrollment_type"` // 'manual', 'dep', 'user'
	IsSupervised   bool      `json:"is_supervised"`
	IsEnrolled     bool      `json:"is_enrolled"`

	// DEP information
	DEPProfileUUID string `json:"dep_profile_uuid,omitempty"`
}

// DeviceStore handles device database operations
type DeviceStore struct {
	db *DB
}

// NewDeviceStore creates a new device store with database backend
func NewDeviceStore(db *DB) *DeviceStore {
	return &DeviceStore{db: db}
}

// SaveDevice saves or updates a device from MDM enrollment
func (s *DeviceStore) SaveDevice(tenantID, udid string, tokenBytes []byte, pushMagic string) (*Device, error) {
	token := hex.EncodeToString(tokenBytes)
	now := time.Now()

	// Check if device exists
	existing, err := s.GetByUDID(tenantID, udid)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Update existing device
		_, err = s.db.Exec(`
			UPDATE devices SET
				push_token = ?, push_magic = ?, last_seen_at = ?, is_enrolled = 1
			WHERE id = ?
		`, token, pushMagic, now, existing.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to update device: %w", err)
		}

		existing.PushToken = token
		existing.PushMagic = pushMagic
		existing.LastSeenAt = now
		existing.IsEnrolled = true
		return existing, nil
	}

	// Create new device
	device := &Device{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		UDID:           udid,
		PushToken:      token,
		PushMagic:      pushMagic,
		EnrolledAt:     now,
		LastSeenAt:     now,
		EnrollmentType: "manual",
		IsEnrolled:     true,
	}

	_, err = s.db.Exec(`
		INSERT INTO devices (id, tenant_id, udid, push_token, push_magic, enrolled_at, last_seen_at, enrollment_type, is_enrolled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.TenantID, device.UDID, device.PushToken, device.PushMagic,
		device.EnrolledAt, device.LastSeenAt, device.EnrollmentType, device.IsEnrolled)

	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}

// GetByID retrieves a device by ID
func (s *DeviceStore) GetByID(id string) (*Device, error) {
	device := &Device{}
	var lastSeenAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, tenant_id, udid, serial_number, push_token, push_magic,
		       device_name, model, model_name, os_version, build_version, product_name,
		       enrolled_at, last_seen_at, enrollment_type, is_supervised, is_enrolled, dep_profile_uuid
		FROM devices WHERE id = ?
	`, id).Scan(
		&device.ID, &device.TenantID, &device.UDID, &device.SerialNumber,
		&device.PushToken, &device.PushMagic,
		&device.DeviceName, &device.Model, &device.ModelName, &device.OSVersion,
		&device.BuildVersion, &device.ProductName,
		&device.EnrolledAt, &lastSeenAt, &device.EnrollmentType, &device.IsSupervised,
		&device.IsEnrolled, &device.DEPProfileUUID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	if lastSeenAt.Valid {
		device.LastSeenAt = lastSeenAt.Time
	}

	return device, nil
}

// GetByUDID retrieves a device by UDID within a tenant
func (s *DeviceStore) GetByUDID(tenantID, udid string) (*Device, error) {
	device := &Device{}
	var lastSeenAt sql.NullTime
	var serialNumber, deviceName, model, modelName, osVersion, buildVersion, productName, depProfileUUID sql.NullString

	err := s.db.QueryRow(`
		SELECT id, tenant_id, udid, serial_number, push_token, push_magic,
		       device_name, model, model_name, os_version, build_version, product_name,
		       enrolled_at, last_seen_at, enrollment_type, is_supervised, is_enrolled, dep_profile_uuid
		FROM devices WHERE tenant_id = ? AND udid = ?
	`, tenantID, udid).Scan(
		&device.ID, &device.TenantID, &device.UDID, &serialNumber,
		&device.PushToken, &device.PushMagic,
		&deviceName, &model, &modelName, &osVersion, &buildVersion, &productName,
		&device.EnrolledAt, &lastSeenAt, &device.EnrollmentType, &device.IsSupervised,
		&device.IsEnrolled, &depProfileUUID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device by UDID: %w", err)
	}

	// Handle nullable fields
	device.SerialNumber = serialNumber.String
	device.DeviceName = deviceName.String
	device.Model = model.String
	device.ModelName = modelName.String
	device.OSVersion = osVersion.String
	device.BuildVersion = buildVersion.String
	device.ProductName = productName.String
	device.DEPProfileUUID = depProfileUUID.String

	if lastSeenAt.Valid {
		device.LastSeenAt = lastSeenAt.Time
	}

	return device, nil
}

// GetDevice retrieves a device by UDID (legacy compatibility - finds across all tenants)
func (s *DeviceStore) GetDevice(udid string) (*Device, bool) {
	device := &Device{}
	var lastSeenAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, tenant_id, udid, push_token, push_magic, enrolled_at, last_seen_at, is_enrolled
		FROM devices WHERE udid = ? AND is_enrolled = 1 LIMIT 1
	`, udid).Scan(
		&device.ID, &device.TenantID, &device.UDID,
		&device.PushToken, &device.PushMagic,
		&device.EnrolledAt, &lastSeenAt, &device.IsEnrolled,
	)

	if err != nil {
		return nil, false
	}

	if lastSeenAt.Valid {
		device.LastSeenAt = lastSeenAt.Time
	}

	return device, true
}

// ListByTenant returns all devices for a tenant
func (s *DeviceStore) ListByTenant(tenantID string) ([]*Device, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, udid, serial_number, push_token, push_magic,
		       device_name, model, model_name, os_version,
		       enrolled_at, last_seen_at, enrollment_type, is_supervised, is_enrolled
		FROM devices WHERE tenant_id = ? AND is_enrolled = 1
		ORDER BY last_seen_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		device := &Device{}
		var lastSeenAt sql.NullTime
		var serialNumber, deviceName, model, modelName, osVersion sql.NullString

		if err := rows.Scan(
			&device.ID, &device.TenantID, &device.UDID, &serialNumber,
			&device.PushToken, &device.PushMagic,
			&deviceName, &model, &modelName, &osVersion,
			&device.EnrolledAt, &lastSeenAt, &device.EnrollmentType,
			&device.IsSupervised, &device.IsEnrolled,
		); err != nil {
			return nil, err
		}

		device.SerialNumber = serialNumber.String
		device.DeviceName = deviceName.String
		device.Model = model.String
		device.ModelName = modelName.String
		device.OSVersion = osVersion.String

		if lastSeenAt.Valid {
			device.LastSeenAt = lastSeenAt.Time
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// ListDevices returns all enrolled devices (legacy compatibility)
func (s *DeviceStore) ListDevices() []*Device {
	devices, _ := s.ListAll()
	return devices
}

// ListAll returns all enrolled devices across all tenants
func (s *DeviceStore) ListAll() ([]*Device, error) {
	rows, err := s.db.Query(`
		SELECT id, tenant_id, udid, push_token, push_magic, enrolled_at, last_seen_at, is_enrolled
		FROM devices WHERE is_enrolled = 1
		ORDER BY last_seen_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list all devices: %w", err)
	}
	defer rows.Close()

	var devices []*Device
	for rows.Next() {
		device := &Device{}
		var lastSeenAt sql.NullTime

		if err := rows.Scan(
			&device.ID, &device.TenantID, &device.UDID,
			&device.PushToken, &device.PushMagic,
			&device.EnrolledAt, &lastSeenAt, &device.IsEnrolled,
		); err != nil {
			return nil, err
		}

		if lastSeenAt.Valid {
			device.LastSeenAt = lastSeenAt.Time
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// UpdateDeviceInfo updates device information from DeviceInformation command
func (s *DeviceStore) UpdateDeviceInfo(id string, info map[string]interface{}) error {
	// Extract fields from info map
	deviceName, _ := info["DeviceName"].(string)
	model, _ := info["Model"].(string)
	modelName, _ := info["ModelName"].(string)
	osVersion, _ := info["OSVersion"].(string)
	buildVersion, _ := info["BuildVersion"].(string)
	productName, _ := info["ProductName"].(string)
	serialNumber, _ := info["SerialNumber"].(string)

	_, err := s.db.Exec(`
		UPDATE devices SET
			device_name = ?, model = ?, model_name = ?, os_version = ?,
			build_version = ?, product_name = ?, serial_number = ?, last_seen_at = ?
		WHERE id = ?
	`, deviceName, model, modelName, osVersion, buildVersion, productName, serialNumber, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to update device info: %w", err)
	}

	return nil
}

// UpdateLastSeen updates the last seen timestamp
func (s *DeviceStore) UpdateLastSeen(id string) error {
	_, err := s.db.Exec(`UPDATE devices SET last_seen_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// RemoveDevice unenrolls a device
func (s *DeviceStore) RemoveDevice(udid string) error {
	_, err := s.db.Exec(`
		UPDATE devices SET is_enrolled = 0, last_seen_at = ? WHERE udid = ?
	`, time.Now(), udid)

	if err != nil {
		return fmt.Errorf("failed to remove device: %w", err)
	}

	return nil
}

// Delete permanently deletes a device
func (s *DeviceStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}
	return nil
}
