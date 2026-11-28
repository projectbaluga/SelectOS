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
grep -q 'selectos-debug.log' "$SRC"
grep -q 'L"selectos-debug.log"' "$SRC"
grep -q 'SELECTOS_SCAN_RETURNED' "$SRC"
grep -q 'BuildFileDevicePath' "$SRC"
grep -q 'OpenPathFromRoot' "$SRC"
grep -q 'Walk one component at a time' "$SRC"
grep -q 'LocalPrint' "$SRC"
if grep -Eq '(^|[^[:alnum:]_])Print[[:space:]]*\(' "$SRC"; then
  echo "UEFI loader must not call GNU-EFI Print/CatPrint." >&2
  exit 1
fi
if grep -Eq '(^|[^[:alnum:]_])FileDevicePath[[:space:]]*\(' "$SRC"; then
  echo "UEFI loader must not depend on the distro GNU-EFI FileDevicePath helper." >&2
  exit 1
fi
grep -q 'LoadImage' "$SRC"
grep -q 'StartImage' "$SRC"
grep -q 'EFI_SECURITY_VIOLATION' "$SRC"
grep -q -- '-march=x86-64' "$ROOT/scripts/build-uefi-loader.sh"
grep -q -- '-mno-sse4.1' "$ROOT/scripts/build-uefi-loader.sh"
echo "UEFI loader source safety checks passed."
