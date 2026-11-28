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
#!/bin/sh
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
#!/bin/sh
target="\${@: -1}"
mkdir -p "$target/EFI/Microsoft/Boot"
: > "$target/EFI/Microsoft/Boot/bootmgfw.efi"
MOCK
cat > "$TMP/bin/umount" <<'MOCK'
#!/bin/sh
exit 0
MOCK
chmod +x "$TMP/bin/"*
printf '1\n' > "$TMP/console/input"
# Bash runs the script for the host test; BusyBox absolute paths are replaced
# through PATH by small test shims only for the commands used by this script.
mkdir -p "$TMP/bin/bin"
for cmd in rm tr busybox wc sed; do
  case "$cmd" in
    busybox) cat > "$TMP/bin/bin/busybox" <<'SH'
#!/bin/sh
exec /bin/"$@"
SH
      ;;
    *) ln -s "$(command -v "$cmd")" "$TMP/bin/bin/$cmd" ;;
  esac
done
# The test substitutes the absolute utility paths used by the initramfs script.
sed \
  -e "s|/bin/blkid|$TMP/bin/blkid|g" \
  -e "s|/bin/mount|$TMP/bin/mount|g" \
  -e "s|/bin/umount|$TMP/bin/umount|g" \
  -e "s|/bin/rm|$TMP/bin/bin/rm|g" \
  -e "s|/bin/tr|$TMP/bin/bin/tr|g" \
  -e "s|/bin/busybox wc|$TMP/bin/bin/wc|g" \
  -e "s|/bin/busybox tr|$TMP/bin/bin/tr|g" \
  -e "s|/bin/busybox sed|$TMP/bin/bin/sed|g" \
  -e "s|/bin/console|$TMP/console/input|g" \
  "$MENU" > "$TMP/menu-test"
chmod +x "$TMP/menu-test"
output="$(PATH="$TMP/bin:$PATH" SELECTOS_SYS_BLOCK="$TMP/sys" SELECTOS_DEV_ROOT="$TMP/dev" SELECTOS_MOUNT_ROOT="$TMP/mount-root" SELECTOS_CONSOLE="$TMP/console/input" bash "$TMP/menu-test")"
grep -q 'Windows' <<< "$output"
grep -q 'Napili:' <<< "$output"
grep -q 'Hindi pa available ang UEFI handoff' <<< "$output"
# Ensure the source contains no executable chainload/firmware-write operation.
if grep -Eiq 'BootNext|efivarfs|chainload|(^|[[:space:]])(exec|chroot)[[:space:]]' "$MENU"; then
  echo "Unexpected firmware write or execution path in EFI menu." >&2
  exit 1
fi
echo "Native EFI menu read-only scan and selection-record tests passed."
