#!/usr/bin/env bash
# Render compose + Caddy site for an app. Called from Woodpecker ship pipeline.
set -euo pipefail

: "${APP_NAME:?}"
: "${IMAGE:?}"
: "${IMAGE_TAG:?}"
: "${SERVICE_CONTAINER:?}"
: "${SERVICE_PORT:?}"
: "${EXPOSURE:?}"
: "${ROUTE_HOST:?}"
: "${COMPOSE_TEMPLATE:=default}"

PC_ROOT="${PC_ROOT:-/opt/personal-cloud}"
TEMPLATE_ROOT="${TEMPLATE_ROOT:-${PC_ROOT}/templates}"
APP_DIR="${PC_ROOT}/apps/${APP_NAME}"
COMPOSE_TEMPLATE_FILE="${TEMPLATE_ROOT}/compose/${COMPOSE_TEMPLATE}.yaml.tmpl"
CADDY_SITES="${PC_ROOT}/platform/caddy/sites"

if [[ ! -f "${COMPOSE_TEMPLATE_FILE}" ]]; then
  echo "Unknown compose template: ${COMPOSE_TEMPLATE}" >&2
  exit 1
fi

mkdir -p "${APP_DIR}" "${CADDY_SITES}"

export APP_NAME IMAGE IMAGE_TAG SERVICE_CONTAINER SERVICE_PORT

# POSTGRES_PASSWORD for with-postgres template (from env file if present)
ENV_FILE="${COMPOSE_ENV_FILE:-${APP_DIR}/.env}"
if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck source=/dev/null
  source "${ENV_FILE}"
  set +a
fi
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-change-me}"
export MEDIA_HOST_PATH="${MEDIA_HOST_PATH:-/opt/personal-cloud/data/orthodox-talks}"
export DATA_HOST_PATH="${DATA_HOST_PATH:-/opt/personal-cloud/data/${APP_NAME}}"

mkdir -p "${DATA_HOST_PATH}"

echo "Rendering compose -> ${APP_DIR}/compose.yaml"
envsubst '${APP_NAME} ${IMAGE} ${IMAGE_TAG} ${SERVICE_CONTAINER} ${SERVICE_PORT} ${POSTGRES_PASSWORD} ${MEDIA_HOST_PATH} ${DATA_HOST_PATH}' \
  < "${COMPOSE_TEMPLATE_FILE}" > "${APP_DIR}/compose.yaml"

# Host-network apps bind on the VM; Caddy (bridge) reaches them via host gateway.
PROXY_UPSTREAM="${SERVICE_CONTAINER}:${SERVICE_PORT}"
case "${COMPOSE_TEMPLATE}" in
  host-network|with-data-volume-host)
    PROXY_UPSTREAM="${DOCKER_HOST_GATEWAY:-host.docker.internal}:${SERVICE_PORT}"
    echo "Host-network template: Caddy upstream ${PROXY_UPSTREAM}"
    ;;
esac

SITE_FILE="${CADDY_SITES}/${APP_NAME}.caddy"
echo "Rendering Caddy site -> ${SITE_FILE}"

if [[ "${EXPOSURE}" == "private" ]]; then
  cat > "${SITE_FILE}" <<EOF
# ${APP_NAME} — Tailscale / private (tls internal)
${ROUTE_HOST} {
	tls internal
	reverse_proxy ${PROXY_UPSTREAM}
}
EOF
else
  cat > "${SITE_FILE}" <<EOF
# ${APP_NAME} — public
${ROUTE_HOST} {
	reverse_proxy ${PROXY_UPSTREAM}
}
EOF
fi

echo "Provision complete for ${APP_NAME}"
