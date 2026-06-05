#!/usr/bin/env bash
# Manual deploy helper (same paths Woodpecker uses on the VM).
set -euo pipefail

APP="${1:?Usage: deploy-app.sh <app-name> [image-tag]}"
TAG="${2:-${IMAGE_TAG:-latest}}"
ROOT="${PERSONAL_CLOUD_ROOT:-/opt/personal-cloud}"
COMPOSE="${ROOT}/apps/${APP}/compose.yaml"
ENV_FILE="${ROOT}/apps/${APP}/.env"

if [[ ! -f "$COMPOSE" ]]; then
  echo "Missing compose file: $COMPOSE" >&2
  exit 1
fi
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE (copy from .env.example)" >&2
  exit 1
fi

export IMAGE_TAG="$TAG"
echo "Deploying ${APP} with IMAGE_TAG=${IMAGE_TAG}"

docker compose -f "$COMPOSE" --env-file "$ENV_FILE" pull
docker compose -f "$COMPOSE" --env-file "$ENV_FILE" up -d

echo "Done. Check: docker compose -f $COMPOSE ps"
