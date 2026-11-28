#!/bin/bash
# Builds gate-test.img (GPT + FAT32, UEFI + Secure Boot capable) using Ubuntu's signed shim/grub/kernel.
# Usage: sudo src/gate-runtime/tools/build-image.sh [serial]   ("serial" adds console=ttyS0 for QEMU tests)
set -euo pipefail
cd "$(dirname "$0")/.."
REPO="$(cd ../.. && pwd)"
KVER=6.8.0-151-generic
W="$(mktemp -d "${TMPDIR:-/tmp}/gate-build.XXXXXX")"
trap 'rm -rf -- "$W"' EXIT
mkdir -p ""$W"/k" "$W/rd"/{bin,dev,proc,sys,modules,etc,tmp} "$W/esp"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$W/rd/init" ./gate
( cd "$W/k" && apt-get download linux-image-$KVER linux-modules-$KVER busybox-static >/dev/null 2>&1; for f in *.deb; do dpkg -x $f ex; done )
K="$W/k/ex"
cp $K/boot/vmlinuz-$KVER $W/vmlinuz
cp $K/bin/busybox $W/rd/bin/busybox 2>/dev/null || cp $(ls $K/usr/bin/busybox $K/bin/busybox 2>/dev/null | head -1) $W/rd/bin/busybox
ln -s busybox $W/rd/bin/sh
# kernel modules the Gate needs (USB keyboard, USB/SATA/NVMe disks to read gate.json), with dependencies
depmod -b $K $KVER
: > $W/rd/modules/order
for m in hid-generic usbhid xhci-pci-renesas xhci-pci usb-storage uas libahci ahci nvme-core nvme; do
  while IFS= read -r f; do
    b=$(basename $f .ko.zst)
    if ! grep -qx "$b" $W/rd/modules/order; then
      zstd -dq $f -o $W/rd/modules/$b.ko
      echo $b >> $W/rd/modules/order
    fi
  done < <(modprobe -d "$K" -S "$KVER" --show-depends "$m" 2>/dev/null | awk '$1=="insmod"{print $2}')
done
echo "modules: $(tr '\n' ' ' < $W/rd/modules/order)"
mknod -m 600 $W/rd/dev/console c 5 1
( cd $W/rd && find . | cpio -o -H newc --quiet | zstd -19 -q > $W/gate.img )
# ESP contents
mkdir -p $W/esp/EFI/BOOT $W/esp/boot/grub
cp /usr/lib/shim/shimx64.efi.dualsigned    $W/esp/EFI/BOOT/BOOTX64.EFI   # signed by both Microsoft 2011 and 2023 CAs
cp /usr/lib/grub/x86_64-efi-signed/grubx64.efi.signed $W/esp/EFI/BOOT/grubx64.efi
cp /usr/lib/shim/mmx64.efi.signed $W/esp/EFI/BOOT/mmx64.efi 2>/dev/null || cp /usr/lib/shim/mmx64.efi $W/esp/EFI/BOOT/mmx64.efi 2>/dev/null || true
cp $W/vmlinuz $W/gate.img $W/esp/boot/
touch $W/esp/gate.marker
# configuration: the repository's config/select.json (as gate.json) and icons/, unless CONFIG=<dir> is given
CFG="${CONFIG:-}"
if [ -z "$CFG" ]; then
  CFG=$W/cfg; mkdir -p $CFG
  cp $REPO/config/select.json $CFG/gate.json
  [ -d $REPO/icons ] && cp -r $REPO/icons $CFG/icons
fi
cp -r $CFG/. $W/esp/
CMD="quiet loglevel=3 console=tty0"; [ "${1:-}" = serial ] && CMD="loglevel=3 console=tty0 console=ttyS0"
CMD="$CMD ${EXTRA:-}"
cat > $W/esp/EFI/BOOT/grub.cfg <<G
search --no-floppy --file --set=root /gate.marker
set prefix=(\$root)/boot/grub
configfile /boot/grub/grub.cfg
G
cat > $W/esp/boot/grub/grub.cfg <<G
set timeout=2
set default=0
menuentry "NCF CCS Lab Gate" {
    linux /boot/vmlinuz $CMD rdinit=/init
    initrd /boot/gate.img
}
G
# disk image: classic MBR + one bootable FAT32 partition (same layout Windows/Rufus UEFI USBs use;
# no GPT backup header, so it works no matter how big the USB stick is). FS size == partition size.
IMG="${OUT:-$REPO/dist/gate-test.img}"
mkdir -p "$(dirname "$IMG")"
if [ -e "$IMG" ]; then
  echo "Refusing to overwrite existing image: $IMG" >&2
  echo "Choose a new OUT= path or move the existing image first." >&2
  exit 1
fi
TOTAL=1048576           # 512 MiB in 512B sectors (>=256MiB so FAT32 uses normal 4K clusters)
START=2048
SIZE=$((TOTAL - START))
truncate -s $((TOTAL*512)) $IMG
sfdisk -q $IMG <<S
label: dos
start=$START, size=$SIZE, type=c, bootable
S
mkfs.fat -F 32 -s 8 -n GATE -S 512 -h $START --offset $START $IMG $((SIZE/2)) >/dev/null
mcopy -i "$IMG"@@1M -s $W/esp/* ::
echo "Built $IMG"
[ -n "${ZIP:-}" ] && (cd $W/esp && python3 -c "import shutil,sys;shutil.make_archive(sys.argv[1],'zip','.')" "$ZIP") || true
