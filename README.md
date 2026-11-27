# SelectOS

Lightweight UEFI multi-boot selector.

SelectOS is intended to boot a minimal Linux environment, detect installed operating systems, and present a configurable menu for choosing an OS.

> **Build status:** the repository contains the image-building scripts and configuration, but does not currently contain the kernel, initramfs, or UEFI executable binaries. A bootable image cannot be produced until those real, compatible files are supplied. The build intentionally fails rather than creating placeholder binaries.

## Repository contents and build inputs

The image builder expects these files:

| Required path | Purpose |
|---|---|
| `boot/vmlinuz` | Linux kernel |
| `boot/select.img` | SelectOS initramfs, including its `/init` program |
| `EFI/BOOT/BOOTX64.EFI` | UEFI removable-media boot manager |
| `EFI/BOOT/grubx64.efi` | GRUB EFI executable |
| `EFI/BOOT/mmx64.efi` | MokManager executable, if required by the chosen GRUB build |

The repository currently tracks configuration and build scripts, not the source/build recipes needed to recreate all five binaries. Obtain them from a trusted SelectOS build or build them from their corresponding upstream sources and the project's own initramfs source. Do not substitute empty files, arbitrary EFI executables, or binaries from an unrelated image.

## Prepare the boot-binaries release asset

Create a gzip-compressed tar archive whose paths are relative to the repository root and exactly match the table above. The archive must contain all five files, not an enclosing directory.

Example archive layout:

```text
boot/vmlinuz
boot/select.img
EFI/BOOT/BOOTX64.EFI
EFI/BOOT/grubx64.efi
EFI/BOOT/mmx64.efi
```

After obtaining the real files, create `boot-binaries.tar.gz` and attach it to a GitHub Release in this repository. Name the asset exactly `boot-binaries.tar.gz`. The workflow input `binaries_release` must be `latest` or the exact release tag.

For maintainers using GitHub CLI from a trusted machine where the files are already prepared:

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

Keep the release asset private to the repository's intended distribution and verify the licensing and provenance of every included binary.

## Build with GitHub Actions

1. Open **Actions** in this repository.
2. Select **Build SelectOS image**.
3. Choose **Run workflow**.
4. Set `binaries_release` to `latest` or the exact tag containing `boot-binaries.tar.gz`.
5. Run the workflow and inspect its logs.
6. If successful, download the `SelectOS-image` artifact.

The workflow validates the archive and all required files before invoking the image builder. A missing release, missing archive member, corrupt archive, or empty file stops the job with an explicit error. The workflow does not manufacture boot binaries.

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

Edit `config/select.json` to change branding and operating-system presentation rules.

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

## Known implementation work

The current tracked `src/` directory does not yet provide the initramfs build recipe or the runtime `/init` implementation. Those are prerequisites for a reproducible end-to-end build. The GRUB configuration and image layout also need to be tested against the actual kernel/initramfs pair and target firmware before describing the image as production-ready.

## License

See [LICENSE](LICENSE). Kernel, GRUB, MokManager, and other third-party binaries retain their respective upstream licenses.
