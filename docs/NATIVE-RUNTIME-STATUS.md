# Native SelectOS runtime — implementation status

## Scope of the first implementation

This branch introduces a source-controlled native initramfs/PID 1 instead of modifying the opaque Gate/Gbox binary. It is deliberately a **bring-up diagnostic runtime**, not a complete multi-boot selector.

Implemented:
- Source-controlled PID 1 startup script.
- Mounts proc, sysfs, devtmpfs, devpts, and volatile tmpfs filesystems with safe fallbacks.
- Populates device nodes with BusyBox mdev where available.
- Displays SelectOS branding and reports detected block devices.
- No username/password prompt, auth client, LAN, internet, PostgreSQL, or server dependency.
- Reproducible initramfs packaging using statically linked BusyBox and cpio newc + deterministic gzip.
- Separate experimental artifacts under dist/native/; the legacy boot/select.img and current image builder are not overwritten.
- Build manifest and SHA-256 checksums explicitly mark the runtime as not production-ready.

## Phase 2 prototype — GPT ESP and loader discovery

`scripts/discover-efi-entries.py` is a host-side, read-only discovery prototype. It:
- Enumerates partition nodes from sysfs and asks `blkid` for partition metadata.
- Identifies EFI System Partitions by the GPT type GUID (not by partition number or label).
- By default, reads metadata only and does not mount or modify disks.
- With explicit `--mount`, temporarily mounts each detected ESP using `ro,nosuid,nodev,noexec`, checks a bounded allowlist of common Windows/Linux EFI loader paths, and unmounts it.
- Emits JSON suitable for a future selector UI.

Run on a Linux test machine:

    python3 scripts/discover-efi-entries.py
    python3 scripts/discover-efi-entries.py --mount

The optional mount mode requires root and working `blkid`, `mount`, and `umount`. It never mounts read-write and does not execute discovered EFI files. This prototype is **not wired into PID 1** and is not yet part of the initramfs image.

Run its source tests:

    python3 -m unittest tests/test-efi-discovery.py

## Build the experimental initramfs

On Debian/Ubuntu:

    sudo apt-get update
    sudo apt-get install -y busybox-static cpio gzip coreutils file
    bash scripts/build-native-initramfs.sh

Output:
- dist/native/select-native.img
- dist/native/BUILD-INFO.txt
- dist/native/SHA256SUMS

The script does not write a USB, replace boot/select.img, or touch the existing legacy artifact. It does not include a kernel; the image build must supply a compatible kernel separately.

## Optional Control Center connection

Control Center is not contacted by PID 1. To connect an installed Linux SelectOS PC after its operating system starts, use the guided agent installer from a repository checkout. See [Connect a SelectOS Linux PC to Control Center](CONTROL-CENTER-CONNECT.md).

The agent reports inventory/health only. It does not provide remote command execution, remote desktop, reboot/shutdown, or login synchronization. Server outage does not block local boot or use.

## What is not implemented yet

1. Integrating the ESP scanner into the native initramfs (including the required static tooling).
2. Full Windows/Linux boot-entry parsing, including vendor-specific layouts and architecture-aware fallback handling.
3. Keyboard-driven OS selector and recovery UI.
4. UEFI handoff/chainloading and fallback behavior.
5. Persistent configuration and controlled power actions.
6. QEMU/OVMF and physical UEFI boot validation.

This milestone does **not** prove SelectOS can boot another operating system. Do not flash or deploy it as a replacement for the current working image.

## Next phases

- Add fixture-backed tests for actual GPT images and more filesystems.
- Port the bounded, read-only scanner into a small static initramfs utility.
- Build a keyboard-accessible selector whose choices are restricted to enumerated loader paths.
- Implement and test a narrowly scoped EFI handoff, including Secure Boot implications.
- Validate Windows and Linux selection, offline operation, and recovery on QEMU/OVMF and physical UEFI hardware before release.

## Safety rules

- Never overwrite boot/select.img from the experimental build script.
- Never partition, format, repair, or write to disks during discovery.
- Never execute a discovered loader from the scanner.
- Never require the server for initial boot.
- Never describe this milestone as a complete login-free multi-boot release.
