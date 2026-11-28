# Native SelectOS runtime — implementation status

## Current experimental scope

This branch introduces a source-controlled native initramfs/PID 1 instead of modifying the opaque Gate/Gbox binary. It remains a **bring-up prototype**, not a complete multi-boot release.

Implemented:
- Source-controlled PID 1 mounts proc, sysfs, devtmpfs, devpts, and volatile tmpfs, populates device nodes with BusyBox mdev, and reports detected block devices.
- PID 1 has no username/password prompt, auth client, LAN, internet, PostgreSQL, or server dependency.
- Reproducible initramfs packaging uses statically linked BusyBox and cpio newc + deterministic gzip. Output is separate under dist/native; the legacy boot/select.img is not overwritten. A compatible kernel must be supplied separately.
- `scripts/discover-efi-entries.py` enumerates partition nodes from sysfs and identifies EFI System Partitions by GPT type GUID. Default mode reads metadata only. Optional `--mount` uses `ro,nosuid,nodev,noexec` and checks a fixed list of known Windows/Linux EFI loader paths; it never executes them.
- `scripts/select-boot-entry.py` is a keyboard-driven menu prototype that consumes discovery JSON. It accepts only a numeric menu index and an explicit allowlist of loader paths, then prints a selection record. It does **not** execute a loader, modify UEFI variables, or chainload.
- Unit tests cover GPT GUID recognition, discovery edge cases, path allowlisting, and device path validation.

## Host-side prototype usage

On a Linux test machine with Python 3:

    python3 scripts/discover-efi-entries.py > /tmp/selectos-efi.json
    sudo python3 scripts/discover-efi-entries.py --mount > /tmp/selectos-efi.json
    python3 scripts/select-boot-entry.py --input /tmp/selectos-efi.json

The first scanner command only reads partition metadata. The second requires root and temporarily mounts discovered ESPs read-only. The selector is an interactive terminal prototype; it outputs a validated record only.

Run tests:

    python3 tests/test-efi-discovery.py
    python3 tests/test-boot-selector.py

## Experimental initramfs and image build

The separate UEFI image packager is documented in [EXPERIMENTAL-NATIVE-IMAGE.md](EXPERIMENTAL-NATIVE-IMAGE.md). It assembles a GPT/FAT32 image using the native diagnostic initramfs and marks the artifact `PRODUCTION_READY=false`. It is intentionally separate from the legacy image output.


On Debian/Ubuntu:

    sudo apt-get update
    sudo apt-get install -y busybox-static cpio gzip coreutils file
    bash scripts/build-native-initramfs.sh

Output: `dist/native/select-native.img`, `BUILD-INFO.txt`, and `SHA256SUMS`. This archive is not a complete bootable OS image and does not include a kernel.

## Optional Control Center connection

Control Center is not contacted by PID 1. The optional Linux agent runs only after the installed OS starts and reports inventory/health. It does not provide remote command execution, remote desktop, reboot/shutdown, or login synchronization. Server outage does not block local boot or use. See [Control Center connection guide](CONTROL-CENTER-CONNECT.md).

## Not implemented / not validated

1. The scanner and selector are host-side Python prototypes, not included in the initramfs and not invoked by PID 1.
2. Full filesystem and vendor-specific boot-entry discovery, architecture-aware loader selection, and robust duplicate-entry handling.
3. An audited UEFI handoff/chainloader, firmware fallback, Secure Boot handling, and persistent boot configuration.
4. QEMU/OVMF and physical UEFI validation of Windows and Linux boot.
5. Recovery UI and production-grade failure handling.

This code does **not** yet boot Windows or Linux. Do not flash or deploy it as a replacement for the current working image.

## Next engineering steps

- Replace host-side Python dependencies with a small static scanner suitable for the initramfs.
- Integrate discovery and selector into native PID 1 only after the static utility is tested.
- Implement a narrowly scoped UEFI handoff with strict path validation and explicit cancellation/fallback.
- Test GPT disk fixtures, malformed metadata, multiple ESPs, Secure Boot, QEMU/OVMF, and physical machines before release.

## Safety rules

- Never overwrite the existing boot/select.img from the experimental build script.
- Never partition, format, repair, or write to disks during discovery.
- Never execute discovered EFI files from the scanner or selector prototype.
- Never require the Control Center/server for initial boot.
- Never describe this milestone as a complete login-free multi-boot release.
