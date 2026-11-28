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
grep -q '#define KNOWN_LOADER_COUNT 10' "$SRC"
grep -q 'CopyAsciiToChar16' "$SRC"
grep -q 'BuildLoaderPrefix' "$SRC"
grep -q 'BuildLoaderPath' "$SRC"
grep -q 'BuildLoaderLabel' "$SRC"
grep -Fq '"\\EFI\\ubuntu\\grubx64.efi"' "$SRC"
grep -q 'CHAR16 Path\[128\]' "$SRC"
grep -q 'CHAR16 Label\[64\]' "$SRC"
grep -Fq 'Buffer[Length] = 0;' "$SRC"
grep -q 'return ConsoleWriteString(Buffer);' "$SRC"
grep -Fq 'SELECTOS_TEST_CHILD_PATH_NOT_FOUND\r\n' "$SRC"
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
