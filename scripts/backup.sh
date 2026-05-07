#!/usr/bin/env bash
# =============================================================================
#  MPanel — PostgreSQL Backup Script
#  Runs pg_dump inside the postgres container and rotates old backups.
#
#  Usage (manual):     ./scripts/backup.sh
#  Usage (scheduled):  add to crontab — see README for examples
#
#  Environment (reads from .env automatically if present):
#    BACKUP_DIR      Local directory to store dumps  (default: ./backups)
#    BACKUP_RETAIN   Days to keep old backups         (default: 7)
#    POSTGRES_DB     Database name                    (default: mpanel)
#    POSTGRES_USER   Database user                    (default: mpanel)
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Load .env if present (without overriding already-exported vars)
if [ -f "$PROJECT_DIR/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  source "$PROJECT_DIR/.env"
  set +a
fi

BACKUP_DIR="${BACKUP_DIR:-$PROJECT_DIR/backups}"
BACKUP_RETAIN="${BACKUP_RETAIN:-7}"
POSTGRES_DB="${POSTGRES_DB:-mpanel}"
POSTGRES_USER="${POSTGRES_USER:-mpanel}"
CONTAINER="mpanel-postgres"
TIMESTAMP="$(date '+%Y%m%d_%H%M%S')"
DUMP_FILE="$BACKUP_DIR/mpanel_${POSTGRES_DB}_${TIMESTAMP}.sql.gz"

# ── Preflight ─────────────────────────────────────────────────────────────────
if ! docker inspect "$CONTAINER" > /dev/null 2>&1; then
  echo "[ERROR] Container '$CONTAINER' is not running." >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"

# ── Dump ──────────────────────────────────────────────────────────────────────
echo "[INFO] Starting backup → $DUMP_FILE"
docker exec "$CONTAINER" \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-password \
  | gzip > "$DUMP_FILE"

SIZE="$(du -sh "$DUMP_FILE" | cut -f1)"
echo "[INFO] Backup complete. Size: $SIZE"

# ── Rotate ────────────────────────────────────────────────────────────────────
echo "[INFO] Removing backups older than ${BACKUP_RETAIN} days..."
find "$BACKUP_DIR" -maxdepth 1 -name "mpanel_*.sql.gz" \
  -mtime "+${BACKUP_RETAIN}" -delete -print \
  | sed 's/^/[INFO] Deleted: /'

REMAINING="$(find "$BACKUP_DIR" -maxdepth 1 -name "mpanel_*.sql.gz" | wc -l | tr -d ' ')"
echo "[INFO] Retained backups: $REMAINING"
