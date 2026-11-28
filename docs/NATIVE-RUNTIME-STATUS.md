# Native SelectOS runtime — implementation status

## Current experimental scope

This branch introduces a source-controlled native initramfs/PID 1 instead of
modifying the opaque Gate/Gbox binary. It remains a **bring-up prototype**, not
a complete multi-boot release.

Implemented:
- Source-controlled PID 1 mounts proc, sysfs, devtmpfs, devpts, and volatile
  tmpfs, populates device nodes with BusyBox mdev, and reports block devices.
- PID 1 has no username/password prompt, auth client, LAN, internet,
  PostgreSQL, or server dependency.
- Reproducible initramfs packaging uses statically linked BusyBox and cpio
  newc + deterministic gzip. Output is separate under dist/native; the legacy
  boot/select.img is not overwritten. A compatible kernel is supplied
  separately.
- A small BusyBox-shell EFI scanner is included in the initramfs. It reads
  partition metadata, recognizes the GPT EFI System Partition GUID, mounts
  candidate ESPs only with \`ro,nosuid,nodev,noexec\`, checks fixed allowlisted
  loader paths, and unmounts immediately.
- A keyboard menu accepts only a numeric index or cancel. It prints a
  diagnostic selection record; it never executes the EFI file or changes
  firmware variables.
- Host-side Python discovery and selector prototypes remain available for
  offline fixture testing and structured JSON output.
- Unit/regression tests cover GPT GUID recognition, host selector path
  allowlisting, initramfs shell menu selection, and builder safety.

## Host-side prototype usage

On a Linux test machine with Python 3:

    python3 scripts/discover-efi-entries.py > /tmp/selectos-efi.json
    sudo python3 scripts/discover-efi-entries.py --mount > /tmp/selectos-efi.json
    python3 scripts/select-boot-entry.py --input /tmp/selectos-efi.json

The host scanner is a separate prototype. The native PID 1 uses the BusyBox
shell implementation included in the initramfs.

Run tests:

    python3 tests/test-efi-discovery.py
    python3 tests/test-boot-selector.py
    bash tests/test-native-efi-menu.sh

## Experimental initramfs and image build

See [EXPERIMENTAL-NATIVE-IMAGE.md](EXPERIMENTAL-NATIVE-IMAGE.md).

On Debian/Ubuntu:

    sudo apt-get update
    sudo apt-get install -y busybox-static cpio gzip coreutils file
    sudo bash scripts/build-native-initramfs.sh

Output: \`dist/native/select-native.img\`, \`BUILD-INFO.txt\`, and \`SHA256SUMS\`.
This archive is not a complete bootable OS image and does not include a kernel.

## Optional Control Center connection

Control Center is not contacted by PID 1. The optional Linux agent runs only
after the installed OS starts and reports inventory/health. It does not provide
remote command execution, remote desktop, reboot/shutdown, or login
synchronization. Server outage does not block local boot or use. See
[Control Center connection guide](CONTROL-CENTER-CONNECT.md).

## Not implemented / not validated

1. UEFI-stage boot selector and handoff. PID 1 runs after UEFI Boot Services
   have ended; it cannot directly execute an EFI application.
2. Firmware boot-variable enumeration, persistent boot configuration,
   Secure Boot policy, recovery UI, and production-grade failure handling.
3. Complete vendor-specific loader discovery, architecture-specific paths
   beyond the initial allowlist, encrypted/RAID filesystems, and duplicate
   entry resolution.
4. QEMU/OVMF and physical UEFI validation of Windows and Linux boot.

This code does **not** yet boot Windows or Linux. Do not flash or deploy it as
a replacement for the current working image.

## Next engineering steps

- Build a small EFI application that runs before ExitBootServices and owns the
  actual menu/handoff, with a signed/Secure Boot strategy.
- Define a strict device-path and loader allowlist and implement explicit
  cancellation and firmware fallback.
- Add QEMU/OVMF disk fixtures and automated boot tests, then test representative
  physical UEFI hardware before release.

## Safety rules

- Never overwrite the existing boot/select.img from the experimental build.
- Never partition, format, repair, or write to disks during discovery.
- Never execute discovered EFI files from the Linux PID 1 menu.
- Never write BootNext or efivarfs in this phase.
- Never require the Control Center/server for initial boot.
- Never describe this milestone as a complete login-free multi-boot release.
