chattr -i /usr/local/bin/device-agent-linux
chattr -i /etc/systemd/system/device-agent-linux.service
chattr -i /var/lib/.device-cache/.d
systemctl stop device-agent-linux
rm /usr/local/bin/device-agent-linux
rm /etc/systemd/system/device-agent-linux.service
rm /var/lib/.device-cache/.d
