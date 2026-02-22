#!/bin/bash

# Variables
SERVICE_FILE="/etc/systemd/system/device-agent-linux.service"
BINARY_FILE="/usr/local/bin/device-agent-linux"
BACKUP_DIR="/var/lib/.device-cache"

# Reload systemd to recognize the removed service
systemctl daemon-reload

# Signal to the agent that this is an intentional stop (no device lock)
mkdir -p /var/lib/device-agent-linux && touch /var/lib/device-agent-linux/no-lock.flag

# Stop and disable the service
systemctl stop device-agent-linux
systemctl disable device-agent-linux

# Remove immutability
chattr -i $BINARY_FILE
chattr -i $SERVICE_FILE
chattr -i /usr/local/bin/device-agent-linux.sha256

# Remove binary and backup
rm -f $BINARY_FILE
rm -rf $BACKUP_DIR
rm -f /usr/local/bin/device-agent-linux.sha256 

# Remove service
rm -f $SERVICE_FILE

# Remove cache file if exists
rm -f /var/lib/device-agent-linux/lock.cache

# enable outbound traffic
iptables -P OUTPUT ACCEPT

# Output success message
echo "Device Agent uninstalled successfully."
