# Native SelectOS runtime — implementation status

## Scope of this first implementation

This branch introduces the first native initramfs/PID 1 source instead of modifying the opaque Gate/Gbox binary. It is deliberately a **bring-up diagnostic runtime**, not a complete multi-boot selector.

Implemented:
- A source-controlled /init script that runs as PID 1.
- Mounts proc, sysfs, devtmpfs, devpts, and volatile tmpfs filesystems with safe fallbacks.
- Populates device nodes with BusyBox mdev where available.
- Displays SelectOS branding and reports detected block devices.
- Has no username/password prompt, auth client, LAN dependency, internet dependency, PostgreSQL, or server calls.
- Reproducible initramfs packaging using a statically linked BusyBox and cpio newc + deterministic gzip.
- Separate experimental artifacts under dist/native/; the legacy boot/select.img and current image builder are not overwritten.
- Build manifest and SHA-256 checksums explicitly mark the runtime as not production-ready.

## Build on Debian/Ubuntu

    sudo apt-get update
    sudo apt-get install -y busybox-static cpio gzip coreutils file
    bash scripts/build-native-initramfs.sh

Output:
- dist/native/select-native.img
- dist/native/BUILD-INFO.txt
- dist/native/SHA256SUMS

The script does not write a USB, replace boot/select.img, or touch the existing legacy artifact.

## Control Center connection

The Control Center remains optional and is not contacted by PID 1. To connect an installed Linux SelectOS PC after its operating system starts, use the guided agent installer from a repository checkout: install Go, then run `sudo bash auth/agent/install-control-center-agent.sh` from the repository root. It asks for the HTTPS Control Center URL, a unique device ID, and the enrollment token without echoing the secret. The agent runs as a restricted systemd service and sends heartbeats every 30 seconds. Full details are in [Connect a SelectOS Linux PC to Control Center](CONTROL-CENTER-CONNECT.md).

This agent reports inventory/health only. It does not provide remote command execution, remote desktop, reboot/shutdown, or login synchronization. Server outage does not block local boot or use.

## What is not implemented yet

This is the honest boundary of the current milestone:
1. Filesystem/partition probing beyond listing kernel block devices.
2. Windows/Linux boot-entry discovery and safe parsing.
3. A graphical or keyboard-driven OS selector.
4. UEFI chainloading / BootNext integration and fallback behavior.
5. Recovery UI, persistent configuration, and controlled power actions.
6. Physical UEFI and virtual UEFI boot validation.

Therefore this initramfs proves a native, login-free PID 1 can be assembled from repository source, but it does **not** yet prove that SelectOS can boot another operating system. Do not flash or deploy it as a replacement for the current working image.

## Next implementation phases

### Phase 2 — hardware and filesystem discovery
- Add read-only partition discovery using kernel sysfs and blkid/udev-compatible tooling.
- Identify EFI System Partitions by GPT type GUID, not device-name assumptions.
- Mount candidate ESPs read-only and enumerate Microsoft boot manager, Linux shim/GRUB/systemd-boot entries, and fallback loaders.
- Never write to the ESP during discovery.
- Add fixtures for absent disks, malformed paths, duplicate labels, and unreadable partitions.

### Phase 3 — selector
- Build a keyboard-accessible console selector first; graphical UI is a separate decision.
- Show detected OS name, loader path, disk identity, and clear recovery state.
- Validate every selected loader path against an enumerated ESP; never accept arbitrary shell input as a boot path.
- Keep the selector usable with no network and no writable disk.

### Phase 4 — UEFI handoff
- Implement and test a narrowly scoped EFI application or a GRUB menu-generation handoff.
- Prefer firmware-supported boot entries where available; provide a tested fallback when EFI variables are unavailable.
- Do not assume efibootmgr works from every firmware/runtime.
- Test secure boot implications and architecture compatibility before release.

### Phase 5 — release validation
- QEMU/OVMF boot tests plus physical UEFI hardware tests.
- Confirm no login prompt, no network requirement, successful selection of Windows and Linux, and recovery when disks or EFI entries are missing.
- Only then consider replacing the legacy initramfs in the release image.

## Safety rules

- Never overwrite boot/select.img from this experimental build script.
- Never run destructive partitioning or filesystem repair as part of discovery.
- Never require the server for initial boot.
- Never describe this milestone as a complete login-free multi-boot release.
