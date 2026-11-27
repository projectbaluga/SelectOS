#!/usr/bin/env bash
set -euo pipefail

# SelectOS – production image builder
# Creates a 512 MiB bootable FAT32 UEFI image.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist"
IMG="${DIST}/SelectOS.img"
SIZE_MB=512
LABEL="SELECTOS"

echo "==> SelectOS image builder"

# -------------------------------------------------------
# Check required binary components
# -------------------------------------------------------
missing=0
for f in \
  "boot/vmlinuz" \
  "boot/select.img" \
  "EFI/BOOT/BOOTX64.EFI" \
  "EFI/BOOT/grubx64.efi" \
  "EFI/BOOT/mmx64.efi"
do
  if [[ ! -f "${ROOT}/${f}" ]]; then
    echo "ERROR: Missing required file: ${f}"
    missing=1
  fi
done

if [[ ${missing} -eq 1 ]]; then
  echo
  echo "Please place the required binary files in the repository tree."
  echo "See README.md for extraction instructions from the original image."
  exit 1
fi

# -------------------------------------------------------
# Build
# -------------------------------------------------------
rm -rf "${DIST}"
mkdir -p "${DIST}"

echo "Creating ${SIZE_MB} MiB image..."
dd if=/dev/zero of="${IMG}" bs=1M count="${SIZE_MB}" status=none

echo "Partitioning..."
parted -s "${IMG}" mklabel msdos
parted -s "${IMG}" mkpart primary fat32 1MiB 100%
parted -s "${IMG}" set 1 boot on
parted -s "${IMG}" set 1 esp on 2>/dev/null || true

LOOP=$(losetup -fP --show "${IMG}")
PART="${LOOP}p1"
echo "Loop device: ${LOOP}"

cleanup() {
  set +e
  umount "${MNT}" 2>/dev/null || true
  losetup -d "${LOOP}" 2>/dev/null || true
}
trap cleanup EXIT

echo "Formatting FAT32 (label: ${LABEL})..."
mkfs.vfat -F 32 -n "${LABEL}" "${PART}"

MNT=$(mktemp -d)
mount "${PART}" "${MNT}"

echo "Copying files..."
mkdir -p "${MNT}/EFI/BOOT" "${MNT}/boot/grub" "${MNT}/icons"

cp "${ROOT}/EFI/BOOT/BOOTX64.EFI"   "${MNT}/EFI/BOOT/"
cp "${ROOT}/EFI/BOOT/grubx64.efi"   "${MNT}/EFI/BOOT/"
cp "${ROOT}/EFI/BOOT/mmx64.efi"     "${MNT}/EFI/BOOT/"
cp "${ROOT}/EFI/BOOT/grub.cfg"      "${MNT}/EFI/BOOT/"
cp "${ROOT}/boot/vmlinuz"           "${MNT}/boot/"
cp "${ROOT}/boot/select.img"        "${MNT}/boot/"
cp "${ROOT}/boot/grub/grub.cfg"     "${MNT}/boot/grub/"
# Runtime-path assumption: the original initramfs and compiled Go /init are
# not available in this repository, so the expected config filename cannot
# be verified. Preserve the legacy on-image /gate.json path for compatibility;
# config/select.json is the source filename used by the SelectOS project.
cp "${ROOT}/config/select.json"     "${MNT}/gate.json"
cp "${ROOT}/icons/"*                "${MNT}/icons/" 2>/dev/null || true
touch "${MNT}/select.marker"

sync
umount "${MNT}"
rmdir "${MNT}"
losetup -d "${LOOP}"
trap - EXIT

echo
echo "==> Success: ${IMG}"
ls -lh "${IMG}"
echo
echo "Write to USB:"
echo "  sudo dd if=${IMG} of=/dev/sdX bs=4M status=progress oflag=sync"
echo "(Replace /dev/sdX with your actual USB device)"
