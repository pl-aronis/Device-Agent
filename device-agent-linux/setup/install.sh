#!/bin/bash

# Variables
SERVICE_FILE="/etc/systemd/system/device-agent-linux.service"
BINARY_FILE="/usr/local/bin/device-agent-linux"
BACKUP_DIR="/backup"

# Copy binary to multiple locations
mkdir -p $BACKUP_DIR
cp bin/device-agent-linux $BINARY_FILE
cp bin/device-agent-linux $BACKUP_DIR/device-agent-linux

# Set up systemd service
cp bin/device-agent-linux.service $SERVICE_FILE
# Reload systemd to recognize the new service
sudo systemctl daemon-reload
# Enable the service to start at boot
systemctl enable device-agent-linux
# Start the service
systemctl start device-agent-linux

# Apply immutability
chattr +i $BINARY_FILE
chattr +i $SERVICE_FILE

# Configure firewall
iptables -P OUTPUT DROP
iptables -A OUTPUT -d <backend-ip> -j ACCEPT

# Output success message
echo "Device Agent installed successfully."
