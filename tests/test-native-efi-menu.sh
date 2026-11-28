#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MENU="$ROOT/src/native-initramfs/efi-menu"
bash -n "$MENU"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/sys/sda1" "$TMP/dev" "$TMP/bin" "$TMP/mount-root" "$TMP/console"
: > "$TMP/sys/sda1/partition"
: > "$TMP/dev/sda1"

cat > "$TMP/bin/blkid" <<'MOCK'
#!/bin/bash
field=""
while (($#)); do
  if [[ "$1" == "-s" ]]; then field="$2"; shift 2; else shift; fi
done
case "$field" in
  PART_ENTRY_TYPE) echo "c12a7328-f81f-11d2-ba4b-00a0c93ec93b" ;;
  PARTUUID) echo "1111-2222" ;;
esac
MOCK
cat > "$TMP/bin/mount" <<'MOCK'
#!/bin/bash
case "$*" in
  *"ro,nosuid,nodev,noexec"*) : ;;
  *) echo "mount options were not read-only/noexec" >&2; exit 1 ;;
esac
for last; do :; done
printf '%s\n' "$*" >> "$SELECTOS_TEST_MOUNT_LOG"
mkdir -p "$last/EFI/Microsoft/Boot"
: > "$last/EFI/Microsoft/Boot/bootmgfw.efi"
MOCK
cat > "$TMP/bin/umount" <<'MOCK'
#!/bin/sh
exit 0
MOCK
chmod +x "$TMP/bin/"*
printf '1\n' > "$TMP/console/input"

# Replace absolute BusyBox paths with host utilities for this fixture test.
mkdir -p "$TMP/bin/bin"
for cmd in rm tr wc sed; do
  ln -s "$(command -v "$cmd")" "$TMP/bin/bin/$cmd"
done
sed \
  -e "s|/bin/blkid|$TMP/bin/blkid|g" \
  -e "s|/bin/mount|$TMP/bin/mount|g" \
  -e "s|/bin/umount|$TMP/bin/umount|g" \
  -e "s|/bin/rm|$TMP/bin/bin/rm|g" \
  -e "s|/bin/tr|$TMP/bin/bin/tr|g" \
  -e "s|/bin/busybox wc|$TMP/bin/bin/wc|g" \
  -e "s|/bin/busybox tr|$TMP/bin/bin/tr|g" \
  -e "s|/bin/busybox sed|$TMP/bin/bin/sed|g" \
  "$MENU" > "$TMP/menu-test"
chmod +x "$TMP/menu-test"
output="$(SELECTOS_SYS_BLOCK="$TMP/sys" SELECTOS_DEV_ROOT="$TMP/dev" SELECTOS_MOUNT_ROOT="$TMP/mount-root" SELECTOS_CONSOLE="$TMP/console/input" SELECTOS_TEST_MOUNT_LOG="$TMP/mount.log" bash "$TMP/menu-test")"
grep -q 'Windows' <<< "$output"
grep -q 'Napili:' <<< "$output"
grep -q 'Hindi pa available ang UEFI handoff' <<< "$output"
grep -q 'ro,nosuid,nodev,noexec' "$TMP/mount.log"

# No firmware-write or command-execution mechanism is permitted in this phase.
if grep -Eiq 'BootNext|efivarfs|(^|[[:space:]])(exec|chroot)[[:space:]]' "$MENU"; then
  echo "Unexpected firmware write or execution path in EFI menu." >&2
  exit 1
fi
echo "Native EFI menu read-only scan and selection-record tests passed."
