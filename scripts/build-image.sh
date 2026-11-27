#!/usr/bin/env bash
set -euo pipefail

# SelectOS image builder
# Creates a 512 MiB bootable FAT32 UEFI image from the repository tree.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist"
IMG="${DIST}/SelectOS.img"
SIZE_MB=512
LABEL="SELECTOS"

echo "==> Building SelectOS image (${SIZE_MB} MiB)"

rm -rf "${DIST}"
mkdir -p "${DIST}"

echo "Creating empty image..."
dd if=/dev/zero of="${IMG}" bs=1M count="${SIZE_MB}" status=none

echo "Partitioning..."
parted -s "${IMG}" mklabel msdos
parted -s "${IMG}" mkpart primary fat32 1MiB 100%
parted -s "${IMG}" set 1 boot on
parted -s "${IMG}" set 1 esp on 2>/dev/null || true

LOOP=$(losetup -fP --show "${IMG}")
PART="${LOOP}p1"
echo "Using ${LOOP}"

cleanup() {
  set +e
  umount "${MNT}" 2>/dev/null || true
  losetup -d "${LOOP}" 2>/dev/null || true
}
trap cleanup EXIT

echo "Formatting FAT32 (${LABEL})..."
mkfs.vfat -F 32 -n "${LABEL}" "${PART}"

MNT=$(mktemp -d)
mount "${PART}" "${MNT}"

echo "Copying files..."
mkdir -p "${MNT}/EFI/BOOT" "${MNT}/boot/grub" "${MNT}/icons"

cp "${ROOT}/EFI/BOOT/"*          "${MNT}/EFI/BOOT/" 2>/dev/null || true
cp "${ROOT}/boot/vmlinuz"        "${MNT}/boot/" 2>/dev/null || true
cp "${ROOT}/boot/gate.img"       "${MNT}/boot/" 2>/dev/null || true
cp "${ROOT}/boot/grub/grub.cfg"  "${MNT}/boot/grub/"
cp "${ROOT}/config/gate.json"    "${MNT}/"
cp "${ROOT}/icons/"*             "${MNT}/icons/" 2>/dev/null || true
touch "${MNT}/gate.marker"

chmod 755 "${MNT}/EFI/BOOT/"* "${MNT}/boot/vmlinuz" "${MNT}/boot/gate.img" 2>/dev/null || true

sync
umount "${MNT}"
rmdir "${MNT}"
losetup -d "${LOOP}"
trap - EXIT

echo
echo "==> Done: ${IMG}"
ls -lh "${IMG}"
echo
echo "Write to USB with:"
echo "  sudo dd if=${IMG} of=/dev/sdX bs=4M status=progress oflag=sync"
echo "(Replace /dev/sdX carefully!)"
