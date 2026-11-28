#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
BACKUP_DIR="${SELECTOS_BACKUP_DIR:-$ROOT/backups}"
mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
TMP="$BACKUP_DIR/.selectos-$STAMP.sql.tmp"
OUT="$BACKUP_DIR/selectos-$STAMP.sql.gz"
cleanup(){ rm -f "$TMP"; }
trap cleanup EXIT
docker compose exec -T postgres pg_dump -U selectos -d selectos --no-owner --no-acl > "$TMP"
test -s "$TMP"
gzip -9 "$TMP"
gzip -t "$TMP.gz"
mv "$TMP.gz" "$OUT"
chmod 600 "$OUT"
find "$BACKUP_DIR" -type f -name 'selectos-*.sql.gz' -mtime +30 -delete
echo "Backup created: $OUT"
