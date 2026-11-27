# SelectOS

Lightweight UEFI multi-boot selector.

SelectOS boots a minimal Linux environment, detects installed operating systems, and presents a clean configurable menu so the user can choose which OS to start.

## Features

- Full UEFI support (including Secure Boot / MokManager)
- Simple JSON configuration for branding and OS presentation
- Tiny initramfs (BusyBox + essential drivers + custom Go binary)
- Easy to rebuild and deploy to USB

## Requirements to build

You need these binary components (not included in the git repository because of size):

| File                    | Description                     | Approx size |
|-------------------------|---------------------------------|-------------|
| `boot/vmlinuz`          | Linux kernel                    | ~15 MB      |
| `boot/select.img`       | Initramfs (contains the Go init)| ~2.2 MB     |
| `EFI/BOOT/BOOTX64.EFI`  | UEFI bootloader                 | ~1 MB       |
| `EFI/BOOT/grubx64.efi` | GRUB EFI                        | ~2.6 MB     |
| `EFI/BOOT/mmx64.efi`   | MokManager (Secure Boot)        | ~0.8 MB     |

These can be extracted from the original `select-test.img`:

```bash
gunzip -c select-test.img.gz > select-test.img
sudo losetup -fP --show select-test.img   # note the loop device
sudo mount /dev/loopXp1 /mnt

mkdir -p boot EFI/BOOT
cp /mnt/boot/vmlinuz          boot/
cp /mnt/boot/select.img       boot/
cp /mnt/EFI/BOOT/*.efi        EFI/BOOT/
```

## Quick build

```bash
# Make sure the binary files above are in place, then:
./scripts/build-image.sh
```

This creates `dist/SelectOS.img` (512 MB FAT32 UEFI image).

## Building via GitHub Actions

For a manual cloud build, open **Actions → Build SelectOS image → Run workflow**, then download the `SelectOS.img` artifact from the completed run. This workflow runs only when manually triggered.

The workflow downloads `boot-binaries.tar.gz` from a GitHub Release and extracts it into the repository before checking the required files. The `binaries_release` input selects the release tag; it defaults to `latest`. Set it to a specific tag (for example, `binaries`) when the latest release is not the one containing the boot binaries. If the download or extraction fails, the required-files check stops the build with a clear error.

### Creating or updating the binaries release

1. Extract the kernel, initramfs, and EFI files from the test image using the commands in [Requirements to build](#requirements-to-build), so the files exist at `boot/vmlinuz`, `boot/select.img`, and `EFI/BOOT/*.efi`.
2. From the repository root, create the archive:

   ```bash
   tar czf boot-binaries.tar.gz boot/vmlinuz boot/select.img EFI/BOOT/*.efi
   ```

3. Create the release and upload the asset:

   ```bash
   gh release create binaries boot-binaries.tar.gz
   ```

   To update an existing release instead, upload the new archive:

   ```bash
   gh release upload binaries boot-binaries.tar.gz --clobber
   ```

The workflow's `GITHUB_TOKEN` needs permission to read repository releases. The local build above remains available as the primary path.

## Write to USB

```bash
sudo dd if=dist/SelectOS.img of=/dev/sdX bs=4M status=progress oflag=sync
```

**Warning:** Double-check the target device (`/dev/sdX`). This will erase everything on it.

## Configuration

Edit `config/select.json`:

```json
{
  "brand": { "name": "SelectOS", "subtitle": "Multi-boot Selector" },
  "os": [
    { "match": "windows", "name": "Windows 11", "subtitle": "Microsoft Windows", "order": 1 },
    { "match": "ubuntu",  "name": "Ubuntu",     "subtitle": "Ubuntu Linux",      "order": 2 }
  ]
}
```

- New operating systems appear automatically.
- Only add rules when you want to rename, reorder, hide, or assign a custom icon.

## Project structure

```
SelectOS/
├── config/select.json       ← main configuration
├── boot/select.img          ← initramfs (provided before build)
├── boot/grub/grub.cfg       ← GRUB menu
├── EFI/BOOT/grub.cfg        ← EFI chainloader
├── scripts/build-image.sh   ← image builder
├── icons/                   ← optional icons
└── src/                     ← future Go source
```

## License

Scripts, configuration and documentation: MIT License.  
Kernel, GRUB binaries and the Go init binary retain their original licenses.
