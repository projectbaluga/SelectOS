# Documentation

## Current architecture

1. **UEFI firmware** loads `EFI/BOOT/BOOTX64.EFI` (or `grubx64.efi`).
2. `EFI/BOOT/grub.cfg` searches for `/gate.marker` and loads `/boot/grub/grub.cfg`.
3. GRUB boots the Linux kernel and initramfs (`/boot/select.img`) with `rdinit=/init`.
4. The current initramfs is a prebuilt legacy Gate/Gbox runtime. It detects operating systems and presents its selector UI. Its complete source and build recipe are not in this repository.
5. The current image builder copies the prebuilt runtime; it does not compile or modify its login/UI behavior.

## Authentication status

Authentication is deferred. The `auth/server` and `auth/client` directories are future-feature source only and are not part of the boot image or boot sequence. Initial SelectOS boot must work with no LAN and no internet.

## Runtime configuration compatibility

The legacy boot image receives the configuration as `/gate.json`; `scripts/build-image.sh` copies `config/select.json` to that compatibility path and creates `gate.marker`. These names are legacy runtime requirements, not the intended native SelectOS interface.

## No-login migration

The current prebuilt Gate/Gbox runtime may still display its legacy login. Removing that prompt requires replacing or rebuilding the initramfs from source. See [NO-LOGIN-MIGRATION.md](NO-LOGIN-MIGRATION.md) for the migration contract and acceptance tests.

## Server management

See [CONTROL-CENTER.md](CONTROL-CENTER.md) for the Debian LAN dashboard, device agent, API, TLS reverse proxy, backups, and restore operations.

## Branding

Logo files, colours and usage rules: [BRANDING.md](BRANDING.md).
