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
| `EFI/BOOT/grubx64.efi`  | GRUB EFI                        | ~2.6 MB     |
| `EFI/BOOT/mmx64.efi`    | MokManager (Secure Boot)        | ~0.8 MB     |

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

For a manual cloud build, open **Actions → Build SelectOS image → Run workflow**, then download the `SelectOS.img` artifact from the completed run. This workflow runs only when manually triggered. The required kernel, initramfs, and EFI binaries must already be present in the repository; otherwise, the workflow stops with a clear error. The local build above remains the primary path.

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
