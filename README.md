# Device Agent

This project implements a secure device management agent for Windows. It enforces compliance policies triggered by a remote backend, featuring advanced locking mechanisms and user alerts.

## Key Features

*   **Heartbeat Monitoring**: Polls the backend every 10 seconds for compliance status.
*   **Compliance Warning**: Displays a full-screen, always-on-top red warning window for non-compliant devices (`action: WARNING`).
*   **Secure Lockdown**:
    *   **BitLocker Recovery**: If `action: LOCK` is received, the agent forces the device into BitLocker Recovery mode.
    *   **Safety Handshake**: The agent **only** locks only if it successfully generates a new recovery key and receives a `200 OK` acknowledgement from the backend.
    *   **Master Backdoor**: Attempts to inject a static Master Password (`MasterKey@123`) before locking as a secondary entry method.
    *   **Immediate Reboot**: Reboots the system immediately after enforcement.

## Architecture

*   **Agent**: Runs as a Windows service (simulated loop), polling `http://localhost:8080/api/heartbeat`.
*   **Backend**: A mock Go server that controls the policy state and logs received recovery keys.

## Usage (Testing)

**⚠️ DANGER**: Run this inside a Virtual Machine. The Lock command is destructive.

### 1. Start the Backend
```powershell
go run backend/server.go
# Backend listens on :8080
```

### 2. Start the Agent
```powershell
BACKEND_HOST="192.168.12.82:8080" go run main.go
```

### 3. Control Policy
Use the admin API to switch modes:

*   **Normal**: `curl http://localhost:8080/admin/set?action=NONE`
*   **Warning**: `curl http://localhost:8080/admin/set?action=WARNING` (Shows full screen alert)
*   **Lock**: `curl http://localhost:8080/admin/set?action=LOCK` (Reboots into BitLocker Recovery)

## Recovery

If the device is locked:
1.  Check the **Backend Console Logs** for the randomized **48-digit Recovery Key**.
2.  Alternatively, try the **Master Password**: `MasterKey@123` (press Esc at the BitLocker screen to switch to password mode).
