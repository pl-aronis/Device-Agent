# Device Agent - Quick Reference Card

## 🚀 Installation & Setup

```bash
# 1. Build
cd device-agent-linux && go build -o device-agent-linux

# 2. Install
sudo cp device-agent-linux /usr/local/bin/
sudo chmod +x /usr/local/bin/device-agent-linux

# 3. Start (auto-registers)
export BACKEND_HOST="192.168.1.11:8080"
sudo /usr/local/bin/device-agent-linux

# 4. Save recovery key from logs!
sudo journalctl -u device-agent-linux | grep "Recovery Key"
```

## 🔐 LUKS Encryption Setup

```bash
# Run setup script
sudo ./scripts/setup-luks.sh

# Manual setup
sudo cryptsetup luksFormat /dev/sda2 /var/lib/device-agent-linux/luks.key
echo 'RECOVERY_KEY' | sudo cryptsetup luksAddKey /dev/sda2 --key-file=/var/lib/device-agent-linux/luks.key
```

## 🔒 BIOS Password Setup

```bash
# Run setup script
sudo ./scripts/setup-bios.sh

# Retrieve password
sudo cat /var/lib/device-agent-linux/bios.pwd
```

## 📡 Backend API

```bash
# Register device
curl -X POST http://BACKEND:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"mac_id":"00:11:22:33:44:55","location":"hostname","os_details":"Linux"}'

# Heartbeat
curl -X POST http://BACKEND:8080/api/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"device_id":"abc123"}'

# Lock device
curl "http://BACKEND:8080/admin/set?id=abc123&status=LOCK"

# Unlock device (returns recovery key)
curl "http://BACKEND:8080/admin/set?id=abc123&status=ACTIVE"

# List all devices
curl http://BACKEND:8080/admin/status
```

## 🔓 Recovery Procedures

### Get Recovery Key

```bash
# Method 1: Backend API
curl "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=ACTIVE" | jq -r '.recovery_key'

# Method 2: Backend file
cat /path/to/backend/data/devices.json | jq -r '.[] | select(.id=="DEVICE_ID") | .recovery_key'

# Method 3: Device file (if accessible)
sudo cat /var/lib/device-agent-linux/device.json | jq -r '.recovery_key'

# Method 4: Agent logs
sudo journalctl -u device-agent-linux | grep "Recovery Key"
```

### Unlock LUKS Disk

```bash
# From running system
echo 'RECOVERY_KEY' | sudo cryptsetup open /dev/sda2 unlocked_disk
sudo mount /dev/mapper/unlocked_disk /mnt

# From Live USB
sudo lsblk -f  # Find LUKS partition
echo 'RECOVERY_KEY' | sudo cryptsetup open /dev/sda2 unlocked_root
sudo mount /dev/mapper/unlocked_root /mnt
cd /mnt  # Access data
```

### Get BIOS Password

```bash
# From running system
sudo cat /var/lib/device-agent-linux/bios.pwd

# From Live USB
sudo mount /dev/sda1 /mnt
sudo cat /mnt/var/lib/device-agent-linux/bios.pwd
```

## 📂 File Locations

```
/var/lib/device-agent-linux/
├── device.json              # DeviceID, Status, RecoveryKey
├── luks.key                 # LUKS encryption key (32 bytes)
├── bios.pwd                 # BIOS password (32-char hex)
├── lock.cache               # Cached lock command
└── bios-setup-instructions.txt  # Manual BIOS setup guide

/usr/local/bin/
└── device-agent-linux       # Agent binary

/etc/systemd/system/
└── device-agent-linux.service  # Systemd service file
```

## 🔍 Troubleshooting

### Registration Failed
```bash
# Check backend connectivity
curl http://BACKEND_HOST:8080/ping

# Check logs
sudo journalctl -u device-agent-linux -f

# Manual registration test
curl -X POST http://BACKEND:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"mac_id":"test","location":"test","os_details":"test"}'
```

### LUKS Unlock Failed
```bash
# Verify LUKS partition
sudo cryptsetup luksDump /dev/sda2

# Check if already unlocked
sudo lsblk -o NAME,TYPE | grep crypt

# Try with key file
sudo cryptsetup open /dev/sda2 unlocked --key-file=/var/lib/device-agent-linux/luks.key
```

### Device Not Locking
```bash
# Check agent status
sudo systemctl status device-agent-linux

# Check heartbeat
sudo journalctl -u device-agent-linux | grep "Heartbeat"

# Manual lock test
curl "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=LOCK"
```

