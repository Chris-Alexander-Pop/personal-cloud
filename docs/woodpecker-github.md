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
3. **Repositories** → **New repository** → enable:
   - `your-github-username/personal-cloud` (compose validation on push)
   - `your-github-username/your-app` (release pipeline on tags)

You need **admin** on each repo so Woodpecker can install webhooks.

## 3. Woodpecker secrets (your-app repo)

In Woodpecker: **your-app** repo → **Settings** → **Secrets**:

| Name | Value |
|------|--------|
| `ghcr_token` | GitHub PAT (classic or fine-grained) with `write:packages` and `read:packages` |

Fine-grained PAT: limit to the `your-app` repo and Packages permission.

## 4. GHCR package visibility

After the first successful release build, the image appears at:

`ghcr.io/your-github-username/example-app`

For a **private** repo, set the package to **private** or grant the VM’s pull credentials (the pipeline logs in with `ghcr_token` before `compose pull`).

## 5. VM paths the pipeline expects

| Path | Purpose |
|------|---------|
| `/opt/personal-cloud/apps/example-app/.env` | Runtime secrets (not in git) |
| `/var/run/docker.sock` | Agent mounts this for build/deploy steps |
| Network `personal-cloud_edge` | Created by `platform/docker compose up` |

Bootstrap clones the repo to `/opt/personal-cloud`. The deploy step clones fresh compose YAML from GitHub but uses the **stable** `.env` on disk.

## 6. First release

```bash
cd your-app
git tag v0.1.0
git push origin v0.1.0
```

In Woodpecker → your-app → Pipelines, confirm:

1. `test-server` passes  
2. `build-and-push` publishes to GHCR  
3. `deploy` runs `compose pull` / `up -d`

Verify:

```bash
curl -fsS https://api.example.com/health
# or on the VM:
docker exec example-app timeout 1 bash -c 'cat < /dev/null > /dev/tcp/127.0.0.1/8080'
```

## Rollback

```bash
IMAGE_TAG=v0.0.9 /opt/personal-cloud/scripts/deploy-app.sh example-app
```

Or re-run a previous tag pipeline from the Woodpecker UI.
