# Personal Cloud

Single-VM control plane for small apps: **Woodpecker CI** (build/deploy on Git tag), **Caddy** (public HTTPS), **Docker Compose** (runtime). Admin access via **Tailscale**; app APIs on public domains.

## Layout

```
personal-cloud/
├── platform/          Woodpecker + Caddy (docker compose)
├── apps/              Per-app production compose (pull-only images)
├── scripts/           VM bootstrap and manual deploy helpers
└── docs/              Setup guides
```

## Quick start (after VM is installed)

1. Push this repo to `github.com/your-github-username/personal-cloud` (or update clone URLs in your-app `.woodpecker/release.yaml`).
2. On the VM as `deploy`: run `scripts/bootstrap-vm.sh`.
3. Copy `platform/woodpecker/.env.example` → `platform/woodpecker/.env` and fill OAuth values ([docs/woodpecker-github.md](docs/woodpecker-github.md)).
4. Copy `apps/example-app/.env.example` → `apps/example-app/.env` with real secrets.
5. Edit `platform/caddy/Caddyfile` with your API domain and email.
6. `cd /opt/personal-cloud/platform && docker compose up -d`
7. Activate repos in Woodpecker UI (Tailscale → `http://<vm>:8000`).
8. Install `pc` and run `pc ship` from your-app (or tag + legacy Woodpecker workflow)

## Deploy with `pc` CLI (recommended)

From any repo with [`.personal-cloud.yaml`](.personal-cloud.yaml):

```bash
make -C Infra/personal-cloud install-pc
pc validate
pc ship --wait
```

See [docs/pc-cli.md](docs/pc-cli.md). Woodpecker runs [`.woodpecker/ship.yaml`](.woodpecker/ship.yaml) — build, provision compose + Caddy, deploy.

## Manual deploy

```bash
IMAGE_TAG=v0.1.0 ./scripts/deploy-app.sh example-app
```

## Docs

| Doc | Description |
|-----|-------------|
| [docs/vm-setup.md](docs/vm-setup.md) | Debian netinst ISO, sizing, first boot |
| [docs/woodpecker-github.md](docs/woodpecker-github.md) | OAuth app, secrets, repo activation |
| [docs/adding-an-app.md](docs/adding-an-app.md) | Add another service |
| [docs/pc-cli.md](docs/pc-cli.md) | `pc` CLI install and ship |

## Secrets (never commit)

| Location | Contents |
|----------|----------|
| `platform/woodpecker/.env` | Woodpecker OAuth, agent secret |
| `apps/<app>/.env` | App runtime secrets, `IMAGE_TAG` default |
| Woodpecker UI | `ghcr_token` for your-app pipeline |

## Reference app

[your-app server](https://github.com/your-github-username/your-app) — `.personal-cloud.yaml` + `pc ship` (or legacy tag + `.woodpecker/release.yaml`).
