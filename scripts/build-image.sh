#!/usr/bin/env bash
set -Eeuo pipefail

# SelectOS image builder. The legacy Gate initramfs is retained as the runtime
# for this rebrand; its on-disk compatibility names are gate.img/gate.json/
# gate.marker internally. The image filenames and visible branding are SelectOS.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist"
IMG="${DIST}/SelectOS.img"
SIZE_MB=512
LABEL="SELECTOS"
MNT=""
LOOP=""

echo "==> SelectOS image builder"

required=(
  "boot/vmlinuz"
  "boot/select.img"
  "EFI/BOOT/BOOTX64.EFI"
  "EFI/BOOT/grubx64.efi"
  "EFI/BOOT/mmx64.efi"
  "EFI/BOOT/grub.cfg"
  "boot/grub/grub.cfg"
  "config/select.json"
)
missing=()
for f in "${required[@]}"; do
  if [[ ! -s "${ROOT}/${f}" ]]; then
    missing+=("$f")
  fi
done
if (( ${#missing[@]} )); then
  printf 'ERROR: Required file missing or empty: %s\n' "${missing[@]}" >&2
  echo "Provide the five boot binaries extracted from the compatible legacy image." >&2
  echo "See README.md for the required source image and extracted boot files." >&2
  exit 1
fi

for cmd in dd parted losetup mkfs.vfat mount umount; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: Missing build dependency: $cmd" >&2; exit 1; }
done
if (( EUID != 0 )); then
  echo "ERROR: Run this image builder as root (for example: sudo bash scripts/build-image.sh)." >&2
  exit 1
fi

cleanup() {
  set +e
  if [[ -n "$MNT" ]]; then
    umount "$MNT" 2>/dev/null || true
    rmdir "$MNT" 2>/dev/null || true
  fi
  if [[ -n "$LOOP" ]]; then
    losetup -d "$LOOP" 2>/dev/null || true
  fi
}
trap cleanup EXIT

rm -rf "$DIST"
mkdir -p "$DIST"
echo "Creating ${SIZE_MB} MiB image..."
dd if=/dev/zero of="$IMG" bs=1M count="$SIZE_MB" status=none

echo "Partitioning..."
parted -s "$IMG" mklabel msdos
parted -s "$IMG" mkpart primary fat32 1MiB 100%
parted -s "$IMG" set 1 boot on
parted -s "$IMG" set 1 esp on 2>/dev/null || true

LOOP="$(losetup -fP --show "$IMG")"
PART="${LOOP}p1"
echo "Loop device: $LOOP"
mkfs.vfat -F 32 -n "$LABEL" "$PART"

MNT="$(mktemp -d)"
mount "$PART" "$MNT"

echo "Copying boot files..."
mkdir -p "$MNT/EFI/BOOT" "$MNT/boot/grub" "$MNT/icons"
cp "$ROOT/EFI/BOOT/BOOTX64.EFI" "$ROOT/EFI/BOOT/grubx64.efi" "$ROOT/EFI/BOOT/mmx64.efi" "$MNT/EFI/BOOT/"
cp "$ROOT/EFI/BOOT/grub.cfg" "$MNT/EFI/BOOT/"
cp "$ROOT/boot/vmlinuz" "$ROOT/boot/select.img" "$MNT/boot/"
cp "$ROOT/boot/grub/grub.cfg" "$MNT/boot/grub/"

# The legacy initramfs explicitly loads /gate.json and searches for
# /sys/class/block/esp/gate.marker. Keep these internal compatibility names
# while the boot menu, image label, and JSON brand use SelectOS.
cp "$ROOT/config/select.json" "$MNT/gate.json"

if compgen -G "$ROOT/icons/*" >/dev/null; then
  cp -r "$ROOT"/icons/. "$MNT/icons/"
fi
touch "$MNT/gate.marker"
sync
cleanup
trap - EXIT

[[ -s "$IMG" ]] || { echo "ERROR: Image was not created." >&2; exit 1; }
echo "==> Success: $IMG"
ls -lh "$IMG"
echo "Write to USB only after verifying the target device:"
echo "  sudo dd if=$IMG of=/dev/sdX bs=4M status=progress oflag=sync"
