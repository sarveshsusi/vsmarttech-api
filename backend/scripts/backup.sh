#!/usr/bin/env bash
# Daily Postgres backup → local file (+ optional S3 upload).
# Install: crontab -e
#   15 2 * * * /home/ubuntu/app/scripts/backup.sh >> /home/ubuntu/app/logs/backup.log 2>&1
set -euo pipefail

APP_DIR="${APP_DIR:-$HOME/app}"
BACKUP_DIR="${BACKUP_DIR:-$HOME/backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date +%Y%m%d_%H%M%S)"
FILE="${BACKUP_DIR}/vsmartcrm_${STAMP}.sql.gz"

mkdir -p "$BACKUP_DIR" "${APP_DIR}/logs"
cd "${APP_DIR}/backend"

# Load DB name/user from .env without printing secrets
set -a
# shellcheck disable=SC1091
source .env
set +a

echo "[$(date -Is)] starting backup → ${FILE}"

docker compose exec -T postgres \
  pg_dump -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" \
  | gzip -c > "${FILE}"

# Optional: upload to S3 if AWS CLI is configured / IAM role available
if command -v aws >/dev/null 2>&1 && [[ -n "${BACKUP_S3_URI:-}" ]]; then
  aws s3 cp "${FILE}" "${BACKUP_S3_URI%/}/$(basename "${FILE}")"
  echo "[$(date -Is)] uploaded to ${BACKUP_S3_URI}"
fi

# Prune old local backups
find "${BACKUP_DIR}" -name 'vsmartcrm_*.sql.gz' -mtime "+${KEEP_DAYS}" -delete

echo "[$(date -Is)] backup done ($(du -h "${FILE}" | awk '{print $1}'))"
