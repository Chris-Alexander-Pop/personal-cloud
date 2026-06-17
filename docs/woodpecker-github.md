# Woodpecker + GitHub

## 1. GitHub OAuth application

1. GitHub → **Settings** → **Developer settings** → **OAuth Apps** → **New OAuth App**
2. **Application name:** `Woodpecker (personal-cloud)`
3. **Homepage URL:** your Woodpecker URL (e.g. `http://100.x.x.x:8000` or `https://ci.<machine>.<tailnet>.ts.net`)
4. **Authorization callback URL:** `{WOODPECKER_HOST}/authorize`  
   Must match `WOODPECKER_HOST` in `platform/woodpecker/.env` exactly (no trailing slash).

Copy **Client ID** and **Client secret** into `platform/woodpecker/.env`:

```bash
WOODPECKER_GITHUB_CLIENT=...
WOODPECKER_GITHUB_SECRET=...
WOODPECKER_AGENT_SECRET=$(openssl rand -hex 32)   # if not set yet
WOODPECKER_ADMIN=your-github-username
WOODPECKER_HOST=http://100.x.x.x:8000             # your Tailscale-reachable URL
WOODPECKER_OPEN=true
WOODPECKER_GITHUB=true
```

Restart Woodpecker after changes:

```bash
cd /opt/personal-cloud/platform && docker compose up -d
```

## 2. Log in and activate repositories

1. Open `WOODPECKER_HOST` in a browser (via Tailscale).
2. Sign in with GitHub.
3. **Repositories** → **New repository** → enable **your-github-username/personal-cloud**.

You need **admin** on the repo so Woodpecker can install webhooks.

App repos do **not** need to be activated in Woodpecker when using `pc ship` — only personal-cloud runs the ship pipeline.

## 3. Woodpecker secrets (personal-cloud repo)

In Woodpecker: **personal-cloud** repo → **Settings** → **Secrets**:

| Name | Value |
|------|--------|
| `ghcr_token` | GitHub PAT (classic or fine-grained) with `write:packages` and `read:packages` |
| `github_clone_token` | Optional — PAT with repo read access for private app repos |

Fine-grained PAT: limit to the repos and Packages permission you need.

## 4. GHCR package visibility

After the first successful ship, images appear at:

`ghcr.io/your-github-username/<app-name>`

For a **private** package, set visibility in GitHub Packages or grant the VM pull credentials (the pipeline logs in with `ghcr_token` before `compose pull`).

## 5. VM paths the pipeline expects

| Path | Purpose |
|------|---------|
| `/opt/personal-cloud/apps/<app>/.env` | Runtime secrets (not in git) |
| `/var/run/docker.sock` | Agent mounts this for build/deploy steps |
| Network `personal-cloud_edge` | Created by `platform/docker compose up` |

Bootstrap clones the repo to `/opt/personal-cloud`. The ship pipeline renders compose and Caddy config on disk and uses the **stable** `.env` per app.

## 6. Verify a deploy

From your app repo:

```bash
pc validate
pc ship --wait
```

On the VM:

```bash
docker compose -f /opt/personal-cloud/apps/<app>/compose.yaml ps
curl -fsS "https://<route-host>/health"
```

## Rollback

```bash
IMAGE_TAG=v0.0.9 /opt/personal-cloud/scripts/deploy-app.sh <app-name>
```

Or re-run a previous pipeline from the Woodpecker UI.
