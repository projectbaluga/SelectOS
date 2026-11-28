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
  *) echo "mount options were not read-only/noexec: $*" >&2; exit 1 ;;
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

# Replace only the BusyBox utility paths; scanner and mount tools are injected
# through explicit environment overrides, just as the runtime supports.
mkdir -p "$TMP/bin/bin"
for cmd in rm tr wc sed; do
  ln -s "$(command -v "$cmd")" "$TMP/bin/bin/$cmd"
done
sed \
  -e "s|/bin/rm|$TMP/bin/bin/rm|g" \
  -e "s|/bin/tr|$TMP/bin/bin/tr|g" \
  -e "s|/bin/busybox wc|$TMP/bin/bin/wc|g" \
  -e "s|/bin/busybox tr|$TMP/bin/bin/tr|g" \
  -e "s|/bin/busybox sed|$TMP/bin/bin/sed|g" \
  "$MENU" > "$TMP/menu-test"
chmod +x "$TMP/menu-test"
if ! output="$(SELECTOS_SYS_BLOCK="$TMP/sys" SELECTOS_DEV_ROOT="$TMP/dev" SELECTOS_MOUNT_ROOT="$TMP/mount-root" SELECTOS_CONSOLE="$TMP/console/input" SELECTOS_BLKID="$TMP/bin/blkid" SELECTOS_MOUNT="$TMP/bin/mount" SELECTOS_UMOUNT="$TMP/bin/umount" SELECTOS_TEST_MOUNT_LOG="$TMP/mount.log" bash "$TMP/menu-test" 2>"$TMP/stderr")"; then
  cat "$TMP/stderr" >&2
  echo "EFI menu returned a failure." >&2
  exit 1
fi
for expected in Windows 'Napili:' 'Hindi pa available ang UEFI handoff'; do
  if ! grep -q "$expected" <<< "$output"; then
    echo "Missing expected menu output: $expected" >&2
    printf '%s\n' "$output" >&2
    exit 1
  fi
done
if ! grep -q 'ro,nosuid,nodev,noexec' "$TMP/mount.log"; then
  echo "ESP was not mounted with the required read-only/noexec options." >&2
  cat "$TMP/mount.log" >&2 || true
  exit 1
fi

# No firmware-write or command-execution mechanism is permitted in this phase.
if grep -Eiq 'BootNext|efivarfs|(^|[[:space:]])(exec|chroot)[[:space:]]' "$MENU"; then
  echo "Unexpected firmware write or execution path in EFI menu." >&2
  exit 1
fi
echo "Native EFI menu read-only scan and selection-record tests passed."
