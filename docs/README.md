# Documentation

## Architecture overview

1. **UEFI firmware** loads `EFI/BOOT/BOOTX64.EFI` (or `grubx64.efi`).
2. The small EFI GRUB config searches for `/gate.marker` and loads the real menu from `/boot/grub/grub.cfg`.
3. GRUB boots the Linux kernel + initramfs (`gate.img`).
4. The Go binary inside the initramfs becomes PID 1, mounts real disks, detects OSes, and presents a selection UI (branded via `gate.json`).
5. On selection it sets EFI BootNext / reboots into the chosen OS (or uses kexec, depending on implementation).
