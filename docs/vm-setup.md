# VM setup (Debian netinst)

## ISO

Download **[Debian 13 (Trixie) netinst](https://www.debian.org/distrib/netinst)** (~600 MB). Use the netinst image, not the full DVD.

## Recommended resources

| Resource | Minimum | Comfortable |
|----------|---------|-------------|
| RAM | 4 GB | 8 GB |
| vCPU | 2 | 4 |
| Disk | 40 GB SSD | 80 GB SSD |

Rust release builds in Woodpecker need several GB RAM during `cargo build`.

## Installer choices

1. Hostname: `personal-cloud` (or your preference).
2. **Software selection:** enable **SSH server** and **standard system utilities** only. Do **not** install a desktop.
3. **Firmware:** enable non-free firmware if the installer offers it (Wi-Fi on mini PCs).
4. Disk: ext4 root; add 2–4 GB swap if RAM ≤ 4 GB.
5. Create user `deploy` with sudo, or create it post-install (see below).

## First boot (as root)

```bash
apt update && apt full-upgrade -y

# If deploy user was not created in the installer:
adduser deploy
usermod -aG sudo deploy
mkdir -p /home/deploy/.ssh
# Paste your public key into /home/deploy/.ssh/authorized_keys
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys
```

### SSH hardening (recommended)

Edit `/etc/ssh/sshd_config`:

- `PermitRootLogin no`
- `PasswordAuthentication no`
- `AllowUsers deploy` (optional)

```bash
systemctl restart ssh
```

## Bootstrap script

Log in as `deploy` and run:

```bash
git clone https://github.com/your-github-username/personal-cloud.git /tmp/personal-cloud
bash /tmp/personal-cloud/scripts/bootstrap-vm.sh
```

This installs Docker CE, Tailscale, clones the repo to `/opt/personal-cloud`, and prints firewall and env-file steps.

## Tailscale (admin access)

After `tailscale up`:

- Use MagicDNS or the machine’s `100.x` address for Woodpecker: `http://<host>:8000`
- Set `WOODPECKER_HOST` in `platform/woodpecker/.env` to that URL (HTTPS if you terminate TLS on Tailscale; HTTP is fine on tailnet-only)

Woodpecker should **not** be exposed on the public internet without extra auth.

## Public DNS (app APIs)

Point your API hostname (e.g. `api.example.com`) A/AAAA records at the VM’s **public** IP. Caddy obtains Let’s Encrypt certificates once ports 80/443 reach the VM.

Edit `platform/caddy/Caddyfile` and set `CADDY_ACME_EMAIL` in `platform/docker-compose.yml` (or via shell env when starting compose).

## Firewall (ufw)

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # omit if SSH is Tailscale-only
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

## Start platform stack

```bash
cp /opt/personal-cloud/platform/woodpecker/.env.example /opt/personal-cloud/platform/woodpecker/.env
# Edit .env — see docs/woodpecker-github.md

cd /opt/personal-cloud/platform
docker compose up -d
docker compose ps
```

Create app env before first deploy:

```bash
cp /opt/personal-cloud/apps/example-app/.env.example /opt/personal-cloud/apps/example-app/.env
chmod 600 /opt/personal-cloud/apps/example-app/.env
# Edit secrets for your app
```

Next: [woodpecker-github.md](woodpecker-github.md).
