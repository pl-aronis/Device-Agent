# Device Agent for Linux

A secure device management system that enforces compliance policies, manages encryption, and provides comprehensive security features for Linux systems.

## 🚀 Quick Start

### Installation

```bash
# 1. Build the agent
cd device-agent-linux
go build -o device-agent-linux

# 2. Install
sudo cp device-agent-linux /usr/local/bin/
sudo chmod +x /usr/local/bin/device-agent-linux

# 3. Configure environment
export BACKEND_IP="192.168.1.11"
export BACKEND_PORT="8080"
export SETUP_BIOS_PASSWORD="true"  # Optional

# 4. Run the agent
sudo /usr/local/bin/device-agent-linux
```

### First Run - Save Your Credentials!

On first run, the agent will:
1. ✅ Register with the backend
2. ✅ Generate and set BIOS password (if enabled)
3. ✅ Log all credentials to multiple locations

**Important**: Save these credentials immediately!

```bash
# View credentials
sudo cat /var/lib/device-agent-linux/credentials.log
```

## 📋 Key Features

### 1. **Automatic Device Registration**
- Registers with backend on first startup
- Receives unique DeviceID and RecoveryKey
- Stores device information locally

### 2. **BIOS/UEFI Password Protection**
- Automatic vendor detection (Dell, HP, Lenovo)
- Vendor-specific tool usage
- Manual fallback for unsupported vendors

### 3. **LUKS Disk Encryption** (Optional)
- Full disk encryption setup
- Recovery key integration
- Auto-unlock configuration

### 4. **Multi-Layer Device Locking**
When a device is locked:
- 🔒 Screen locked
- 🌐 Network disabled
- 💾 LUKS partitions closed
- 👤 User accounts locked
- 🔥 Firewall configured

### 5. **Comprehensive Credentials Logging**
All credentials logged to:
- `/var/lib/device-agent-linux/credentials.json` (JSON)
- `/var/lib/device-agent-linux/credentials.log` (Human-readable)
- Console output during startup

### 6. **Fail-Closed Security**
- Automatically locks device if backend unreachable for 6+ hours
- Caches lock commands for offline enforcement
- Tamper detection and integrity monitoring

## 📁 File Locations

```
/var/lib/device-agent-linux/
├── credentials.json          # All credentials (JSON format)
├── credentials.log           # All credentials (human-readable)
├── device.json              # Device registration info
├── bios.pwd                 # BIOS password
├── bios-manual-setup.txt    # Manual BIOS instructions
├── luks.key                 # LUKS encryption key
└── lock.cache               # Cached lock command

/usr/local/bin/
└── device-agent-linux       # Agent binary
```

## 🔐 Credentials Overview

After registration, you'll receive:

| Credential | Purpose | Location |
|------------|---------|----------|
| **Device ID** | Unique device identifier | credentials.log, device.json |
| **Recovery Key** | Unlock LUKS partitions | credentials.log, backend |
| **BIOS Password** | Access BIOS/UEFI | credentials.log, bios.pwd |
| **Backend URL** | Server connection | credentials.log |

**⚠️ Critical**: Save `credentials.log` to a secure location immediately after first run!

## 🔧 Configuration

### Environment Variables

```bash
# Backend connection (required)
export BACKEND_IP="192.168.1.11"
export BACKEND_PORT="8080"

# Optional features
export SETUP_BIOS_PASSWORD="true"   # Enable BIOS password setup
```

### Vendor-Specific BIOS Tools

The agent automatically detects your system vendor and uses the appropriate tool:

| Vendor | Tool | Command |
|--------|------|---------|
| **Dell** | CCTK | `cctk --setuppwd=PASSWORD` |
| **HP** | hpsetup | `hpsetup -s -a SetupPassword=PASSWORD` |
| **Lenovo** | thinkvantage | `thinkvantage --set-bios-password PASSWORD` |
| **Others** | Manual | Instructions saved to `bios-manual-setup.txt` |

## 🆘 Recovery Procedures

### Get Recovery Key

```bash
# Method 1: From backend API
curl "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=ACTIVE"

# Method 2: From local file (if accessible)
sudo cat /var/lib/device-agent-linux/credentials.log
```

### Unlock LUKS Partition

```bash
# From running system
echo 'RECOVERY_KEY' | sudo cryptsetup open /dev/sda2 unlocked_disk
sudo mount /dev/mapper/unlocked_disk /mnt

# From Live USB
sudo lsblk -f  # Find LUKS partition
echo 'RECOVERY_KEY' | sudo cryptsetup open /dev/sda2 unlocked_root
sudo mount /dev/mapper/unlocked_root /mnt
```

