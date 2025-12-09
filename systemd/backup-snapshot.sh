#!/bin/bash
set -e # Stop on error

# --- Config ---
PROJECT_DIR="/opt/parsa-go"
BACKUP_DIR="/var/backups/parsa-snapshots"
DATE=$(date +%Y-%m-%d_%H%M%S)
ARCHIVE_NAME="parsa_backup_$DATE.tar.gz"
DB_DUMP_NAME="parsadb_dump.sql"
MG_BUCKET="parsa-backup"

# --- 1. Environment Check ---
# We still need DB info, but NO LONGER need AWS keys if 'mgc' is auth'd globally
if [ -z "$DB_PASSWORD" ] || [ -z "$MG_BUCKET" ]; then
    echo "ERROR: Missing environment variables (DB_PASSWORD or MG_BUCKET)."
    exit 1
fi

export PGPASSWORD="$DB_PASSWORD"

# --- 2. Create Backup ---
mkdir -p "$BACKUP_DIR"

echo "1. Creating DB Dump..."
pg_dump -h localhost -U "$DB_USER" -d "$DB_NAME" -F p > "$PROJECT_DIR/$DB_DUMP_NAME"

echo "2. Compressing files..."
tar -czf "$BACKUP_DIR/$ARCHIVE_NAME" \
    --exclude='*.log' \
    --exclude='.git' \
    --exclude='tmp' \
    -C /opt parsa-go

# Remove raw SQL file immediately
rm "$PROJECT_DIR/$DB_DUMP_NAME"

# --- 3. Upload to Magalu (Using mgc-cli) ---
echo "3. Uploading to Magalu Bucket: $MG_BUCKET..."

# Syntax: mgc object-storage objects upload <source> <destination_bucket>/<filename>
mgc object-storage objects upload "$BACKUP_DIR/$ARCHIVE_NAME" "$MG_BUCKET/$ARCHIVE_NAME"

# --- 4. Cleanup Local Storage ---
echo "4. Cleaning up old local files..."
find "$BACKUP_DIR" -type f -name "parsa_backup_*.tar.gz" -mtime +2 -delete

echo "SUCCESS: Backup uploaded via mgc-cli."