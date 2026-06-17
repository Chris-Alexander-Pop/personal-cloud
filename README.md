# Personal Cloud

Single-VM control plane for small apps: **Woodpecker CI** (build/deploy), **Caddy** (public HTTPS), **Docker Compose** (runtime). Admin access via **Tailscale**; app APIs on public or tailnet-only domains.

Fork this repo, point it at your GitHub org/user, and run your own VM — nothing here connects to anyone else's server.

## Layout

```
personal-cloud/
├── cli/               `pc` deploy CLI
├── platform/          Woodpecker + Caddy (docker compose)
├── apps/              Per-app env examples (compose generated on ship)
├── templates/         Compose templates
├── scripts/           VM bootstrap and deploy helpers
└── docs/              Setup guides
```

## Quick start

1. Fork or clone to your GitHub account (`your-github-username/personal-cloud`).
2. On a Debian VM as user `deploy`: run `scripts/bootstrap-vm.sh`.
3. Copy `platform/woodpecker/.env.example` → `platform/woodpecker/.env` and fill OAuth values ([docs/woodpecker-github.md](docs/woodpecker-github.md)).
4. `cd /opt/personal-cloud/platform && docker compose up -d`
5. Activate **personal-cloud** in Woodpecker (Tailscale → `http://<vm>:8000`).
6. On your laptop: copy [cli/config.yaml.example](cli/config.yaml.example) → `~/.config/pc/config.yaml`, run `make install-pc`.
7. In an app repo: add `.personal-cloud.yaml` ([example](.personal-cloud.yaml.example)), then `pc validate` and `pc ship --wait`.

## Deploy with `pc` CLI

From any repo with a `.personal-cloud.yaml` manifest:

```bash
git clone https://github.com/your-github-username/personal-cloud.git ~/personal-cloud
make -C ~/personal-cloud install-pc
pc validate
pc ship --wait
```

See [docs/pc-cli.md](docs/pc-cli.md). Woodpecker runs [`.woodpecker/ship.yaml`](.woodpecker/ship.yaml) — clone, test, build, provision compose + Caddy, deploy.

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
| [SECURITY.md](SECURITY.md) | Deployment security notes |

## Secrets (never commit)

| Location | Contents |
|----------|----------|
| `platform/woodpecker/.env` | Woodpecker OAuth, agent secret |
| `apps/<app>/.env` | App runtime secrets on the VM |
| `~/.config/pc/config.yaml` | Woodpecker token, GHCR PAT, your VM settings |
| Woodpecker UI | `ghcr_token`, optional `github_clone_token` |

## Example app

[`apps/example-app/`](apps/example-app/) is a minimal reference for env files and CI validation. Replace it with your own apps as you deploy.
