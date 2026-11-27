# SelectOS

Lightweight UEFI multi-boot selector, rebranded from the supplied legacy Gate image.

SelectOS boots a minimal Linux environment, detects installed operating systems, and presents a configurable menu for choosing an OS.

> **Build status:** the original compressed disk image, `gate-test.img.gz`, is stored in this repository. GitHub Actions extracts the required boot files from it automatically before building the SelectOS image. The legacy initramfs remains a prebuilt binary; its original source/build recipe is not present in this repository.

## Boot components

The workflow extracts these files from the committed source image:

| Output path | Source path | Purpose |
|---|---|---|
| `boot/vmlinuz` | `BOOT/VMLINUZ` | Linux kernel |
| `boot/select.img` | `BOOT/GATE.IMG` | Legacy Gate initramfs, renamed for SelectOS layout |
| `EFI/BOOT/BOOTX64.EFI` | `EFI/BOOT/BOOTX64.EFI` | UEFI removable-media boot manager |
| `EFI/BOOT/grubx64.efi` | `EFI/BOOT/GRUBX64.EFI` | GRUB EFI executable |
| `EFI/BOOT/mmx64.efi` | `EFI/BOOT/MMX64.EFI` | MokManager executable |

No release asset or separate boot-binaries archive is required. Keep `gate-test.img.gz` in the repository root.

## Legacy runtime compatibility

The original initramfs is still the Gate runtime binary. It expects the on-image configuration file `/gate.json` and marker file `/gate.marker`. The builder therefore copies `config/select.json` to `gate.json` and creates `gate.marker`, while the user-facing branding and GRUB menu use SelectOS. Do not rename these internal compatibility paths unless the initramfs itself is rebuilt and tested.

The original initramfs also contains the legacy Gate application binary. The visible branding is supplied through `config/select.json`; this does not recompile or rename the internal program.

## Build with GitHub Actions

1. Open **Actions** in this repository.
2. Select **Build SelectOS image**.
3. Click **Run workflow**.
4. Run the workflow on the `main` branch.

The workflow checks and decompresses `gate-test.img.gz`, mounts its partition, extracts the five boot files, and runs the image builder. On success, it uploads an artifact named **SelectOS-image** containing `SelectOS.img` (retained for 14 days).

## Build locally

On an Ubuntu/Debian system, decompress and mount the source image, then place the five extracted files in the paths listed above. Install the build dependencies and run:

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
├── gate-test.img.gz
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
