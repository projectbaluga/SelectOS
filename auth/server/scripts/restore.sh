#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
if [[ $# -ne 1 || ! -f "$1" ]]; then echo "Usage: $0 /path/to/selectos-backup.sql[.gz]" >&2; exit 2; fi
FILE="$1"
echo "WARNING: this replaces the current SelectOS database contents."
read -r -p "Type RESTORE to continue: " CONFIRM
[[ "$CONFIRM" == "RESTORE" ]] || { echo "Cancelled."; exit 1; }
if [[ "$FILE" == *.gz ]]; then gzip -dc -- "$FILE" | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U selectos -d selectos
else cat -- "$FILE" | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U selectos -d selectos
fi
echo "Restore completed. Verify application health and records."
