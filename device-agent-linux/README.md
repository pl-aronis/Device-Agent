## Overview
The Device Agent is a secure device management system designed to enforce compliance policies on Linux and Windows systems. It communicates with a backend server to ensure devices adhere to organizational security standards.

### Key Features
- **Compliance Enforcement**: Enforces policies such as locking the device, restricting network access, and displaying warnings.
- **Tamper Detection**: Monitors service integrity and alerts the backend on anomalies.
- **Fail-Closed Logic**: Automatically locks the device if the backend is unreachable for a specified duration.
- **Agent-Controlled Firewall**: Restricts outbound traffic, allowing only communication with the backend.
- **Persistence**: Ensures the agent survives reboots and resists user removal.
- **Secure Boot and Full Disk Encryption**: Enhances security by preventing unsigned bootloaders and encrypting the disk.

## Components
- **Backend**: Handles policy management and communicates with the agents.
- **Device Agent (Windows)**: Enforces policies on Windows systems.
- **Device Agent (Linux)**: Enforces policies on Linux systems.

## Installation
### Prerequisites
- **Linux**: Ensure `systemd` is installed and running.
- **Windows**: Administrator privileges are required.

### Steps
1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/device-agent.git
   ```
2. Navigate to the appropriate directory:
   - For Linux: `cd device-agent-linux`
   - For Windows: `cd device-agent`
3. Run the installation script (see below).

## Usage
- The agent runs as a background service and communicates with the backend server.
- Logs are stored in `/var/log/device-agent-linux/` (Linux) or `C:\Logs\DeviceAgent\` (Windows).
