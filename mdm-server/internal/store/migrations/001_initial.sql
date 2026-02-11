-- Migration 001: Initial schema for multi-tenant MDM server
-- Creates core tables for tenants, users, devices, certificates, commands, and profiles

-- Tenants table: Organizations using the MDM service
CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    domain TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Settings stored as JSON
    settings_json TEXT DEFAULT '{}',
    
    -- APNs configuration (stored encrypted in production)
    apns_cert_data BLOB,
    apns_key_data BLOB,
    apns_topic TEXT,
    apns_cert_expires_at DATETIME,
    
    -- SCEP CA for this tenant
    ca_cert_pem TEXT,
    ca_key_pem TEXT,
    
    -- Status
    is_active INTEGER DEFAULT 1
);

-- Users table: Admin users for the MDM dashboard
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT,
    role TEXT DEFAULT 'admin', -- 'superadmin', 'admin', 'readonly'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME,
    is_active INTEGER DEFAULT 1,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(tenant_id, email)
);

-- Devices table: Enrolled Apple devices
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    udid TEXT NOT NULL,
    serial_number TEXT,
    
    -- Push notification credentials
    push_token TEXT,
    push_magic TEXT,
    
    -- Device information (from DeviceInformation command)
    device_name TEXT,
    model TEXT,
    model_name TEXT,
    os_version TEXT,
    build_version TEXT,
    product_name TEXT,
    
    -- Enrollment status
    enrolled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME,
    enrollment_type TEXT DEFAULT 'manual', -- 'manual', 'dep', 'user'
    is_supervised INTEGER DEFAULT 0,
    
    -- DEP information
    dep_profile_uuid TEXT,
    
    -- Status
    is_enrolled INTEGER DEFAULT 1,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(tenant_id, udid)
);

-- Device certificates issued via SCEP
CREATE TABLE IF NOT EXISTS device_certificates (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    device_id TEXT,
    
    -- Certificate data
    cert_pem TEXT NOT NULL,
    subject TEXT,
    serial_number TEXT,
    
    -- Validity
    issued_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL
);

-- MDM Commands queue and history
CREATE TABLE IF NOT EXISTS commands (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    
    -- Command details
    command_uuid TEXT NOT NULL UNIQUE,
    request_type TEXT NOT NULL,
    payload_json TEXT,
    
    -- Status tracking
    status TEXT DEFAULT 'pending', -- 'pending', 'sent', 'acknowledged', 'error', 'notnow'
    error_chain_json TEXT,
    
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME,
    responded_at DATETIME,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Configuration profiles (stored for reference/reinstall)
CREATE TABLE IF NOT EXISTS profiles (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    
    -- Profile identification
    identifier TEXT NOT NULL,
    uuid TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    
    -- Profile content
    profile_data BLOB NOT NULL, -- The actual .mobileconfig plist
    
    -- Metadata
    version INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(tenant_id, identifier)
);

-- DEP/ABM tokens for tenant ABM integration
CREATE TABLE IF NOT EXISTS dep_tokens (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL UNIQUE,
    
    -- OAuth credentials (encrypted in production)
    consumer_key TEXT NOT NULL,
    consumer_secret TEXT NOT NULL,
    access_token TEXT NOT NULL,
    access_secret TEXT NOT NULL,
    
    -- Server information from ABM
    server_name TEXT,
    server_uuid TEXT,
    org_name TEXT,
    org_email TEXT,
    org_phone TEXT,
    org_address TEXT,
    
    -- Sync tracking
    last_sync_at DATETIME,
    sync_cursor TEXT,
    
    -- Token validity
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- DEP device records synced from ABM
CREATE TABLE IF NOT EXISTS dep_devices (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    
    -- Device info from ABM
    model TEXT,
    description TEXT,
    color TEXT,
    device_family TEXT,
    os TEXT,
    
    -- Assignment status
    profile_status TEXT DEFAULT 'empty', -- 'empty', 'assigned', 'pushed'
    profile_uuid TEXT,
    assigned_at DATETIME,
    
    -- Link to enrolled device (if enrolled)
    device_id TEXT,
    
    -- Sync metadata
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL,
    UNIQUE(tenant_id, serial_number)
);

-- Audit log for compliance
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT,
    user_id TEXT,
    device_id TEXT,
    
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    details_json TEXT,
    
    ip_address TEXT,
    user_agent TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE SET NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_devices_tenant ON devices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_devices_udid ON devices(udid);
CREATE INDEX IF NOT EXISTS idx_devices_serial ON devices(serial_number);
CREATE INDEX IF NOT EXISTS idx_commands_device ON commands(device_id);
CREATE INDEX IF NOT EXISTS idx_commands_status ON commands(status);
CREATE INDEX IF NOT EXISTS idx_dep_devices_serial ON dep_devices(serial_number);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);
