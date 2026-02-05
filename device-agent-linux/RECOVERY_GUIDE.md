# Device Recovery and Decryption Guide

This guide explains how to retrieve passwords and decrypt LUKS-encrypted disks when the device is locked by the Device Agent.

## Table of Contents
1. [Recovery Key Overview](#recovery-key-overview)
2. [Retrieving the Recovery Key](#retrieving-the-recovery-key)
3. [Unlocking LUKS Encrypted Disks](#unlocking-luks-encrypted-disks)
4. [BIOS Password Retrieval](#bios-password-retrieval)
5. [Emergency Recovery Procedures](#emergency-recovery-procedures)

---

## Recovery Key Overview

When a device is registered with the Device Agent backend, a **Recovery Key** is automatically generated. This key serves multiple purposes:

- **LUKS Disk Decryption**: Used as an additional passphrase for LUKS encrypted partitions
- **Device Unlock**: Can be used to unlock the device through the admin interface
- **Emergency Access**: Provides a way to recover data even if the device is locked

### Where is the Recovery Key Stored?

1. **Backend Server**: Stored in `data/devices.json` on the backend server
2. **Local Device**: Stored in `/var/lib/device-agent-linux/device.json` on the device
3. **Agent Logs**: Displayed during initial device registration (check systemd logs)

---

## Retrieving the Recovery Key

### Method 1: From Backend Admin Interface

The recovery key is returned when transitioning a device from LOCK to ACTIVE status:

```bash
# Unlock device and retrieve recovery key
curl "http://BACKEND_HOST:8080/admin/set?id=DEVICE_ID&status=ACTIVE"
```

**Response:**
```json
{
  "device_id": "abc123def456",
  "status": "ACTIVE",
  "recovery_key": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "message": "device unlocked - recovery key displayed"
}
```

### Method 2: From Backend Storage File

If you have access to the backend server:

```bash
# View all devices and their recovery keys
cat /path/to/backend/data/devices.json
```

**Example output:**
```json
[
  {
    "id": "abc123def456",
    "status": "LOCK",
    "recovery_key": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "mac_id": "00:11:22:33:44:55",
    "location": "office-laptop-01",
    "os_details": "Linux 5.15.0-generic",
    "last_seen": "2026-02-05T12:30:00Z"
  }
]
```

### Method 3: From Device Local Storage

If you have physical access to the device and can boot from a Live USB:

```bash
# Mount the root partition
sudo mount /dev/sda1 /mnt

# View device info (requires root access)
sudo cat /mnt/var/lib/device-agent-linux/device.json
```

### Method 4: From Agent Startup Logs

The recovery key is logged when the device first registers:

```bash
# View systemd logs
sudo journalctl -u device-agent-linux | grep "Recovery Key"
```

**Example output:**
```
[STARTUP] Recovery Key: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6 (SAVE THIS SECURELY!)
```

---

## Unlocking LUKS Encrypted Disks

### Scenario 1: Device is Running but Locked

If the device is locked but still running, you can unlock LUKS partitions using the recovery key:

```bash
# List LUKS encrypted partitions
sudo lsblk -o NAME,TYPE,FSTYPE

# Unlock a LUKS partition
echo 'YOUR_RECOVERY_KEY' | sudo cryptsetup open /dev/sdX unlocked_disk

# Mount the unlocked partition
sudo mount /dev/mapper/unlocked_disk /mnt

# Access your data
ls /mnt
```

### Scenario 2: Boot from Live USB

If the device won't boot or is completely locked:

1. **Boot from a Live USB** (Ubuntu, Fedora, etc.)

2. **Identify the LUKS partition:**
   ```bash
   sudo lsblk -f
   # Look for partitions with TYPE=crypto_LUKS
   ```

3. **Unlock the LUKS partition:**
   ```bash
   # Use the recovery key from the backend
   echo 'a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6' | sudo cryptsetup open /dev/sda2 unlocked_root
   ```

4. **Mount the unlocked partition:**
   ```bash
   sudo mount /dev/mapper/unlocked_root /mnt
   ```

5. **Access your data:**
   ```bash
   cd /mnt/home
   # Copy files to external drive or cloud storage
   ```

### Scenario 3: Using the Agent's Built-in Unlock Function

If you can run the agent with elevated privileges:

```go
package main

import (
    "device-agent-linux/enforcement"
    "log"
)

func main() {
    recoveryKey := "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
    partition := "/dev/sda2"
    
    err := enforcement.UnlockLUKSPartition(partition, recoveryKey)
    if err != nil {
        log.Fatalf("Failed to unlock: %v", err)
    }
    
    log.Println("Partition unlocked successfully!")
}
```

### Verifying LUKS Unlock

After unlocking, verify the partition is accessible:

```bash
# Check if the mapper device exists
ls -l /dev/mapper/unlocked_*

# Check filesystem
sudo fsck /dev/mapper/unlocked_root

# Mount and verify
sudo mount /dev/mapper/unlocked_root /mnt
df -h /mnt
```

---

## BIOS Password Retrieval

### Where is the BIOS Password Stored?

The BIOS password is stored in `/var/lib/device-agent-linux/bios.pwd` on the device.

### Retrieving the BIOS Password

**Method 1: From Running System (with root access)**
```bash
sudo cat /var/lib/device-agent-linux/bios.pwd
```

**Method 2: From Live USB**
```bash
# Mount the root partition
sudo mount /dev/sda1 /mnt

# Read the BIOS password file
sudo cat /mnt/var/lib/device-agent-linux/bios.pwd
```

**Method 3: Using the Agent's Built-in Function**
```go
package main

import (
    "device-agent-linux/enforcement"
    "log"
)

func main() {
    password, err := enforcement.GetBIOSPassword()
    if err != nil {
        log.Fatalf("Failed to get BIOS password: %v", err)
    }
    
    log.Printf("BIOS Password: %s", password)
}
```

### Using the BIOS Password

1. **Reboot the device**
2. **Press F2, DEL, or ESC** (depending on manufacturer) to enter BIOS
3. **Enter the retrieved password** when prompted
4. **Modify BIOS settings** as needed:
   - Re-enable USB boot
   - Change boot order
   - Disable Secure Boot (if needed)
   - Remove BIOS password

---

## Emergency Recovery Procedures

### Complete Recovery Workflow

If a device is completely locked and you need to recover data:

1. **Retrieve Recovery Key from Backend:**
   ```bash
   curl "http://BACKEND_HOST:8080/admin/set?id=DEVICE_ID&status=ACTIVE" | jq -r '.recovery_key'
   ```

2. **Boot from Live USB:**
   - Create a bootable USB with Ubuntu/Fedora
   - Boot the locked device from the USB

3. **Unlock LUKS Partition:**
   ```bash
   # Identify encrypted partition
   sudo lsblk -f
   
   # Unlock with recovery key
   echo 'RECOVERY_KEY' | sudo cryptsetup open /dev/sda2 unlocked_root
   ```

4. **Mount and Access Data:**
   ```bash
   sudo mount /dev/mapper/unlocked_root /mnt
   cd /mnt
   ```

5. **Copy Important Data:**
   ```bash
   # Copy to external drive
   sudo cp -r /mnt/home/user/important_files /media/external_drive/
   ```

6. **Optional: Remove Agent:**
   ```bash
   # Remove agent binary
   sudo rm /mnt/usr/local/bin/device-agent-linux
   
   # Disable service
   sudo rm /mnt/etc/systemd/system/device-agent-linux.service
   ```

7. **Reboot into Normal System:**
   ```bash
   sudo umount /mnt
   sudo cryptsetup close unlocked_root
   sudo reboot
   ```

### Resetting a Locked Device

To completely reset a locked device:

```bash
# 1. Boot from Live USB
# 2. Unlock LUKS partition (as shown above)
# 3. Remove agent files
sudo mount /dev/mapper/unlocked_root /mnt
sudo rm -rf /mnt/var/lib/device-agent-linux
sudo rm /mnt/usr/local/bin/device-agent-linux
sudo rm /mnt/etc/systemd/system/device-agent-linux.service

# 4. Re-enable user accounts
sudo chroot /mnt
usermod -U username  # Unlock user account
exit

# 5. Unmount and reboot
sudo umount /mnt
sudo cryptsetup close unlocked_root
sudo reboot
```

---

## Security Best Practices

### For Administrators

1. **Store Recovery Keys Securely:**
   - Use a password manager
   - Keep encrypted backups
   - Maintain offline copies in secure locations

2. **Document Device Information:**
   - Device ID
   - Recovery Key
   - BIOS Password
   - MAC Address
   - Last known location

3. **Regular Backups:**
   - Back up `data/devices.json` from the backend
   - Store in multiple secure locations

### For Users

1. **Never Share Recovery Keys** with unauthorized personnel
2. **Report Lost Devices Immediately** to administrators
3. **Keep Device Information Updated** in the backend system

---

## Troubleshooting

### Issue: "cryptsetup: device already in use"

**Solution:**
```bash
# Close existing mapping
sudo cryptsetup close unlocked_root

# Try again
echo 'RECOVERY_KEY' | sudo cryptsetup open /dev/sda2 unlocked_root
```

### Issue: "Invalid passphrase"

**Possible causes:**
- Wrong recovery key
- Partition not LUKS encrypted
- Corrupted LUKS header

**Solution:**
```bash
# Verify it's a LUKS partition
sudo cryptsetup luksDump /dev/sda2

# Try with the LUKS key file instead
sudo cryptsetup open /dev/sda2 unlocked_root --key-file=/path/to/luks.key
```

### Issue: "Cannot access BIOS password file"

**Solution:**
```bash
# Boot from Live USB
sudo mount /dev/sda1 /mnt

# Check if file exists
sudo find /mnt -name "bios.pwd"

# If not found, BIOS password may not have been set
# Try default BIOS password reset methods (manufacturer-specific)
```

---

## API Reference

### Get Recovery Key via API

**Endpoint:** `GET /admin/set?id=DEVICE_ID&status=ACTIVE`

**Response:**
```json
{
  "device_id": "string",
  "status": "ACTIVE",
  "recovery_key": "string",
  "message": "device unlocked - recovery key displayed"
}
```

### Get All Devices

**Endpoint:** `GET /admin/status`

**Response:**
```json
[
  {
    "id": "string",
    "status": "string",
    "recovery_key": "string",
    "mac_id": "string",
    "location": "string",
    "os_details": "string",
    "last_seen": "timestamp"
  }
]
```

---

## Summary

- **Recovery Key**: Stored in backend, used for LUKS decryption and device unlock
- **BIOS Password**: Stored locally in `/var/lib/device-agent-linux/bios.pwd`
- **LUKS Unlock**: Use `cryptsetup open` with recovery key
- **Emergency Access**: Boot from Live USB and unlock partitions
- **Security**: Always store recovery keys and passwords securely

For additional support, contact your system administrator or refer to the main IMPLEMENTATION.md file.
