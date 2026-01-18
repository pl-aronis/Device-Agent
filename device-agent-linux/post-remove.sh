#!/bin/bash
cp /usr/local/bin/device-agent-linux /tmp/device-agent-linux
systemctl enable device-agent-linux
systemctl start device-agent-linux

# Reapply immutability
chattr +i /usr/local/bin/device-agent-linux
chattr +i /etc/systemd/system/device-agent-linux.service