## 🧪 Testing Commands

```bash
# Test registration
sudo systemctl start device-agent-linux
sudo journalctl -u device-agent-linux | grep "STARTUP"
curl http://BACKEND:8080/admin/status

# Test LUKS
sudo ./scripts/setup-luks.sh
sudo cryptsetup luksDump /dev/sda2
echo 'KEY' | sudo cryptsetup open /dev/sda2 test && sudo cryptsetup close test

# Test lock
curl "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=LOCK"
# Verify: screen locked, network off, LUKS closed, users locked

# Test recovery
RECOVERY_KEY=$(curl -s "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=ACTIVE" | jq -r '.recovery_key')
echo "$RECOVERY_KEY" | sudo cryptsetup open /dev/sda2 unlocked
```

## 📊 Status Checks

```bash
# Agent status
sudo systemctl status device-agent-linux

# Device registration
sudo cat /var/lib/device-agent-linux/device.json | jq

# LUKS status
sudo lsblk -o NAME,TYPE,FSTYPE,MOUNTPOINT | grep crypt

# Network status
nmcli networking connectivity

# User lock status
sudo passwd -S username  # L = locked, P = password set
```

## 🔐 Security Checklist

- [ ] Agent registered with backend
- [ ] Recovery key saved securely
- [ ] LUKS encryption set up (optional)
- [ ] BIOS password set (optional)
- [ ] BIOS password saved securely
- [ ] Backend data/devices.json backed up
- [ ] Recovery procedures tested
- [ ] Systemd service enabled
- [ ] Firewall configured (optional)
- [ ] Tamper detection active

## ⚠️ Critical Warnings

1. **LUKS Setup DESTROYS DATA** - Backup before running setup-luks.sh
2. **Save Recovery Keys** - Without them, encrypted data is UNRECOVERABLE
3. **Save BIOS Passwords** - Reset may require manufacturer support
4. **Test Recovery** - Verify recovery works BEFORE production
5. **Protect Backend** - Secure data/devices.json (contains all keys)

## 📚 Documentation

- **IMPLEMENTATION.md** - Full architecture and implementation details
- **RECOVERY_GUIDE.md** - Comprehensive recovery procedures
- **README_FEATURES.md** - Quick-start guide and feature overview
- **SUMMARY.md** - Implementation summary and changes
- **VISUAL_OVERVIEW.md** - Visual diagrams and flows

## 🆘 Emergency Recovery

```bash
# 1. Boot from Live USB
# 2. Get recovery key from backend
RECOVERY_KEY=$(curl -s "http://BACKEND:8080/admin/set?id=DEVICE_ID&status=ACTIVE" | jq -r '.recovery_key')

# 3. Unlock LUKS
echo "$RECOVERY_KEY" | sudo cryptsetup open /dev/sda2 unlocked_root

# 4. Mount and access
sudo mount /dev/mapper/unlocked_root /mnt
cd /mnt

# 5. Copy important data
sudo cp -r /mnt/home/user/important_files /media/external_drive/

# 6. Optional: Remove agent
sudo rm /mnt/usr/local/bin/device-agent-linux
sudo rm /mnt/etc/systemd/system/device-agent-linux.service
sudo rm -rf /mnt/var/lib/device-agent-linux

# 7. Cleanup and reboot
sudo umount /mnt
sudo cryptsetup close unlocked_root
sudo reboot
```

## 🎯 Quick Reference

| Task | Command |
|------|---------|
| Start agent | `sudo systemctl start device-agent-linux` |
| View logs | `sudo journalctl -u device-agent-linux -f` |
| Get device ID | `sudo cat /var/lib/device-agent-linux/device.json \| jq -r '.device_id'` |
| Get recovery key | `curl "http://BACKEND:8080/admin/set?id=ID&status=ACTIVE" \| jq -r '.recovery_key'` |
| Lock device | `curl "http://BACKEND:8080/admin/set?id=ID&status=LOCK"` |
| Unlock LUKS | `echo 'KEY' \| sudo cryptsetup open /dev/sda2 unlocked` |
| Get BIOS pwd | `sudo cat /var/lib/device-agent-linux/bios.pwd` |
| List devices | `curl http://BACKEND:8080/admin/status` |

---

**Version:** 1.0  
**Last Updated:** 2026-02-05  
**For detailed documentation, see IMPLEMENTATION.md and RECOVERY_GUIDE.md**
