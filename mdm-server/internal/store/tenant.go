package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Tenant represents an organization using the MDM service
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`

	// Settings stored as JSON
	SettingsJSON string `json:"-"`

	// APNs configuration
	APNsCertData    []byte    `json:"-"`
	APNsKeyData     []byte    `json:"-"`
	APNsTopic       string    `json:"apns_topic,omitempty"`
	APNsCertExpires time.Time `json:"apns_cert_expires,omitempty"`

	// SCEP CA
	CACertPEM string `json:"-"`
	CAKeyPEM  string `json:"-"`
}

// TenantStore handles tenant database operations
type TenantStore struct {
	db *DB
}

// NewTenantStore creates a new tenant store
func NewTenantStore(db *DB) *TenantStore {
	return &TenantStore{db: db}
}

// Create creates a new tenant
func (s *TenantStore) Create(name, domain string) (*Tenant, error) {
	tenant := &Tenant{
		ID:        uuid.New().String(),
		Name:      name,
		Domain:    domain,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}

	_, err := s.db.Exec(`
		INSERT INTO tenants (id, name, domain, created_at, updated_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tenant.ID, tenant.Name, tenant.Domain, tenant.CreatedAt, tenant.UpdatedAt, tenant.IsActive)

	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return tenant, nil
}

// GetByID retrieves a tenant by ID
func (s *TenantStore) GetByID(id string) (*Tenant, error) {
	tenant := &Tenant{}
	var apnsCertExpires sql.NullTime
	var settingsJSON, apnsTopic, caCertPEM, caKeyPEM sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, domain, created_at, updated_at, is_active,
		       settings_json, apns_cert_data, apns_key_data, apns_topic, apns_cert_expires_at,
		       ca_cert_pem, ca_key_pem
		FROM tenants WHERE id = ?
	`, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.IsActive,
		&settingsJSON, &tenant.APNsCertData, &tenant.APNsKeyData, &apnsTopic, &apnsCertExpires,
		&caCertPEM, &caKeyPEM,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Handle nullable fields
	tenant.SettingsJSON = settingsJSON.String
	tenant.APNsTopic = apnsTopic.String
	tenant.CACertPEM = caCertPEM.String
	tenant.CAKeyPEM = caKeyPEM.String
	if apnsCertExpires.Valid {
		tenant.APNsCertExpires = apnsCertExpires.Time
	}

	return tenant, nil
}

// GetByDomain retrieves a tenant by domain
func (s *TenantStore) GetByDomain(domain string) (*Tenant, error) {
	tenant := &Tenant{}
	var apnsCertExpires sql.NullTime
	var settingsJSON, apnsTopic, caCertPEM, caKeyPEM sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, domain, created_at, updated_at, is_active,
		       settings_json, apns_cert_data, apns_key_data, apns_topic, apns_cert_expires_at,
		       ca_cert_pem, ca_key_pem
		FROM tenants WHERE domain = ?
	`, domain).Scan(
		&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.CreatedAt, &tenant.UpdatedAt, &tenant.IsActive,
		&settingsJSON, &tenant.APNsCertData, &tenant.APNsKeyData, &apnsTopic, &apnsCertExpires,
		&caCertPEM, &caKeyPEM,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant by domain: %w", err)
	}

	// Handle nullable fields
	tenant.SettingsJSON = settingsJSON.String
	tenant.APNsTopic = apnsTopic.String
	tenant.CACertPEM = caCertPEM.String
	tenant.CAKeyPEM = caKeyPEM.String
	if apnsCertExpires.Valid {
		tenant.APNsCertExpires = apnsCertExpires.Time
	}

	return tenant, nil
}

// List returns all tenants
func (s *TenantStore) List() ([]*Tenant, error) {
	rows, err := s.db.Query(`
		SELECT id, name, domain, created_at, updated_at, is_active, apns_topic
		FROM tenants WHERE is_active = 1
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*Tenant
	for rows.Next() {
		tenant := &Tenant{}
		var apnsTopic sql.NullString
		if err := rows.Scan(
			&tenant.ID, &tenant.Name, &tenant.Domain,
			&tenant.CreatedAt, &tenant.UpdatedAt, &tenant.IsActive, &apnsTopic,
		); err != nil {
			return nil, err
		}
		tenant.APNsTopic = apnsTopic.String
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

// Update updates a tenant
func (s *TenantStore) Update(tenant *Tenant) error {
	tenant.UpdatedAt = time.Now()

	_, err := s.db.Exec(`
		UPDATE tenants SET
			name = ?, domain = ?, updated_at = ?, is_active = ?, settings_json = ?
		WHERE id = ?
	`, tenant.Name, tenant.Domain, tenant.UpdatedAt, tenant.IsActive, tenant.SettingsJSON, tenant.ID)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// UpdateAPNs updates APNs certificate for a tenant
func (s *TenantStore) UpdateAPNs(tenantID string, certData, keyData []byte, topic string, expiresAt time.Time) error {
	_, err := s.db.Exec(`
		UPDATE tenants SET
			apns_cert_data = ?, apns_key_data = ?, apns_topic = ?, apns_cert_expires_at = ?, updated_at = ?
		WHERE id = ?
	`, certData, keyData, topic, expiresAt, time.Now(), tenantID)

	if err != nil {
		return fmt.Errorf("failed to update APNs: %w", err)
	}

	return nil
}

// UpdateCA updates SCEP CA for a tenant
func (s *TenantStore) UpdateCA(tenantID string, certPEM, keyPEM string) error {
	_, err := s.db.Exec(`
		UPDATE tenants SET
			ca_cert_pem = ?, ca_key_pem = ?, updated_at = ?
		WHERE id = ?
	`, certPEM, keyPEM, time.Now(), tenantID)

	if err != nil {
		return fmt.Errorf("failed to update CA: %w", err)
	}

	return nil
}

// Delete soft-deletes a tenant
func (s *TenantStore) Delete(id string) error {
	_, err := s.db.Exec(`
		UPDATE tenants SET is_active = 0, updated_at = ? WHERE id = ?
	`, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// GetDeviceCount returns the number of enrolled devices for a tenant
func (s *TenantStore) GetDeviceCount(tenantID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM devices WHERE tenant_id = ? AND is_enrolled = 1
	`, tenantID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get device count: %w", err)
	}

	return count, nil
}
