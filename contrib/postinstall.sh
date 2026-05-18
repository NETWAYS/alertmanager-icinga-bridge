#!/bin/bash
BRIDGE_USER="alertmanager-icinga-bridge"
BRIDGE_GROUP="alertmanager-icinga-bridge"

# Create group and user
if ! getent group "$BRIDGE_GROUP" >/dev/null 2>&1; then
    groupadd -r "$BRIDGE_GROUP"
fi

if ! getent passwd "$BRIDGE_USER" >/dev/null 2>&1; then
    useradd -m -r -g "$BRIDGE_GROUP" -d /var/lib/bridge -s /sbin/nologin -c "Alertmanager Icinga Bridge user" "$BRIDGE_USER"
    chmod 0755 /var/lib/bridge
    chown "$BRIDGE_USER:$BRIDGE_GROUP" /var/lib/bridge
fi

# Discover systemd unit
systemctl daemon-reload
