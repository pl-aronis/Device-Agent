#!/bin/bash

# Variables
SERVICE_FILE="/etc/systemd/system/device-agent-linux.service"
BINARY_FILE="/usr/local/bin/device-agent-linux"
BACKUP_DIR="/var/lib/.device-cache"

chmod +x bin/device-agent-linux

# Create the agent data directory before the service starts.
# The agent writes device.json, luks.key, etc. here at first run.
# Creating it here (as root) avoids a read-only-filesystem error when
# the agent tries to create it itself after chattr/firewall are applied.
sudo mkdir -p /var/lib/device-agent-linux
sudo chmod 700 /var/lib/device-agent-linux
sudo chown root:root /var/lib/device-agent-linux

# Copy binary to multiple locations
sudo mkdir -p $BACKUP_DIR
sudo cp bin/device-agent-linux $BINARY_FILE
sudo cp bin/device-agent-linux $BACKUP_DIR/device-agent-linux

# Protect the binary
sudo chown root:root $BINARY_FILE
sudo chmod 755 $BINARY_FILE

sha256sum $BINARY_FILE > /usr/local/bin/device-agent-linux.sha256 && sha256sum -c /usr/local/bin/device-agent-linux.sha256

# Protect the checksum file
sudo chown root:root /usr/local/bin/device-agent-linux.sha256
sudo chmod 644 /usr/local/bin/device-agent-linux.sha256
sudo chattr +i /usr/local/bin/device-agent-linux.sha256

# Set up systemd service
sudo cp bin/device-agent-linux.service $SERVICE_FILE
# Reload systemd to recognize the new service
sudo systemctl daemon-reload
# Enable the service to start at boot
systemctl enable device-agent-linux

# Apply immutability to binary and service file
chattr +i $BINARY_FILE
chattr +i $SERVICE_FILE

# Start the service
systemctl start device-agent-linux

# Output success message
echo "Device Agent installed successfully."
