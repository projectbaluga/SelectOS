# Documentation

## Architecture overview

1. **UEFI firmware** loads `EFI/BOOT/BOOTX64.EFI` (or `grubx64.efi`).
2. The small EFI GRUB config searches for `/select.marker` and loads the real menu from `/boot/grub/grub.cfg`.
3. GRUB boots the Linux kernel + initramfs (`select.img`).
4. The Go binary inside the initramfs becomes PID 1, mounts real disks, detects OSes, and presents a selection UI (branded via `select.json`).
5. On selection it sets EFI BootNext / reboots into the chosen OS (or uses kexec, depending on implementation).

## Runtime config path

The boot image currently receives the configuration as `/gate.json`. The source file is named `config/select.json` to match the SelectOS project branding; `scripts/build-image.sh` copies it to the legacy runtime filename for compatibility with the compiled init binary. The original initramfs and its Go `/init` binary are not available in this repository or its releases, so this runtime path remains an explicit compatibility assumption and has not been confirmed with `strings`. If the init binary is later inspected and expects a different path, update the destination in the build script and this note together.
