# Adding an app

## Fast path (`pc`)

1. In your app repo: `pc init` (or add `.personal-cloud.yaml` by hand).
2. On VM: `cp apps/<name>/.env.example apps/<name>/.env` and set secrets.
3. `pc validate` then `pc ship`.

Only **personal-cloud** must be activated in Woodpecker. App repos do not need Woodpecker webhooks.

## Checklist

1. **`.personal-cloud.yaml`** at repo root — see [pc-cli.md](pc-cli.md).
2. **Dockerfile** at `build.dockerfile` path.
3. **VM env file** — `/opt/personal-cloud/apps/<name>/.env` from `.env.example`.
4. **Compose template** — `default` (single container) or `with-postgres`.
5. **DNS** — public `route.host` A record to VM, or private via Tailscale only.
6. **Woodpecker secrets** — `ghcr_token` on personal-cloud repo; `github_clone_token` if repo is private.

## What `pc ship` does

1. Triggers manual pipeline [`.woodpecker/ship.yaml`](../.woodpecker/ship.yaml)
2. Clones your app at the current git SHA
3. Optional `test` script
4. Builds and pushes image (unless `--local`)
5. Renders `apps/<name>/compose.yaml` and `platform/caddy/sites/<name>.caddy`
6. `docker compose up` and `caddy reload`

## Network convention

- `personal-cloud_edge` — Caddy + app containers
- `<app>_internal` — databases (with-postgres template)

Start platform before first ship:

```bash
cd /opt/personal-cloud/platform && docker compose up -d
```

## Legacy: per-repo Woodpecker

You can still use `.woodpecker/release.yaml` in an app repo for tag-only deploys. Prefer `pc ship` for one workflow across all apps.
