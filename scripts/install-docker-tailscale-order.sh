#!/bin/bash
# Install the boot order and heal hook for containers published on the
# Tailscale address. Idempotent. Run as root, or as deploy with sudo.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"

if [[ "$(id -u)" -ne 0 ]]; then
  exec sudo bash "$0" "$@"
fi

install -d /usr/local/sbin
install -m 0755 "$ROOT/docker-tailscale-reattach" /usr/local/sbin/docker-tailscale-reattach
install -m 0755 "$ROOT/docker-tailscale-wait-online" /usr/local/sbin/docker-tailscale-wait-online
install -m 0644 "$ROOT/systemd/tailscale-online.service" /etc/systemd/system/tailscale-online.service
install -m 0644 "$ROOT/systemd/docker-tailscale-reattach.service" /etc/systemd/system/docker-tailscale-reattach.service
install -d /etc/systemd/system/docker.service.d
install -d /etc/systemd/system/tailscaled.service.d
install -m 0644 "$ROOT/systemd/docker.service.d/after-tailscale.conf" /etc/systemd/system/docker.service.d/after-tailscale.conf
install -m 0644 "$ROOT/systemd/tailscaled.service.d/reattach.conf" /etc/systemd/system/tailscaled.service.d/reattach.conf

systemctl daemon-reload
systemctl enable tailscale-online.service docker-tailscale-reattach.service
echo "installed docker/tailscale order"
