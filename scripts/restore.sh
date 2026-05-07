#!/usr/bin/env bash
# =============================================================================
#  MPanel — PostgreSQL Restore Script
#  Restores a .sql.gz dump produced by backup.sh into the running container.
#
#  Usage: ./scripts/restore.sh <backup-file.sql.gz>
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ -f "$PROJECT_DIR/.env" ]; then
  set -a
  source "$PROJECT_DIR/.env"
  set +a
fi

POSTGRES_DB="${POSTGRES_DB:-mpanel}"
POSTGRES_USER="${POSTGRES_USER:-mpanel}"
CONTAINER="mpanel-postgres"
DUMP_FILE="${1:-}"

if [ -z "$DUMP_FILE" ]; then
  echo "Usage: $0 <backup-file.sql.gz>" >&2
  exit 1
fi

if [ ! -f "$DUMP_FILE" ]; then
  echo "[ERROR] File not found: $DUMP_FILE" >&2
  exit 1
fi

if ! docker inspect "$CONTAINER" > /dev/null 2>&1; then
  echo "[ERROR] Container '$CONTAINER' is not running." >&2
  exit 1
fi

echo "[WARN] This will drop and recreate the '$POSTGRES_DB' database."
read -r -p "Continue? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
  echo "Aborted."
  exit 0
fi

echo "[INFO] Restoring $DUMP_FILE → $POSTGRES_DB..."

# Drop & recreate via psql in the container
docker exec -i "$CONTAINER" psql -U "$POSTGRES_USER" -d postgres \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$POSTGRES_DB' AND pid <> pg_backend_pid();" \
  -c "DROP DATABASE IF EXISTS $POSTGRES_DB;" \
  -c "CREATE DATABASE $POSTGRES_DB OWNER $POSTGRES_USER;"

# Restore
gunzip -c "$DUMP_FILE" | docker exec -i "$CONTAINER" \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -q

echo "[INFO] Restore complete."
