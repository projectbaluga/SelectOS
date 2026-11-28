#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/src/uefi/boot-manager.c"
bash -n "$ROOT/scripts/build-uefi-loader.sh"
if grep -Eiq 'SetVariable|GetVariable|ExitBootServices|efivarfs|BootNext' "$SRC"; then
  echo "UEFI loader must not alter firmware variables or call ExitBootServices." >&2
  exit 1
fi
grep -q 'LocateHandleBuffer' "$SRC"
grep -q 'EFI_FILE_MODE_READ' "$SRC"
grep -q 'LoadImage' "$SRC"
grep -q 'StartImage' "$SRC"
grep -q 'EFI_SECURITY_VIOLATION' "$SRC"
echo "UEFI loader source safety checks passed."
