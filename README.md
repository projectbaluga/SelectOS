# SelectOS

Lightweight UEFI multi-boot selector, rebranded from the supplied legacy Gate image.

SelectOS boots a minimal Linux environment, detects installed operating systems, and presents a configurable menu for choosing an OS.

> **Build status:** the original image supplied for this project contains a real Linux kernel, Gate initramfs, and UEFI binaries. These have been extracted into the correctly named SelectOS build archive. The legacy initramfs remains a prebuilt binary; its original source/build recipe is not present in this repository.

## Boot components

The image builder expects these files:

| Required path | Purpose |
|---|---|
| `boot/vmlinuz` | Linux kernel (from the supplied legacy image) |
| `boot/select.img` | Legacy Gate initramfs, renamed for the SelectOS build layout |
| `EFI/BOOT/BOOTX64.EFI` | UEFI removable-media boot manager |
| `EFI/BOOT/grubx64.efi` | GRUB EFI executable |
| `EFI/BOOT/mmx64.efi` | MokManager executable |

The archive `boot-binaries.tar.gz` is prepared from the user-supplied `gate-test.img.gz`. Its five paths match the workflow's required archive layout. The archive must be attached to a GitHub Release as an asset named exactly `boot-binaries.tar.gz` before the cloud build can download it.

## Legacy runtime compatibility

The original initramfs is still the Gate runtime binary. Inspection confirms that it expects the on-image configuration file `/gate.json` and the marker file `/gate.marker`. Therefore the builder deliberately copies `config/select.json` to `gate.json` and creates `gate.marker`, while the user-facing branding and GRUB menu use SelectOS. Do not rename these two internal runtime paths unless the initramfs itself is rebuilt and tested.

The original initramfs also contains the legacy Gate application binary. The visible branding is supplied through `config/select.json`; this does not recompile or rename the internal program.

## Prepare or update the boot-binaries release

The prepared archive must contain these exact relative paths, with no enclosing directory:

```text
boot/vmlinuz
boot/select.img
EFI/BOOT/BOOTX64.EFI
EFI/BOOT/grubx64.efi
EFI/BOOT/mmx64.efi
```

Attach the prepared `boot-binaries.tar.gz` file to a GitHub Release in this repository and name the release asset exactly `boot-binaries.tar.gz`. The workflow input `binaries_release` must be `latest` or the exact release tag.

For future maintainers who have the extracted files available locally:

```bash
tar -czf boot-binaries.tar.gz boot/vmlinuz boot/select.img \
  EFI/BOOT/BOOTX64.EFI EFI/BOOT/grubx64.efi EFI/BOOT/mmx64.efi
tar -tzf boot-binaries.tar.gz
gh release create binaries boot-binaries.tar.gz --repo projectbaluga/SelectOS
```

To replace the asset on an existing release:

```bash
gh release upload binaries boot-binaries.tar.gz \
  --repo projectbaluga/SelectOS --clobber
```

## Build with GitHub Actions

1. Open **Actions** in this repository.
2. Select **Build SelectOS image**.
3. Choose **Run workflow**.
4. Set `binaries_release` to `latest` or the exact tag containing `boot-binaries.tar.gz`.
5. Run the workflow and inspect its logs.
6. If successful, download the `SelectOS-image` artifact.

The workflow validates the archive and all required files before invoking the image builder. A missing release, missing archive member, corrupt archive, or empty file stops the job with an explicit error.

## Build locally

On an Ubuntu/Debian system with the required binaries in place:

```bash
sudo apt-get update
sudo apt-get install -y parted dosfstools util-linux
sudo bash scripts/build-image.sh
```

This creates `dist/SelectOS.img`, a 512 MiB FAT32 UEFI image.

## Write to USB

**Warning:** writing an image erases the selected device. Verify the device path carefully before running this command.

```bash
sudo dd if=dist/SelectOS.img of=/dev/sdX bs=4M status=progress oflag=sync
```

Replace `/dev/sdX` with the correct USB device. Never guess the device name.

## Configuration

Edit `config/select.json` to change SelectOS branding and operating-system presentation rules. The legacy initramfs reads this file at runtime through the compatibility copy at `/gate.json`.

## Project structure

```text
SelectOS/
├── .github/workflows/build-image.yml
├── config/select.json
├── boot/grub/grub.cfg
├── EFI/BOOT/grub.cfg
├── scripts/build-image.sh
├── icons/
└── src/
```

## Known limitations and next steps

- The initramfs and its embedded Gate application are supplied as a prebuilt binary; their source and reproducible build recipe are not included.
- The legacy runtime still contains Gate-specific internal strings and compatibility paths. The JSON branding changes the displayed product name, but does not recompile the program.
- The image layout and firmware behavior still need a successful GitHub Actions build and a real UEFI boot test before calling the image production-ready.

## License

See [LICENSE](LICENSE). Kernel, GRUB, MokManager, and other third-party binaries retain their respective upstream licenses.
