#!/usr/bin/env bash
# One-time VM bootstrap for personal-cloud (Debian 13+).
# Run as user 'deploy' with sudo. Idempotent where possible.
set -euo pipefail

INSTALL_ROOT="/opt/personal-cloud"
REPO_URL="${PERSONAL_CLOUD_REPO:-https://github.com/your-github-username/personal-cloud.git}"
DEPLOY_USER="${SUDO_USER:-${USER}}"

log() { printf '==> %s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

if [[ "$(id -u)" -eq 0 ]]; then
  die "Run as user 'deploy' with sudo, not as root."
fi
if ! sudo -v 2>/dev/null; then
  die "This user needs sudo access."
fi

SUDO="sudo"
DEPLOY_USER="${USER}"

run_root() {
  if [[ -n "$SUDO" ]]; then
    $SUDO "$@"
  else
    "$@"
  fi
}

log "Installing prerequisites"
run_root apt-get update -qq
run_root apt-get install -y -qq ca-certificates curl gnupg git ufw

if ! command -v docker >/dev/null 2>&1; then
  log "Installing Docker CE"
  run_root install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/debian/gpg | run_root gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  run_root chmod a+r /etc/apt/keyrings/docker.gpg
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
    $(. /etc/os-release && echo "${VERSION_CODENAME:-bookworm}") stable" \
    | run_root tee /etc/apt/sources.list.d/docker.list >/dev/null
  run_root apt-get update -qq
  run_root apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
  run_root systemctl enable --now docker
else
  log "Docker already installed"
fi

if ! groups "$DEPLOY_USER" | grep -q docker; then
  log "Adding $DEPLOY_USER to docker group"
  run_root usermod -aG docker "$DEPLOY_USER"
  log "Log out and back in (or newgrp docker) for group membership"
fi

if ! command -v tailscale >/dev/null 2>&1; then
  log "Installing Tailscale"
  curl -fsSL https://tailscale.com/install.sh | run_root sh
  log "Run: tailscale up"
else
  log "Tailscale already installed"
fi

log "Cloning personal-cloud to ${INSTALL_ROOT}"
if [[ -d "$INSTALL_ROOT/.git" ]]; then
  log "Already cloned — git pull"
  run_root git -C "$INSTALL_ROOT" pull --ff-only || true
else
  run_root mkdir -p "$(dirname "$INSTALL_ROOT")"
  run_root git clone "$REPO_URL" "$INSTALL_ROOT"
fi
run_root chown -R "${DEPLOY_USER}:${DEPLOY_USER}" "$INSTALL_ROOT"

chmod +x "$INSTALL_ROOT/scripts/"*.sh 2>/dev/null || true

if [[ ! -f "$INSTALL_ROOT/platform/woodpecker/.env" ]]; then
  cp "$INSTALL_ROOT/platform/woodpecker/.env.example" "$INSTALL_ROOT/platform/woodpecker/.env"
  chmod 600 "$INSTALL_ROOT/platform/woodpecker/.env"
  chown "${DEPLOY_USER}:${DEPLOY_USER}" "$INSTALL_ROOT/platform/woodpecker/.env"
  log "Created platform/woodpecker/.env — edit OAuth values before starting Woodpecker"
fi

if [[ ! -f "$INSTALL_ROOT/apps/example-app/.env" ]]; then
  cp "$INSTALL_ROOT/apps/example-app/.env.example" "$INSTALL_ROOT/apps/example-app/.env"
  chmod 600 "$INSTALL_ROOT/apps/example-app/.env"
  chown "${DEPLOY_USER}:${DEPLOY_USER}" "$INSTALL_ROOT/apps/example-app/.env"
  log "Created apps/example-app/.env — set POSTGRES_PASSWORD, JWT_SECRET, etc."
fi

log "Docker daemon.json (DNS + log rotation)"
DAEMON_JSON="/etc/docker/daemon.json"
if [[ ! -f "$DAEMON_JSON" ]]; then
  run_root tee "$DAEMON_JSON" >/dev/null <<'EOF'
{
  "dns": ["1.1.1.1", "8.8.8.8"],
  "log-driver": "json-file",
  "log-opts": { "max-size": "10m", "max-file": "3" }
}
EOF
  run_root systemctl restart docker
fi

log "UFW (optional — review before enabling)"
cat <<'EOF'

Suggested firewall:
  sudo ufw default deny incoming
  sudo ufw default allow outgoing
  sudo ufw allow 22/tcp      # skip if SSH is Tailscale-only
  sudo ufw allow 80/tcp
  sudo ufw allow 443/tcp
  sudo ufw enable

EOF

log "Next steps"
cat <<EOF

1. Edit ${INSTALL_ROOT}/platform/woodpecker/.env (GitHub OAuth) — see docs/woodpecker-github.md
2. Edit ${INSTALL_ROOT}/platform/caddy/Caddyfile (your API domain)
3. Edit ${INSTALL_ROOT}/apps/example-app/.env (secrets)
4. tailscale up   # if not done
5. cd ${INSTALL_ROOT}/platform && docker compose up -d
6. Open Woodpecker at http://<tailscale-ip>:8000 and activate repos
7. Add ghcr_token secret on your-app repo in Woodpecker
8. git tag v0.1.0 && git push origin v0.1.0   # from your-app repo

EOF

log "Bootstrap finished"
