#!/usr/bin/env bash
set -Eeuo pipefail

# Builds a separate, explicitly experimental image using the native diagnostic
# initramfs. It does not replace dist/SelectOS.img or the legacy boot image.
if [[ "${SELECTOS_BUILD_NATIVE_EXPERIMENTAL:-0}" != "1" ]]; then
  echo "ERROR: This image is experimental. Set SELECTOS_BUILD_NATIVE_EXPERIMENTAL=1 to continue." >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist/native"
OUT="${SELECTOS_NATIVE_IMAGE_OUT:-${DIST}/SelectOS-native-experimental.img}"
TMP_OUT="${OUT}.tmp.${BASHPID}"
SIZE_MB="${SELECTOS_NATIVE_IMAGE_SIZE_MB:-512}"
LABEL="SELECTOS"
MNT=""
LOOP=""
PART=""

required=(
  "${ROOT}/boot/vmlinuz"
  "${ROOT}/EFI/BOOT/BOOTX64.EFI"
  "${ROOT}/EFI/BOOT/grubx64.efi"
  "${ROOT}/EFI/BOOT/mmx64.efi"
  "${ROOT}/EFI/BOOT/grub.cfg"
  "${ROOT}/boot/grub/grub.cfg"
  "${ROOT}/config/select.json"
  "${DIST}/select-native.img"
)
for f in "${required[@]}"; do
  [[ -s "$f" ]] || { echo "ERROR: Required input missing or empty: $f" >&2; exit 1; }
done

for cmd in dd parted losetup mkfs.vfat mount umount sync file install sleep mv rm cp mkdir rmdir touch sha256sum; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: Missing build dependency: $cmd" >&2; exit 1; }
done
[[ "$SIZE_MB" =~ ^[0-9]+$ ]] && (( SIZE_MB >= 128 )) || {
  echo "ERROR: SELECTOS_NATIVE_IMAGE_SIZE_MB must be an integer >= 128." >&2
  exit 1
}
(( EUID == 0 )) || { echo "ERROR: Run this image-packaging step as root (sudo)." >&2; exit 1; }
mkdir -p "$(dirname "$OUT")"
[[ ! -e "$OUT" ]] || { echo "ERROR: Refusing to overwrite existing image: $OUT" >&2; exit 1; }

cleanup() {
  set +e
  if [[ -n "$MNT" ]]; then
    umount "$MNT" 2>/dev/null || true
    rmdir "$MNT" 2>/dev/null || true
  fi
  if [[ -n "$LOOP" ]]; then losetup -d "$LOOP" 2>/dev/null || true; fi
  if [[ -n "$TMP_OUT" && -f "$TMP_OUT" ]]; then rm -f -- "$TMP_OUT"; fi
}
trap cleanup EXIT

echo "==> Building separate experimental native SelectOS image"
dd if=/dev/zero of="$TMP_OUT" bs=1M count="$SIZE_MB" status=none
parted -s "$TMP_OUT" mklabel gpt
parted -s "$TMP_OUT" mkpart ESP fat32 1MiB 100%
parted -s "$TMP_OUT" set 1 esp on
LOOP="$(losetup -fP --show "$TMP_OUT")"
PART="${LOOP}p1"
for attempt in {1..10}; do
  [[ -b "$PART" ]] && break
  sleep 1
done
[[ -b "$PART" ]] || { echo "ERROR: Partition device did not appear: $PART" >&2; exit 1; }
mkfs.vfat -F 32 -n "$LABEL" "$PART"
MNT="$(mktemp -d)"
mount "$PART" "$MNT"

mkdir -p "$MNT/EFI/BOOT" "$MNT/boot/grub" "$MNT/icons"
install -m 0644 "${ROOT}/EFI/BOOT/BOOTX64.EFI" "$MNT/EFI/BOOT/BOOTX64.EFI"
install -m 0644 "${ROOT}/EFI/BOOT/grubx64.efi" "$MNT/EFI/BOOT/grubx64.efi"
install -m 0644 "${ROOT}/EFI/BOOT/mmx64.efi" "$MNT/EFI/BOOT/mmx64.efi"
install -m 0644 "${ROOT}/EFI/BOOT/grub.cfg" "$MNT/EFI/BOOT/grub.cfg"
install -m 0644 "${ROOT}/boot/vmlinuz" "$MNT/boot/vmlinuz"
install -m 0644 "${DIST}/select-native.img" "$MNT/boot/select.img"
install -m 0644 "${ROOT}/boot/grub/grub.cfg" "$MNT/boot/grub/grub.cfg"
install -m 0644 "${ROOT}/config/select.json" "$MNT/selectos.json"
touch "$MNT/gate.marker"
if [[ -d "${ROOT}/icons" ]]; then cp -r "${ROOT}/icons/." "$MNT/icons/"; fi
cat > "$MNT/SELECTOS-BUILD-INFO.txt" <<'INFO'
PRODUCT=SelectOS
RUNTIME=native-initramfs-bringup
LOGIN_FREE=true
BOOT_SELECTOR=not-implemented
OS_DISCOVERY=host-side-prototype-only
UEFI_CHAINLOADING=not-implemented
PRODUCTION_READY=false
WARNING=Experimental diagnostic image only; it does not yet select or boot installed operating systems.
INFO
sync
umount "$MNT"
rmdir "$MNT"
MNT=""
losetup -d "$LOOP"
LOOP=""
trap - EXIT
mv -- "$TMP_OUT" "$OUT"

[[ -s "$OUT" ]] || { echo "ERROR: Native image was not created." >&2; exit 1; }
cat > "${DIST}/IMAGE-BUILD-INFO.txt" <<'INFO'
PRODUCT=SelectOS
RUNTIME=native-initramfs-bringup
LOGIN_FREE=true
BOOT_SELECTOR=not-implemented
OS_DISCOVERY=host-side-prototype-only
UEFI_CHAINLOADING=not-implemented
PRODUCTION_READY=false
WARNING=Experimental diagnostic image only; it does not yet select or boot installed operating systems.
INFO
OUT_FOR_SUM="$OUT"
case "$OUT" in "$ROOT"/*) OUT_FOR_SUM="${OUT#"$ROOT"/}" ;; esac
(cd "$ROOT" && sha256sum "$OUT_FOR_SUM" "dist/native/IMAGE-BUILD-INFO.txt" > "dist/native/IMAGE-SHA256SUMS")
file "$OUT"
echo "Experimental native image created: $OUT"
echo "WARNING: diagnostic console only; not a production multi-boot image."
