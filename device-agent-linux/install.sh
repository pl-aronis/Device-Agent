#!/bin/bash

# Variables
SERVICE_FILE="/etc/systemd/system/device-agent-linux.service"
BINARY_FILE="/usr/local/bin/device-agent-linux"
BACKUP_DIR="/backup"

# Copy binary to multiple locations
mkdir -p $BACKUP_DIR
cp device-agent-linux $BINARY_FILE
cp device-agent-linux $BACKUP_DIR/device-agent-linux

# Set up systemd service
cp device-agent-linux.service $SERVICE_FILE
systemctl enable device-agent-linux
systemctl start device-agent-linux

# Apply immutability
chattr +i $BINARY_FILE
chattr +i $SERVICE_FILE

# Configure firewall
iptables -P OUTPUT DROP
iptables -A OUTPUT -d <backend-ip> -j ACCEPT

# Output success message
echo "Device Agent installed successfully."