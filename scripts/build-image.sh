#!/usr/bin/env bash
set -Eeuo pipefail

# Transitional packager for the prebuilt Gate/Gbox runtime.
# This is NOT the future login-free SelectOS runtime.
if [[ "${SELECTOS_ALLOW_LEGACY_GATE_RUNTIME:-0}" != "1" ]]; then
  cat >&2 <<'EOF'
ERROR: The available initramfs is the legacy Gate/Gbox runtime and still
contains its legacy login/UI behavior. It cannot be made login-free by this
image packager. Refusing to create an image that could be mistaken for the
future no-login SelectOS build.

For an explicitly temporary legacy compatibility image only, set:
  SELECTOS_ALLOW_LEGACY_GATE_RUNTIME=1
See docs/NO-LOGIN-MIGRATION.md for the native runtime requirements.
EOF
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="${ROOT}/dist"
IMG="${DIST}/SelectOS.img"
SIZE_MB=512
LABEL="SELECTOS"
MNT=""
LOOP=""

echo "==> Building explicitly marked LEGACY Gate/Gbox compatibility image"

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
  if [[ ! -s "${ROOT}/${f}" ]]; then missing+=("$f"); fi
done
if (( ${#missing[@]} )); then
  printf 'ERROR: Required file missing or empty: %s\n' "${missing[@]}" >&2
  exit 1
fi
for cmd in dd parted losetup mkfs.vfat mount umount; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: Missing build dependency: $cmd" >&2; exit 1; }
done
if (( EUID != 0 )); then
  echo "ERROR: Run as root (sudo)." >&2
  exit 1
fi

cleanup() {
  set +e
  if [[ -n "$MNT" ]]; then umount "$MNT" 2>/dev/null || true; rmdir "$MNT" 2>/dev/null || true; fi
  if [[ -n "$LOOP" ]]; then losetup -d "$LOOP" 2>/dev/null || true; fi
}
trap cleanup EXIT

rm -rf "$DIST"
mkdir -p "$DIST"
cat > "$DIST/SELECTOS-BUILD-INFO.txt" <<'EOF'
PRODUCT=SelectOS
RUNTIME=legacy-gate-gbox
LOGIN_FREE=false
AUTHENTICATION=deferred-not-in-boot-path
WARNING=This image retains the legacy Gate/Gbox runtime and its login/UI behavior.
EOF
dd if=/dev/zero of="$IMG" bs=1M count="$SIZE_MB" status=none
parted -s "$IMG" mklabel msdos
parted -s "$IMG" mkpart primary fat32 1MiB 100%
parted -s "$IMG" set 1 boot on
parted -s "$IMG" set 1 esp on 2>/dev/null || true
LOOP="$(losetup -fP --show "$IMG")"
PART="${LOOP}p1"
mkfs.vfat -F 32 -n "$LABEL" "$PART"
MNT="$(mktemp -d)"
mount "$PART" "$MNT"

mkdir -p "$MNT/EFI/BOOT" "$MNT/boot/grub" "$MNT/icons"
cp "$ROOT/EFI/BOOT/BOOTX64.EFI" "$ROOT/EFI/BOOT/grubx64.efi" "$ROOT/EFI/BOOT/mmx64.efi" "$MNT/EFI/BOOT/"
cp "$ROOT/EFI/BOOT/grub.cfg" "$MNT/EFI/BOOT/"
cp "$ROOT/boot/vmlinuz" "$ROOT/boot/select.img" "$MNT/boot/"
cp "$ROOT/boot/grub/grub.cfg" "$MNT/boot/grub/"
# Legacy runtime compatibility paths: these do not remove the embedded login.
cp "$ROOT/config/select.json" "$MNT/gate.json"
touch "$MNT/gate.marker"
if compgen -G "$ROOT/icons/*" >/dev/null; then cp -r "$ROOT"/icons/. "$MNT/icons/"; fi
cp "$DIST/SELECTOS-BUILD-INFO.txt" "$MNT/SELECTOS-BUILD-INFO.txt"
sync
cleanup
trap - EXIT
[[ -s "$IMG" ]] || { echo "ERROR: Image was not created." >&2; exit 1; }
echo "==> Legacy compatibility image created: $IMG"
echo "WARNING: Gbox/Gate login has NOT been removed."
ls -lh "$IMG" "$DIST/SELECTOS-BUILD-INFO.txt"
