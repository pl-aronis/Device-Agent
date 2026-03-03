# Device Backend

## Overview

This backend manages device agents.

It handles:
- Registration
- Re-authentication
- Heartbeats
- Lock state management
- Recovery key storage
- Admin dashboard APIs for frontend

The database is a JSON file located at:

data/devices.json

---

# Agent → Backend Flow

1. Agent starts
2. Agent calls POST /register
3. Backend stores device and returns agent_id

4. Agent polls POST /heartbeat
   - Sends mac_id + location
   - Backend returns should_lock

5. If should_lock = true
   - Agent performs BitLocker lock
   - Agent calls POST /lock-success
     - Sends recovery key + recovery ID

6. Backend stores recovery data

7. After reboot:
   - Agent calls POST /re-authenticate
   - Backend returns previous agent_id + recovery_id

---

# Agent API Contract

## POST /register

Request:
{
  "mac_id": "string",
  "os": "windows",
  "arch": "amd64",
  "latitude": 12.34,
  "longitude": 56.78
}

Response:
{
  "agent_id": "uuid"
}

---

## POST /re-authenticate

Request:
{
  "mac_id": "string"
}

Response:
{
  "agent_id": "uuid",
  "recovery_id": "string"
}

---

## POST /heartbeat

Request:
{
  "mac_id": "string",
  "lat": 12.34,
  "lon": 56.78
}

Response:
{
  "should_lock": true | false
}

---

## POST /lock-success

Request:
{
  "mac_id": "string",
  "key": "recovery password",
  "id": "protector id"
}

No response body.

---

## POST /lock-failure

Request:
{
  "mac_id": "string"
}

No response body.

---

# Admin API Contract (Frontend)

## GET /admin/status

Response:
```json
[
  {
    "id": "agent-uuid",
    "status": "ACTIVE | LOCK",
    "location": "12.34000, 56.78000",
    "mac_id": "AA-BB-CC-DD-EE-FF",
    "os_details": "windows/amd64",
    "last_seen": "2026-03-03T11:43:30Z",
    "recovery_key": "optional",
    "recovery_protector_id": "optional",
    "is_locked": false,
    "should_lock": false
  }
]
```

## GET or POST /admin/set

Request:
```json
{
  "id": "agent-uuid",
  "status": "LOCK | ACTIVE"
}
```

Response:
```json
{
  "id": "agent-uuid",
  "status": "LOCK | ACTIVE",
  "recovery_key": "optional",
  "recovery_protector_id": "optional",
  "should_lock": true,
  "is_locked": false
}
```

---

# How to Run

go run cmd/main.go

Server runs on:

http://localhost:8080
