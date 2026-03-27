#!/bin/bash
set -e # Stop on error

# --- Config ---
export PROJECT_DIR="/opt/parsa-go"
[ -f /opt/parsa-go/.env ] && set -a && source /opt/parsa-go/.env && set +a

export BACKUP_DIR="/var/backups/parsa-snapshots"
export MG_BUCKET="${MG_BUCKET:-parsa-backup}"
export DATE=$(date +%Y-%m-%d_%H%M%S)
export ARCHIVE_NAME="parsa_backup_$DATE.tar.gz"
# Dump staging: /var/backups often not writable for non-root; /tmp always is
export DUMP_PATH="${TMPDIR:-/tmp}/parsa_dump_${DATE}.sql"

# --- 1. Environment Check ---
# DB_* required; for upload set MGC_API_KEY in .env (systemd runs as root, not your login mgc profile)
if [ -z "$DB_PASSWORD" ] || [ -z "$MG_BUCKET" ]; then
    echo "ERROR: Missing environment variables (DB_PASSWORD or MG_BUCKET)."
    exit 1
fi

export PGPASSWORD="$DB_PASSWORD"

# --- 2. Create Backup ---
mkdir -p "$BACKUP_DIR"

echo "1. Creating DB Dump..."
pg_dump -h localhost -U "$DB_USER" -d "$DB_NAME" -F p > "$DUMP_PATH"

echo "2. Compressing dump..."
tar -czf "$BACKUP_DIR/$ARCHIVE_NAME" \
    -C "$(dirname "$DUMP_PATH")" "$(basename "$DUMP_PATH")"

rm -f "$DUMP_PATH"

# --- 3. Upload to Magalu (Using mgc-cli) ---
echo "3. Uploading to Magalu Bucket: $MG_BUCKET..."
# systemd runs as root: ~/.config/mgc from your login user is NOT used. Set MGC_API_KEY in .env
# (create key: mgc object-storage api-key create … ; list UUID: mgc object-storage api-key list)
MGC_ARGS=()
[ -n "${MGC_API_KEY:-}" ] && MGC_ARGS+=(--api-key "$MGC_API_KEY")

# Syntax: mgc object-storage objects upload <source> <destination_bucket>/<filename>
mgc "${MGC_ARGS[@]}" object-storage objects upload "$BACKUP_DIR/$ARCHIVE_NAME" "$MG_BUCKET/$ARCHIVE_NAME"

# --- 4. Cleanup Local Storage ---
echo "4. Cleaning up old local files..."
find "$BACKUP_DIR" -type f -name "parsa_backup_*.tar.gz" -mtime +2 -delete

echo "SUCCESS: Backup uploaded via mgc-cli."