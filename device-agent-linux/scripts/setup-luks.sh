#!/bin/bash

# LUKS Encryption Setup Script
# This script sets up LUKS encryption on a specified partition
# WARNING: This will DESTROY all data on the target partition!

set -e

echo "========================================="
echo "LUKS Encryption Setup for Device Agent"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "ERROR: This script must be run as root"
    exit 1
fi

# Check if cryptsetup is installed
if ! command -v cryptsetup &> /dev/null; then
    echo "ERROR: cryptsetup is not installed"
    echo "Install it with: sudo apt install cryptsetup"
    exit 1
fi

# List available partitions
echo "Available partitions:"
lsblk -o NAME,SIZE,TYPE,FSTYPE,MOUNTPOINT
echo ""

# Get partition from user
read -p "Enter the partition to encrypt (e.g., /dev/sda2): " PARTITION

if [ ! -b "$PARTITION" ]; then
    echo "ERROR: $PARTITION is not a valid block device"
    exit 1
fi

# Confirm action
echo ""
echo "WARNING: This will DESTROY ALL DATA on $PARTITION"
echo "Make sure you have backed up all important data!"
echo ""
read -p "Are you absolutely sure you want to continue? (type 'YES' to confirm): " CONFIRM

if [ "$CONFIRM" != "YES" ]; then
    echo "Operation cancelled"
    exit 0
fi

# Get recovery key from backend or generate one
echo ""
echo "Retrieving recovery key from backend..."

BACKEND_HOST=${BACKEND_HOST:-"192.168.1.11:8080"}
DEVICE_INFO_FILE="/var/lib/device-agent-linux/device.json"

if [ -f "$DEVICE_INFO_FILE" ]; then
    RECOVERY_KEY=$(jq -r '.recovery_key' "$DEVICE_INFO_FILE")
    echo "Using recovery key from device registration: $RECOVERY_KEY"
else
    echo "WARNING: Device not registered. Generating temporary recovery key..."
    RECOVERY_KEY=$(openssl rand -hex 16)
    echo "Generated recovery key: $RECOVERY_KEY"
    echo "IMPORTANT: Save this recovery key securely!"
fi

# Generate LUKS key file
echo ""
echo "Generating LUKS key file..."
mkdir -p /var/lib/device-agent-linux
dd if=/dev/urandom of=/var/lib/device-agent-linux/luks.key bs=32 count=1
chmod 400 /var/lib/device-agent-linux/luks.key

# Format partition with LUKS
echo ""
echo "Formatting $PARTITION with LUKS encryption..."
cryptsetup luksFormat "$PARTITION" /var/lib/device-agent-linux/luks.key --batch-mode

# Add recovery key as additional passphrase
echo ""
echo "Adding recovery key as additional passphrase..."
echo "$RECOVERY_KEY" | cryptsetup luksAddKey "$PARTITION" --key-file=/var/lib/device-agent-linux/luks.key

# Open the encrypted partition
echo ""
echo "Opening encrypted partition..."
DEVICE_NAME=$(basename "$PARTITION" | sed 's/\//_/g')
cryptsetup open "$PARTITION" "encrypted_$DEVICE_NAME" --key-file=/var/lib/device-agent-linux/luks.key

# Format with ext4
echo ""
echo "Creating ext4 filesystem..."
mkfs.ext4 "/dev/mapper/encrypted_$DEVICE_NAME"

# Create mount point
MOUNT_POINT="/mnt/encrypted_$DEVICE_NAME"
mkdir -p "$MOUNT_POINT"

# Mount the partition
echo ""
echo "Mounting encrypted partition..."
mount "/dev/mapper/encrypted_$DEVICE_NAME" "$MOUNT_POINT"

# Update /etc/crypttab for auto-unlock at boot
echo ""
echo "Configuring auto-unlock at boot..."
echo "encrypted_$DEVICE_NAME $PARTITION /var/lib/device-agent-linux/luks.key luks" >> /etc/crypttab

# Update /etc/fstab for auto-mount
UUID=$(blkid -s UUID -o value "/dev/mapper/encrypted_$DEVICE_NAME")
echo "UUID=$UUID $MOUNT_POINT ext4 defaults 0 2" >> /etc/fstab

# Display summary
echo ""
echo "========================================="
echo "LUKS Encryption Setup Complete!"
echo "========================================="
echo ""
echo "Partition: $PARTITION"
echo "Mapped Device: /dev/mapper/encrypted_$DEVICE_NAME"
echo "Mount Point: $MOUNT_POINT"
echo "Recovery Key: $RECOVERY_KEY"
echo ""
echo "IMPORTANT: Save the recovery key securely!"
echo "You will need it to unlock the partition if the agent is removed."
echo ""
echo "To manually unlock this partition:"
echo "  echo '$RECOVERY_KEY' | sudo cryptsetup open $PARTITION encrypted_$DEVICE_NAME"
echo ""
echo "To manually lock this partition:"
echo "  sudo umount $MOUNT_POINT"
echo "  sudo cryptsetup close encrypted_$DEVICE_NAME"
echo ""
