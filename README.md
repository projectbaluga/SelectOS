# SelectOS

A lightweight UEFI-bootable multi-boot selector.

SelectOS boots a minimal Linux environment that detects installed operating systems and presents a clean, configurable menu so users can choose which OS to start.

## Features

- UEFI boot (with Secure Boot / MokManager support)
- Simple JSON-based branding and OS presentation rules
- Very small initramfs (BusyBox + storage drivers + custom Go binary)
- Designed for multi-boot machines and lab environments

## Project layout

```
SelectOS/
├── README.md
├── config/
│   └── gate.json          # Branding & OS display rules (edit this)
├── boot/
│   ├── vmlinuz            # Linux kernel
│   ├── gate.img           # Initramfs containing the Go "init"
│   └── grub/
│       └── grub.cfg
├── EFI/BOOT/
│   ├── BOOTX64.EFI
│   ├── grubx64.efi
│   ├── mmx64.efi
│   └── grub.cfg
├── icons/
├── scripts/
│   └── build-image.sh     # Rebuilds the bootable .img
└── src/                   # Future source of the init binary
```

## Quick rebuild

1. Place the binary components into the tree:
   - `boot/vmlinuz`
   - `boot/gate.img`
   - `EFI/BOOT/*.efi`

2. Run:
```bash
./scripts/build-image.sh
```

This produces `dist/SelectOS.img` (512 MiB FAT32 image).

### Obtaining the binary components

Extract them from the original image:

```bash
gunzip -c gate-test.img.gz > gate-test.img
sudo losetup -fP gate-test.img
sudo mount /dev/loopXp1 /mnt
cp /mnt/boot/vmlinuz boot/
cp /mnt/boot/gate.img boot/
cp /mnt/EFI/BOOT/*.efi EFI/BOOT/
```

## Configuration

Edit `config/gate.json`:

- `brand` – name and subtitle shown in the UI
- `os` – rules to rename, reorder, or assign icons to detected operating systems
- `hide` – patterns to hide
- `default` – default selected entry

## Writing the image

```bash
sudo dd if=dist/SelectOS.img of=/dev/sdX bs=4M status=progress oflag=sync
```

**Double-check the target device!**

## License

Configuration files, scripts and documentation are under the MIT License.  
The Linux kernel, GRUB binaries and the Go init binary retain their original licenses.
