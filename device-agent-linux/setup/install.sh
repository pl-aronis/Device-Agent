#!/bin/bash

# Variables
SERVICE_FILE="/etc/systemd/system/device-agent-linux.service"
BINARY_FILE="/usr/local/bin/device-agent-linux"
BACKUP_DIR="/var/lib/.device-cache"

chmod +x bin/device-agent-linux

# Copy binary to multiple locations
sudo mkdir -p /var/lib/.device-cache
sudo cp bin/device-agent-linux $BINARY_FILE
sudo cp bin/device-agent-linux $BACKUP_DIR/device-agent-linux

# Protect the binary file
sudo chown root:root /usr/local/bin/device-agent-linux
sudo chmod 755 /usr/local/bin/device-agent-linux

# Protect the checksum file too
sudo chown root:root /usr/local/bin/device-agent-linux.sha256
sudo chmod 644 /usr/local/bin/device-agent-linux.sha256
sudo chattr +i /usr/local/bin/device-agent-linux.sha256

# Set up systemd service
sudo cp bin/device-agent-linux.service $SERVICE_FILE
# Reload systemd to recognize the new service
sudo systemctl daemon-reload
# Enable the service to start at boot
systemctl enable device-agent-linux
# Start the service
systemctl start device-agent-linux
# mask the service
sudo systemctl mask device-agent-linux

# Apply immutability
chattr +i $BINARY_FILE
chattr +i $SERVICE_FILE

# Configure firewall
iptables -P OUTPUT DROP
iptables -A OUTPUT -d 192.168.1.11 -j ACCEPT

# Output success message
echo "Device Agent installed successfully."
