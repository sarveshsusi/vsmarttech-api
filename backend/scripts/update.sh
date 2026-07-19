#!/usr/bin/env bash
# Pull latest code and rebuild containers, with automatic git rollback on failure.
# Usage: ~/scripts/update.sh
# Rollback only: ~/scripts/update.sh --rollback
set -euo pipefail

APP_DIR="${APP_DIR:-$HOME/app}"
COMPOSE_DIR="${APP_DIR}/backend"
PREV_REF_FILE="${APP_DIR}/.last-good-deploy"

cd "${APP_DIR}"

rollback() {
  if [[ ! -f "${PREV_REF_FILE}" ]]; then
    echo "No previous good deploy recorded at ${PREV_REF_FILE}" >&2
    exit 1
  fi
  local prev
  prev="$(cat "${PREV_REF_FILE}")"
  echo "[$(date -Is)] rolling back to ${prev}…"
  git checkout --force "${prev}"
  cd "${COMPOSE_DIR}"
  docker compose up -d --build
  echo "[$(date -Is)] rollback complete"
  docker compose ps
}

if [[ "${1:-}" == "--rollback" ]]; then
  rollback
  exit 0
fi

PREV="$(git rev-parse HEAD)"
echo "[$(date -Is)] updating from ${PREV}…"

git pull --ff-only

cd "${COMPOSE_DIR}"

# Keep previous image tag for quick image-level recovery
if docker image inspect vsmart-backend:latest >/dev/null 2>&1; then
  docker tag vsmart-backend:latest "vsmart-backend:prev" || true
fi

if ! docker compose up -d --build; then
  echo "[$(date -Is)] deploy failed — restoring git ${PREV}" >&2
  cd "${APP_DIR}"
  git checkout --force "${PREV}"
  cd "${COMPOSE_DIR}"
  docker compose up -d --build || true
  exit 1
fi

echo "[$(date -Is)] health:"
if ! curl -fsS --max-time 10 http://127.0.0.1:8081/healthz; then
  echo
  echo "[$(date -Is)] healthz failed — rolling back" >&2
  cd "${APP_DIR}"
  git checkout --force "${PREV}"
  cd "${COMPOSE_DIR}"
  docker compose up -d --build
  exit 1
fi
echo
curl -fsS --max-time 10 http://127.0.0.1:8081/readyz || true
echo

echo "$(git -C "${APP_DIR}" rev-parse HEAD)" > "${PREV_REF_FILE}"
docker image prune -f
docker compose ps
echo "[$(date -Is)] update done (recorded good deploy)"
