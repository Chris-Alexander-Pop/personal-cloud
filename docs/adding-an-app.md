# Adding an app

## Fast path (`pc`)

1. In your app repo: `pc init` (or add `.personal-cloud.yaml` by hand).
2. On VM: `cp apps/<name>/.env.example apps/<name>/.env` and set secrets.
3. `pc validate` then `pc ship` (from a clone), or `pc ship <app>` to read the manifest from GitHub.

Only **personal-cloud** must be activated in Woodpecker. App repos do not need Woodpecker webhooks.

## Checklist

1. **`.personal-cloud.yaml`** at repo root — see [pc-cli.md](pc-cli.md). After the next [sync-ship-apps](github-actions.md) run it appears in **Actions → ship**.
2. **Dockerfile** at `build.dockerfile` path.
3. **VM env file** — `/opt/personal-cloud/apps/<name>/.env` from `.env.example`.
4. **Compose template** — see table below.
5. **DNS** — public `route.host` A record to VM, or private via Tailscale only.
6. **Woodpecker secrets** — `ghcr_token` on personal-cloud repo; `github_clone_token` if repo is private.

## Compose templates

| Template | Use when |
|----------|----------|
| `default` | Single container on the edge network |
| `with-postgres` | App + Postgres on an internal network |
| `with-media-volume` | Read-only media bind mount |
| `with-data-volume` | Writable `/data` bind mount (SQLite, JSON state) |
| `host-network` | App needs the host network (LAN UDP / multicast) |
| `with-data-volume-host` | Host network **and** `/data` (e.g. IoT controllers) |
| `with-minio` | App + MinIO (S3) on an internal network; MinIO ports published on the VM |
| `with-music-stack` | Ingest + Navidrome + dufs + phone-edge Caddy (8040–8042). Profile `acquire` adds Lidarr/slskd/Soularr (UIs 8043/8045) |
| `with-git-server` | Forgejo: `/data` volume, HTTP+SSH published on `GIT_BIND` (8050/2222), Caddy for git smart HTTP |

Host-network templates bind the app on the VM. Caddy proxies to `host.docker.internal:<port>` (platform compose adds that host mapping).

## What `pc ship` does

1. Triggers manual pipeline [`.woodpecker/ship.yaml`](../.woodpecker/ship.yaml) — same file whether you run `pc` from a clone, `pc ship owner/repo`, or GitHub Actions
2. Clones your app from GitHub.com at the chosen git SHA
3. Optional `test` script
4. Builds and pushes image (unless `--local`)
5. Renders `apps/<name>/compose.yaml` and `platform/caddy/sites/<name>.caddy`
6. `docker compose up` and `caddy reload`

## Network convention

- `personal-cloud_edge` — Caddy + bridge-network app containers
- `<app>_internal` — databases (with-postgres template)
- Host-network apps — listen on the VM; not attached to `edge`

Start platform before first ship:

```bash
cd /opt/personal-cloud/platform && docker compose up -d
```

## Legacy: per-repo Woodpecker

You can still use `.woodpecker/release.yaml` in an app repo for tag-only deploys. Prefer `pc ship` for one workflow across all apps.
