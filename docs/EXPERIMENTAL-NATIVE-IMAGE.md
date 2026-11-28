# Experimental native image build

This procedure creates a **separate experimental UEFI image** using the native
PID 1/initramfs bring-up prototype. It does not replace the legacy image and it
does not claim to boot Windows or Linux.

## Inputs

The build expects these source inputs to be staged in the repository:

- `boot/vmlinuz`: compatible x86_64 Linux kernel
- `EFI/BOOT/BOOTX64.EFI`, `grubx64.efi`, `mmx64.efi`
- GRUB configuration files under `EFI/BOOT` and `boot/grub`
- `config/select.json`
- `dist/native/select-native.img`, produced by `scripts/build-native-initramfs.sh`

The native initramfs build requires static BusyBox, cpio, gzip, coreutils, and
file. The image packaging step additionally requires parted, dosfstools,
util-linux, and root privileges.

## Build

On Debian/Ubuntu:

```sh
sudo apt-get update
sudo apt-get install -y busybox-static cpio gzip coreutils file parted dosfstools util-linux
bash scripts/build-native-initramfs.sh
sudo SELECTOS_BUILD_NATIVE_EXPERIMENTAL=1 bash scripts/build-native-image.sh
```

Output: `dist/native/SelectOS-native-experimental.img`.

The image builder refuses to overwrite an existing output. To choose another
path, set `SELECTOS_NATIVE_IMAGE_OUT`; to change the image size, set
`SELECTOS_NATIVE_IMAGE_SIZE_MB` (minimum 128 MiB). It never removes the
repository's `dist` directory or the legacy `dist/SelectOS.img`.

## Current behavior and limitations

The kernel starts the native `/init` and displays a diagnostic console. This
proves only that the experimental kernel/initramfs packaging path can be
assembled. The current initramfs does not yet include the host-side Python EFI
scanner or selector, does not implement a graphical/keyboard OS menu, and does
not chainload an installed operating system.

The image manifest explicitly marks the build as not production-ready. Do not
flash it as a replacement for a working multi-boot system. Validate it first
with QEMU/OVMF, then representative physical UEFI hardware after the runtime
and handoff implementation is complete.

## Safety

- No partition is created, formatted, repaired, or written on a host disk.
  The script creates a file-backed image and formats only its loop partition.
- Existing legacy image outputs are not overwritten.
- Discovery remains read-only; discovered EFI executables are never run by
  the scanner.
- Control Center, LAN, internet, authentication, and PostgreSQL are not boot
  dependencies.
