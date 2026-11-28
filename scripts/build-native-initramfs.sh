#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "\${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$ROOT/src/native-initramfs"
OUT="$ROOT/dist/native"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
for cmd in busybox cpio gzip sha256sum file find sort install grep; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "ERROR: Missing build dependency: $cmd" >&2
    echo "Debian/Ubuntu: sudo apt-get install busybox-static cpio gzip coreutils file" >&2
    exit 1
  }
done
[[ -s "$ROOT/boot/vmlinuz" ]] || { echo "ERROR: boot/vmlinuz is missing." >&2; exit 1; }
[[ -s "$SRC/init" ]] || { echo "ERROR: native /init source is missing." >&2; exit 1; }
BUSYBOX="$(command -v busybox)"
if ! file "$BUSYBOX" 2>/dev/null | grep -q 'statically linked'; then
  echo "ERROR: BusyBox must be statically linked (install busybox-static)." >&2
  exit 1
fi
mkdir -p "$WORK/root/bin" "$WORK/root/dev" "$WORK/root/proc" "$WORK/root/sys" "$WORK/root/run" "$WORK/root/tmp" "$WORK/root/mnt" "$OUT"
install -m 0755 "$BUSYBOX" "$WORK/root/bin/busybox"
ln -s busybox "$WORK/root/bin/sh"
install -m 0755 "$SRC/init" "$WORK/root/init"
for applet in sh mount mkdir mdev clear cat sleep; do
  "$WORK/root/bin/busybox" --list | grep -qx "$applet" || {
    echo "ERROR: BusyBox is missing required applet: $applet" >&2
    exit 1
  }
done
(
  cd "$WORK/root"
  find . -print0 | sort -z | cpio --null -o --format=newc --owner=0:0 2>"$WORK/cpio.log" | gzip -n -9 > "$OUT/select-native.img"
)
install -m 0644 "$ROOT/boot/vmlinuz" "$OUT/vmlinuz"
cat > "$OUT/BUILD-INFO.txt" <<'INFO'
PRODUCT=SelectOS
RUNTIME=native-initramfs-bringup
LOGIN_FREE=true
OS_SELECTOR=not-yet-implemented
UEFI_CHAINLOADER=not-yet-implemented
PRODUCTION_READY=false
WARNING=Experimental diagnostic initramfs only; does not yet select or boot operating systems.
INFO
(
  cd "$OUT"
  sha256sum vmlinuz select-native.img BUILD-INFO.txt > SHA256SUMS
)
echo "Native initramfs built:"
ls -lh "$OUT/vmlinuz" "$OUT/select-native.img" "$OUT/BUILD-INFO.txt"
echo "NOT a complete bootable SelectOS release. See docs/NATIVE-RUNTIME-STATUS.md."
