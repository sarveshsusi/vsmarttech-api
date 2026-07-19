#!/usr/bin/env bash
# Pull latest code and rebuild containers.
# Usage: ~/app/scripts/update.sh
set -euo pipefail

APP_DIR="${APP_DIR:-$HOME/app}"
cd "${APP_DIR}"

echo "[$(date -Is)] updating…"
git pull --ff-only

cd backend
docker compose up -d --build
docker image prune -f

echo "[$(date -Is)] health:"
curl -fsS http://127.0.0.1:8081/healthz || true
echo
curl -fsS http://127.0.0.1:8081/readyz || true
echo
docker compose ps
echo "[$(date -Is)] update done"
