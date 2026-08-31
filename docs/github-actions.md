# Ship from GitHub (no local clone)

Woodpecker [`.woodpecker/ship.yaml`](../.woodpecker/ship.yaml) is unchanged. There is no second deploy API. `pc` (or the Actions workflow that runs it) reads `.personal-cloud.yaml` from **GitHub.com**, then triggers that same manual pipeline. The VM still clones the app, builds, and deploys.

GitHub cannot reach a Tailscale-only Woodpecker host on its own. The workflow joins your tailnet for the job, then calls Woodpecker the same way `pc ship` does from a phone or laptop.

## From Termux / any machine (no app clone)

Install `pc` and copy `~/.config/pc/config.yaml` once. Then:

```bash
pc ship music-serve --wait
# or
pc ship your-user/music-serve --ref main --wait
```

`music-serve` expands to `github.owner/music-serve`. The CLI fetches the manifest and HEAD SHA from the GitHub API. Woodpecker clones that SHA.

For a **private** app repo, set `github.token` (contents:read) in config, or `GITHUB_TOKEN` / `GH_TOKEN` in the environment.

`pc status` and `pc logs` only need the config file — no clone.

## From GitHub Actions

On **personal-cloud**: **Actions → ship → Run workflow**. Pick the app from the dropdown (and an optional ref). That is the phone-friendly path.

GitHub cannot fill `workflow_dispatch` choice lists at click time. [`.github/workflows/sync-ship-apps.yml`](../.github/workflows/sync-ship-apps.yml) scans your GitHub repos for a root `.personal-cloud.yaml` and commits the dropdown. It runs daily, after a successful **ship**, and on demand (**Actions → sync-ship-apps**). `GH_TOKEN` is required to see **private** app repos in that list.

First-time apps: choose **other** and type `music-serve` (or `owner/music-serve`). After that ship succeeds, sync adds it to the dropdown.

The workflow is [`.github/workflows/ship.yml`](../.github/workflows/ship.yml). It builds `pc`, joins Tailscale, and runs `pc ship <repo>`. The option values are **GitHub repo names** (for example `ro-prayers-app`), not the `name:` field in the manifest.

### Secrets on the personal-cloud repo

| Secret | Purpose |
|--------|---------|
| `WOODPECKER_URL` | Same Tailscale URL as `woodpecker.url` in `pc` config |
| `WOODPECKER_TOKEN` | Woodpecker personal token |
| `TS_OAUTH_CLIENT_ID` / `TS_OAUTH_SECRET` | Tailscale OAuth client ([Trust credentials](https://tailscale.com/kb/1215/oauth-clients)) tagged `tag:ci` |
| `TAILNET_BASE` | e.g. `tail123.ts.net` — private route suffix |
| `GH_TOKEN` | PAT with `repo` (or at least `contents:read` on private apps). Needed to ship private repos **and** to list them in the ship dropdown |

ACL: allow `tag:ci` to the VM on TCP `8000` only. Use an ephemeral CI node.

### Optional: ship on push from an app repo

In `music-serve` (secrets must exist on that repo, or use `secrets: inherit` from a caller that has them):

```yaml
name: ship
on:
  workflow_dispatch:
  push:
    branches: [main]
jobs:
  ship:
    uses: your-user/personal-cloud/.github/workflows/ship-reusable.yml@main
    with:
      repo: ${{ github.repository }}
      ref: ${{ github.sha }}
    secrets:
      WOODPECKER_URL: ${{ secrets.WOODPECKER_URL }}
      WOODPECKER_TOKEN: ${{ secrets.WOODPECKER_TOKEN }}
      TS_OAUTH_CLIENT_ID: ${{ secrets.TS_OAUTH_CLIENT_ID }}
      TS_OAUTH_SECRET: ${{ secrets.TS_OAUTH_SECRET }}
      TAILNET_BASE: ${{ secrets.TAILNET_BASE }}
      GH_TOKEN: ${{ secrets.GH_TOKEN }}
```

No new Woodpecker file, no per-app pipeline variables, no clone on the phone.
