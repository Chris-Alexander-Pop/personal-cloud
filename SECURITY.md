# Security

## Reporting vulnerabilities

If you find a security issue, please open a private advisory on GitHub or contact the maintainer directly. Do not open a public issue for undisclosed vulnerabilities.

## Deployment hygiene

This project runs a CI/CD stack with **Docker socket access**. Treat misconfiguration as high risk.

| Surface | Recommendation |
|---------|----------------|
| Woodpecker UI (`:8000`) | Reachable **only** over Tailscale or another private network — not the public internet |
| SSH | Key-based auth only; disable password login |
| Firewall | Allow `80`/`443` publicly; restrict `22` to your admin network if possible |
| Secrets | Keep `platform/woodpecker/.env`, `apps/*/.env`, and `~/.config/pc/config.yaml` out of git |
| Woodpecker secrets | Store `ghcr_token` and optional `github_clone_token` in the Woodpecker UI, not in the repo |
| Ship pipeline | Manual trigger only; review pipeline variables before approving runs on shared runners |

## What this repo does not contain

Cloneable templates and docs only. Your VM addresses, tokens, tailnet names, and app secrets live on your machines — not in this repository.
