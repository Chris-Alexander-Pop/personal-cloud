#!/usr/bin/env bash
# Rewrite .github/workflows/ship.yml choice options from GitHub repos that
# have a root .personal-cloud.yaml. GitHub cannot populate workflow_dispatch
# dropdowns at click time — the list has to live in the YAML.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKFLOW="${ROOT}/.github/workflows/ship.yml"
BEGIN="# BEGIN SHIP_APPS"
END="# END SHIP_APPS"
OTHER="other"

QUERY='query($login: String!, $after: String) {
  repositoryOwner(login: $login) {
    repositories(first: 100, after: $after, ownerAffiliations: OWNER, isArchived: false) {
      pageInfo { hasNextPage endCursor }
      nodes {
        name
        object(expression: "HEAD:.personal-cloud.yaml") {
          ... on Blob { byteSize }
        }
      }
    }
  }
}'

usage() {
  cat <<EOF
Usage: $(basename "$0") [--dry-run] [--check] [--owner LOGIN] [--workflow PATH]

Discover GitHub repos with .personal-cloud.yaml and update the ship
workflow_dispatch dropdown. Empty discovery leaves the file unchanged.
EOF
}

DRY_RUN=0
CHECK=0
OWNER="${GITHUB_REPOSITORY_OWNER:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    --check) CHECK=1; shift ;;
    --owner)
      OWNER="${2:?--owner requires a login}"
      shift 2
      ;;
    --workflow)
      WORKFLOW="${2:?--workflow requires a path}"
      shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 1 ;;
  esac
done

if [[ -z "$OWNER" ]]; then
  if command -v gh >/dev/null 2>&1; then
    OWNER="$(gh api user --jq .login 2>/dev/null || true)"
  fi
fi
if [[ -z "$OWNER" ]]; then
  echo "set --owner or GITHUB_REPOSITORY_OWNER (or authenticate gh)" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi
if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required" >&2
  exit 1
fi

discover() {
  local after="" has_next="true" resp names
  names="$(mktemp)"

  while [[ "$has_next" == "true" ]]; do
    if [[ -n "$after" ]]; then
      resp="$(gh api graphql -f query="$QUERY" -f login="$OWNER" -f after="$after")"
    else
      resp="$(gh api graphql -f query="$QUERY" -f login="$OWNER")"
    fi
    if [[ "$(jq -r '.data.repositoryOwner // empty' <<<"$resp")" == "" ]]; then
      echo "could not list repos for ${OWNER}" >&2
      jq -r '.errors // .message // .' <<<"$resp" >&2 || true
      rm -f "$names"
      return 1
    fi
    jq -r '
      .data.repositoryOwner.repositories.nodes[]
      | select(.object != null)
      | .name
    ' <<<"$resp" >>"$names"
    has_next="$(jq -r '.data.repositoryOwner.repositories.pageInfo.hasNextPage' <<<"$resp")"
    after="$(jq -r '.data.repositoryOwner.repositories.pageInfo.endCursor // empty' <<<"$resp")"
  done

  sort -fu "$names"
  rm -f "$names"
}

rewrite() {
  local file="$1" apps_file="$2" mode="$3"
  python3 - "$file" "$apps_file" "$mode" "$BEGIN" "$END" <<'PY'
import pathlib, sys

path = pathlib.Path(sys.argv[1])
apps = [l.strip() for l in pathlib.Path(sys.argv[2]).read_text().splitlines() if l.strip()]
mode = sys.argv[3]
begin, end = sys.argv[4], sys.argv[5]
text = path.read_text()
si, ei = text.find(begin), text.find(end)
if si < 0 or ei < 0 or ei <= si:
    sys.exit(f"missing {begin} / {end} markers in {path}")
begin_line_end = text.find("\n", si)
end_line_start = text.rfind("\n", 0, ei) + 1
indent = text[end_line_start:ei]
body = "".join(f"{indent}- {app}\n" for app in apps)
new = text[: begin_line_end + 1] + body + text[end_line_start:]
if mode == "check":
    sys.exit(0 if new == text else 1)
if mode == "print":
    sys.stdout.write(new)
    sys.exit(0)
if new != text:
    path.write_text(new)
PY
}

APPS="$(discover)"
if [[ -z "$APPS" ]]; then
  echo "no repos with .personal-cloud.yaml found for ${OWNER}; leaving ${WORKFLOW} unchanged" >&2
  exit 0
fi

echo "$APPS" | sed 's/^/  /'
echo "owner: ${OWNER}"

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
printf '%s\n' "$APPS" >"$TMP"

# Sentinel is outside the markers and must not appear as a discovered repo.
if grep -qx "$OTHER" "$TMP"; then
  echo "refusing to sync: a repo is named '${OTHER}', which is the dropdown sentinel" >&2
  exit 1
fi

if [[ "$CHECK" -eq 1 ]]; then
  if rewrite "$WORKFLOW" "$TMP" check; then
    echo "ship dropdown is up to date"
    exit 0
  fi
  echo "ship dropdown is stale — run $0" >&2
  exit 1
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
  rewrite "$WORKFLOW" "$TMP" print
  exit 0
fi

rewrite "$WORKFLOW" "$TMP" write
echo "updated ${WORKFLOW}"
