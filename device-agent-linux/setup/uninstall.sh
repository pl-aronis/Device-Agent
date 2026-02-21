#!/bin/bash

# Variables
SERVICE_FILE="/etc/systemd/system/device-agent-linux.service"
BINARY_FILE="/usr/local/bin/device-agent-linux"
BACKUP_DIR="/var/lib/.device-cache"

# Remove binary and backup
rm -f $BINARY_FILE
rm -rf $BACKUP_DIR

# Remove service
rm -f $SERVICE_FILE

# Reload systemd to recognize the removed service
systemctl daemon-reload

# Stop and disable the service
systemctl stop device-agent-linux
systemctl disable device-agent-linux

# Remove immutability
chattr -i $BINARY_FILE
chattr -i $SERVICE_FILE

# Output success message
echo "Device Agent uninstalled successfully."