### Get BIOS Password

```bash
# From running system
sudo cat /var/lib/device-agent-linux/bios.pwd

# From Live USB
sudo mount /dev/sda1 /mnt
sudo cat /mnt/var/lib/device-agent-linux/bios.pwd
```

## 🧪 Testing

```bash
# 1. Start backend
cd backend && go run .

# 2. Start agent
cd device-agent-linux
export BACKEND_IP=192.168.1.11
export BACKEND_PORT=8080
export SETUP_BIOS_PASSWORD=true
sudo go run .

# 3. Verify registration
curl http://192.168.1.11:8080/admin/status

# 4. Test locking
curl "http://192.168.1.11:8080/admin/set?id=DEVICE_ID&status=LOCK"

# 5. Check credentials
sudo cat /var/lib/device-agent-linux/credentials.log
```

## 📚 Documentation

- **[IMPLEMENTATION.md](IMPLEMENTATION.md)** - Detailed technical implementation, architecture, and system flows
- **[RECOVERY_GUIDE.md](RECOVERY_GUIDE.md)** - Comprehensive recovery procedures and troubleshooting
- **[QUICK_REFERENCE.md](QUICK_REFERENCE.md)** - Command cheat sheet and quick reference

## 🔄 System Flow

```
Agent Startup
    ↓
Load Configuration (BACKEND_IP, BACKEND_PORT)
    ↓
Generate BIOS Password (if enabled)
    ↓
Detect Vendor → Set BIOS Password
    ↓
Register with Backend
    ├─ Send: MAC, Hostname, OS, BIOS Password
    └─ Receive: DeviceID, Status, RecoveryKey
    ↓
Log All Credentials
    ├─ JSON: credentials.json
    ├─ Log: credentials.log
    └─ Console output
    ↓
Start Heartbeat Loop (every 10 seconds)
    ├─ Send DeviceID
    ├─ Receive action (NONE/WARNING/LOCK)
    └─ Execute action
```

## ⚠️ Important Notes

1. **Save Credentials**: Without the recovery key, encrypted data is **unrecoverable**
2. **BIOS Password**: May require manufacturer support to reset
3. **Test Recovery**: Verify recovery procedures work **before** production deployment
4. **Backup**: Keep `credentials.log` and backend `data/devices.json` backed up
5. **LUKS Setup**: Destroys all data on the partition - backup first!

## 🛠️ Troubleshooting

### Agent won't start
```bash
# Check logs
sudo journalctl -u device-agent-linux -f

# Verify backend connectivity
curl http://BACKEND_IP:BACKEND_PORT/admin/status
```

### BIOS password setup failed
```bash
# Check vendor detection
sudo cat /sys/class/dmi/id/sys_vendor

# View manual instructions
sudo cat /var/lib/device-agent-linux/bios-manual-setup.txt
```

### Can't unlock LUKS partition
```bash
# Verify it's a LUKS partition
sudo cryptsetup luksDump /dev/sda2

# Get recovery key from backend
curl "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=ACTIVE"
```

## 📊 Backend API

### Register Device
```bash
POST /api/register
{
  "mac_id": "00:11:22:33:44:55",
  "location": "hostname",
  "os_details": "Linux 5.15.0",
  "bios_pass": "optional"
}
```

### Send Heartbeat
```bash
POST /api/heartbeat
{
  "device_id": "abc123"
}
```

### Admin Commands
```bash
# Lock device
GET /admin/set?id=DEVICE_ID&status=LOCK

# Unlock device (returns recovery key)
GET /admin/set?id=DEVICE_ID&status=ACTIVE

# List all devices
GET /admin/status
```

## 🎯 Security Features

- ✅ Automatic device registration
- ✅ Vendor-specific BIOS password setting
- ✅ LUKS disk encryption support
- ✅ Multi-layer device locking
- ✅ Fail-closed security (locks if offline)
- ✅ Tamper detection
- ✅ Firewall configuration
- ✅ Comprehensive credentials logging
- ✅ Recovery procedures

## 📝 License

[Your License Here]

## 🤝 Contributing

[Contributing guidelines]

---

**Version**: 1.0  
**Last Updated**: 2026-02-06

For detailed technical documentation, see [IMPLEMENTATION.md](IMPLEMENTATION.md)
