build the same rental-security agent system on Linux, aligned with what you’re already planning for Windows (persistent agent, device control, tracking, non-payment actions).

> Core Agent Architecture (Linux)

Components
1. Background Daemon (Agent)
    - Runs continuously
    - Auto-starts at boot
    - Cannot be easily stopped by user
2. Persistence Layer
    - Survives reboot
    - Resists user removal
3. Device Control Module
    - Lock screen
    - Network restriction
    - Account restriction
    - Optional data encryption / wipe

> Agent Capabilities

- Communicate with backend
- Execute privileged system commands
- Detect tampering
- Restarts if killed
- Removal triggers reinstall logic (post-rm script)

> Tamper Detection

Detect:
- Service stopped
- Binary modified
- Network disabled
- Time manipulation

Methods:
- File checksum verification
- Heartbeat timeout alerts
- Rootkit-style self-integrity checks
- Alert backend if anomalies detected

> Secure Installation Process (Factory / Vendor Level)

Best Practice
- Install agent before handover
- Lock BIOS/UEFI with password
- Disable boot from USB
- Enable Secure Boot (if possible)

> BIOS / Firmware Level (Advanced but Recommended)

While Linux cannot persist beyond OS reinstall alone:

Add These:
- BIOS password
- Disable external boot
- Secure Boot enabled
- Disk encryption (LUKS) with agent-controlled unlock

This prevents:
- OS reinstallation
- Live USB bypass
- Disk mounting elsewhere

# Persistence Layer Implementation

## Overview
The persistence layer ensures the agent survives reboots and resists user removal. It includes:

1. **Auto-Start at Boot**: The agent is registered as a systemd service.
2. **Tamper Resistance**: The agent monitors its own binary and service status.
3. **Reinstallation Logic**: If removed, the agent reinstalls itself using a post-removal script.

## Implementation Steps

### 1. Systemd Service
Create a systemd service file to auto-start the agent at boot:

```ini
[Unit]
Description=Device Agent for Linux
After=network.target

[Service]
ExecStart=/usr/local/bin/device-agent-linux
Restart=always

[Install]
WantedBy=multi-user.target
```

### 2. Tamper Detection
Use the `tamper` package to monitor the service and binary integrity.

### 3. Post-Removal Script
Create a script to reinstall the agent if removed:

```bash
#!/bin/bash
cp /usr/local/bin/device-agent-linux /tmp/device-agent-linux
systemctl enable device-agent-linux
systemctl start device-agent-linux
```

# BIOS/UEFI Lock

## Steps:
1. **Set a BIOS Password**:
   - Access the BIOS/UEFI settings during boot (usually by pressing `F2`, `DEL`, or `ESC`).
   - Navigate to the Security tab and set a strong password.

2. **Disable USB Boot**:
   - In the Boot Options, disable booting from USB devices.

3. **Disable PXE Boot**:
   - Disable network boot (PXE) in the Boot Options.

4. **Enable Secure Boot**:
   - Ensure Secure Boot is enabled to prevent unsigned bootloaders.

---

# Full Disk Encryption (LUKS)

## Steps:
1. **Install LUKS**:
   - Ensure `cryptsetup` is installed: `sudo apt install cryptsetup`.

2. **Encrypt the Disk**:
   - Backup all data.
   - Use the following command to encrypt the disk:
     ```bash
     sudo cryptsetup luksFormat /dev/sdX
     ```

3. **Open the Encrypted Disk**:
   - ```bash
     sudo cryptsetup open /dev/sdX encrypted_disk
     ```

4. **Format and Mount**:
   - Format the disk: `mkfs.ext4 /dev/mapper/encrypted_disk`
   - Mount it: `mount /dev/mapper/encrypted_disk /mnt`.

5. **Configure Auto-Unlock**:
   - Use the agent to control the unlock key.

---

# BIOS Controls (Reinforced)

1. Prevent external boot.
2. Prevent disk wipe without BIOS access.
3. Ensure the BIOS password is not shared with the user.

# Features
#### 1. Immutability
Use chattr +i to make the agent binary and systemd service file immutable.
#### 2. Systemd Hardening -
Update the systemd service file with:
Restart=always
RestartSec=3
StartLimitIntervalSec=0
#### 3. Package Post-Remove Hooks
Create a .deb post-removal script to reinstall the agent if removed improperly.
#### 4. Multi-Location Persistence
Copy the binary to multiple locations.
Add a hash check in the systemd unit file to verify integrity before starting.
#### 5. Fail-Closed Logic
Implement logic to auto-lock the device if the backend is unreachable for a specified duration.
#### 6. Agent-Controlled Firewall
Use iptables to block all outbound traffic except to the backend.
#### 7. Lock Stored Locally
Cache the lock command locally to enforce it even when offline.
#### 8. BIOS/UEFI Lock
Document steps to:
Set a BIOS password.
Disable USB and PXE boot.
#### 9. Secure Boot Enabled
Ensure Secure Boot is enabled to prevent unsigned bootloaders.
#### 10. Full Disk Encryption (LUKS)
Encrypt the disk using LUKS, ensuring the key is not shared with the user.
#### 11. BIOS Controls (Again)
Reinforce BIOS settings to prevent external boot and disk wipe.

# Points to Remember
⚠️ Reality check: Nothing is 100% unkillable on Linux with root access, but we raise the cost high enough that casual users can’t bypass it.

1️⃣ Make stopping it painful 😏
Auto-restart on kill

Already handled by:

Restart=always

Test:

sudo kill -9 $(pidof device-agent-linux)

👉 It comes back

2️⃣ Detect tampering (inside the agent)

Inside your Go / C / Rust agent, add:

✔️ Self-integrity check

Compute SHA256 of /proc/self/exe

Compare with embedded hash

If mismatch → report + reboot/lock

✔️ systemd detection
getppid() == 1   // must be systemd


If not → agent was launched manually.
 
3️⃣ Prevent simple uninstall

Mask the service:

sudo systemctl mask device-agent-linux

This blocks:

systemctl stop
systemctl disable

(To unmask: systemctl unmask device-agent-linux)

4️⃣ Persistence across rescue attempts (advanced)
If you want serious resistance:
Install agent as:
initramfs hook OR
UEFI service OR
TPM-bound binary
⚠️ This enters enterprise-MDM territory.

🔥 Threats you now STOP
Attack	Result
Kill process	Restarts
Replace binary	Blocked
Overwrite file	Blocked
Stop service	Masked
Modify service	Protected
Reboot	Auto-starts

❗ Threats that still exist (honesty)
Root user + knowledge
Live USB + disk mount
Firmware wipe
These require physical access + expertise.