# Device Agent Backend API Documentation

## Overview

The backend server manages device agents through a registration and heartbeat system. Devices register once, then continuously send heartbeats to receive their current action status. Administrators can change device status (ACTIVE or LOCK) via admin endpoints.

---

## Core Endpoints

### 1. Device Registration
**Endpoint:** `POST /api/register`

Register a new device agent with the backend, providing device metadata. The server generates a unique device ID if one is not provided and creates a recovery key.

**Request:**
```json
{
  "device_id": "optional-device-id",
  "mac_id": "00:1A:2B:3C:4D:5E",
  "location": "Building A, Room 201",
  "os_details": "Windows 10 Pro, Build 19045"
}
```
- `device_id` (optional): If provided, the server uses this as the device ID. If omitted, a random 16-character hex ID is generated.
- `mac_id` (required): MAC address of the primary network interface
- `location` (required): Physical or logical location of the device
- `os_details` (required): Operating system and version information

**Response:** (HTTP 200)
```json
{
  "device_id": "abc123def456",
  "status": "ACTIVE",
  "recovery_key": "f1e2d3c4b5a69798"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid JSON payload or missing required fields
- `500 Internal Server Error`: Storage error

---

### 2. Heartbeat
**Endpoint:** `POST /api/heartbeat`

Device agents send a heartbeat at regular intervals to report they are alive and retrieve the current action status from the server.

**Request:**
```json
{
  "device_id": "abc123def456"
}
```

**Response:** (HTTP 200)
```json
{
  "action": "ACTIVE"
}
```

- `action`: Current status of the device as set by administrators. Values: `ACTIVE` or `LOCK`.

**Error Responses:**
- `400 Bad Request`: Missing or invalid device_id
- `404 Not Found`: Device ID not found in the system

---

## Admin Endpoints

### 3. Set Device Status
**Endpoint:** `GET /admin/set?status=<STATUS>&id=<DEVICE_ID>`

Change the status of a single device or all devices at once. When transitioning a device from LOCK to ACTIVE, the recovery key is returned to the UI for display to the user.

**Query Parameters:**
- `status` (required): New status for the device(s). Values: `ACTIVE` or `LOCK`.
- `id` (optional): Target device ID. If omitted, the status change is applied to **all devices**.

**Response:** (HTTP 200) - Single Device
```json
{
  "device_id": "abc123def456",
  "status": "ACTIVE",
  "recovery_key": "f1e2d3c4b5a69798",
  "message": "device unlocked - recovery key displayed"
}
```

**Response:** (HTTP 200) - All Devices
```json
{
  "status": "ACTIVE",
  "message": "updated all devices"
}
```

**Error Responses:**
- `400 Bad Request`: Missing `status` parameter
- `404 Not Found`: Device ID not found (only when `id` is specified)

**Examples:**
- Set single device to LOCK: `GET /admin/set?status=LOCK&id=abc123def456`
- Set all devices to ACTIVE: `GET /admin/set?status=ACTIVE`

---

### 4. List All Devices
**Endpoint:** `GET /admin/status`

Retrieve the status and metadata of all registered devices.

**Response:** (HTTP 200)
```json
[
  {
    "id": "abc123def456",
    "status": "ACTIVE",
    "last_seen": "2026-01-28T10:30:45.123456Z"
  },
  {
    "id": "xyz789uvw012",
    "status": "LOCK",
    "last_seen": "2026-01-28T10:25:30.654321Z"
  }
]
```

---

## Health Check

### 5. Ping
**Endpoint:** `GET /ping`

Simple health check to verify the server is running and reachable.

**Response:** (HTTP 200)
```
pong
```

---

## Agent-Backend Contract

### Data Model

#### Device Schema
```go
type Device struct {
    ID           string    `json:"id"`         // Unique device identifier
    Status       string    `json:"status"`     // ACTIVE or LOCK
    RecoveryKey  string    `json:"recovery_key"` // 32-char hex key for unlocking
    MacID        string    `json:"mac_id"`     // MAC address of primary interface
    Location     string    `json:"location"`   // Physical/logical device location
    OSDetails    string    `json:"os_details"` // OS version and build information
    LastSeen     time.Time `json:"last_seen"`  // Timestamp of last heartbeat
}
```

### Device Lifecycle

1. **Registration Phase**
   - Device agent starts
   - Sends `POST /api/register` with device metadata (mac_id, location, os_details)
   - Server generates recovery key and assigns unique device_id
   - Receives `device_id`, initial status (`ACTIVE`), and `recovery_key`
   - Stores device_id locally for future requests

2. **Active Monitoring Phase**
   - Device sends `POST /api/heartbeat` with its device_id every N seconds (configurable)
   - Server responds with current action status (`ACTIVE` or `LOCK`)
   - Server updates device's `last_seen` timestamp
   - Device takes appropriate action based on response:
     - `ACTIVE`: Normal operation, no action needed
     - `LOCK`: Execute lock mechanism on the device

3. **Admin Control**
   - Administrator uses UI/CLI to call `GET /admin/set?status=LOCK&id=<device_id>` to lock a device
   - Status change is persisted in JSON database
   - Next heartbeat from device receives the new `LOCK` status
   - Device executes lock mechanism
   
4. **Unlock & Recovery**
   - Administrator calls `GET /admin/set?status=ACTIVE&id=<device_id>` to unlock the device
   - Server detects transition from `LOCK` → `ACTIVE`
   - Recovery key is returned in the response body
   - UI displays the recovery key to the administrator
   - User receives the recovery key and can unlock their device

### Status Values

- **ACTIVE**: Device operates normally
- **LOCK**: Device should enforce lock mechanism (display warning, restrict access, etc.)

### Error Handling

- **Device Not Registered**: Heartbeat with unknown device_id returns 404
- **Missing Fields**: Incomplete JSON payloads return 400
- **Storage Failure**: Server errors return 500

---

## Database

Devices are persisted in a JSON file at `data/devices.json`:

```json
[
  {
    "id": "abc123def456",
    "status": "ACTIVE",
    "recovery_key": "f1e2d3c4b5a69798",
    "mac_id": "00:1A:2B:3C:4D:5E",
    "location": "Building A, Room 201",
    "os_details": "Windows 10 Pro, Build 19045",
    "last_seen": "2026-01-28T10:30:45.123456Z"
  }
]
```

Future database migrations can swap this JSON storage for a real relational or NoSQL database without changing the API contract.
