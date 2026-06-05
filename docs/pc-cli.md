# `pc` CLI

Deploy any repo with a [`.personal-cloud.yaml`](../.personal-cloud.yaml) manifest via Woodpecker on your personal-cloud VM. No per-app Woodpecker setup — only the **personal-cloud** repo must be activated.

## Install

```bash
cd Infra/personal-cloud
make install-pc   # installs ~/.local/bin/pc
```

Ensure `~/.local/bin` is on your `PATH`.

## Config

```bash
mkdir -p ~/.config/pc
cp Infra/personal-cloud/cli/config.yaml.example ~/.config/pc/config.yaml
chmod 600 ~/.config/pc/config.yaml
```

| Key | Description |
|-----|-------------|
| `woodpecker.url` | Woodpecker UI URL (Tailscale), e.g. `http://100.x.x.x:8000` |
| `woodpecker.token` | Personal access token from Woodpecker → User settings |
| `personal_cloud.owner` / `repo` | GitHub repo that runs `.woodpecker/ship.yaml` |
| `ghcr.token` | GitHub PAT with `write:packages` (for `pc ship --local`) |
| `defaults.tailnet_base` | Suffix for private routes, e.g. `example.ts.net` |

## Manifest (per app repo)

Create with `pc init` or copy from [your-app example](../../../path/to/your-app/.personal-cloud.yaml).

```yaml
name: my-app
image: ghcr.io/your-github-username/my-app
build:
  context: .
  dockerfile: Dockerfile
service:
  container: my-app
  port: 8080
route:
  exposure: public    # or private
  host: api.example.com   # required if public
compose:
  template: default   # or with-postgres
test: optional shell command run before build
```

## Commands

```bash
pc init              # wizard: create .personal-cloud.yaml
pc validate          # manifest + Woodpecker connectivity
pc ship              # remote build on VM, deploy, Caddy route
pc ship --local      # build/push from this machine, CI deploy-only
pc ship --private    # Tailscale-only route (tls internal)
pc ship --public     # force public ACME route
pc ship --tag v1.0.0 # image tag (default: dev-<git-sha>)
pc ship --wait       # block until pipeline finishes
pc status            # latest personal-cloud pipeline state
pc logs              # print Woodpecker pipeline URL
```

## First deploy (your-app example)

1. VM: platform stack running, `apps/example-app/.env` created from `.env.example`
2. Woodpecker: activate `your-github-username/personal-cloud`, secrets `ghcr_token`, optional `github_clone_token`
3. Laptop: `~/.config/pc/config.yaml` filled in
4. From your-app repo:

```bash
pc validate
pc ship --wait
```

5. Public: point DNS at VM, `curl https://api.example.com/health`  
   Private: open `https://example-app.example.ts.net/health` on Tailscale

## Public vs private routes

| `exposure` | Caddy snippet | DNS |
|------------|---------------|-----|
| `public` | Let's Encrypt on `route.host` | Public A/AAAA to VM |
| `private` | `tls internal` on `<name>.<tailnet_base>` | Tailscale MagicDNS only |

Woodpecker writes `platform/caddy/sites/<app>.caddy` and reloads Caddy.

## Woodpecker secrets

| Secret | Used for |
|--------|----------|
| `ghcr_token` | docker login / pull / push |
| `github_clone_token` | clone private app repos (optional) |

## One-off local build

```bash
pc ship --local --tag dev-$(git rev-parse --short HEAD) --private --wait
```

Builds with `docker buildx --push` on your machine, then triggers ship with `DO_BUILD=false`.

## Troubleshooting

- **repo not found in Woodpecker** — activate `personal-cloud` in the UI
- **missing variable** — upgrade `pc` and ensure `.personal-cloud.yaml` is complete
- **health check failed** — app may still be starting; check `docker compose -f /opt/personal-cloud/apps/<name>/compose.yaml ps`
- **compose template** — use `with-postgres` for DB-backed apps like example-app
