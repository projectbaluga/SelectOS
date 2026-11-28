# No-login runtime migration

## Product decision

The initial SelectOS release will not implement user authentication. Central LAN authentication is deferred to a future milestone. Boot and OS selection must work without internet, without a LAN, and without the Debian authentication server. No auth client, auth hook, auth API, or PostgreSQL dependency may be added to the initramfs or boot sequence for this milestone.

## What is true today

The current boot chain is:

1. UEFI loads the repository's prebuilt EFI GRUB binaries.
2. `EFI/BOOT/grub.cfg` searches for `/gate.marker` and loads `/boot/grub/grub.cfg`.
3. GRUB loads `/boot/vmlinuz` and `/boot/select.img`, passing `rdinit=/init`.
4. `boot/select.img` is copied from `BOOT/GATE.IMG` inside `gate-test.img.gz`.
5. The initramfs contains a prebuilt Gate/Gbox runtime. Its full source and a reproducible build recipe are not available in this repository.

The image builder only copies this binary. It cannot remove a login prompt embedded in the runtime. Changing the GRUB menu, JSON branding, marker file, or output filename does not remove that behavior.

## Current implementation in this change

- Authentication remains out of the boot image and boot sequence.
- The existing `auth/server` and `auth/client` source is retained for future implementation.
- The legacy image builder requires the explicit environment variable `SELECTOS_ALLOW_LEGACY_GATE_RUNTIME=1`.
- Legacy image artifacts carry `SELECTOS-BUILD-INFO.txt` identifying the embedded runtime as legacy and explicitly stating that login has not been removed.
- GitHub Actions labels the temporary artifact `SelectOS-legacy-Gate-runtime`.
- The build is deliberately not described as a login-free build.

## Required source to complete actual removal

The native runtime must replace the opaque initramfs, not wrap it. The implementation needs:

- Full source for the current Gate/Gbox `/init`, or a clean replacement implementation.
- Kernel/initramfs build inputs and reproducible packaging scripts.
- A documented hardware discovery and partition scanning strategy.
- OS detection and boot-entry parsing logic.
- UEFI variable / BootNext handling and reboot behavior, with safe fallbacks.
- The selector UI implementation and its assets.
- Signal handling and PID 1 responsibilities (mount proc/sysfs/devtmpfs, reap child processes, handle shutdown/reboot).
- A documented writable configuration and logging strategy that does not assume the ESP is writable.
- A build/test environment for x86_64 UEFI, plus additional architectures only if explicitly supported.

Do not attempt to patch a stripped binary by guessing offsets or injecting shell commands into the initramfs. Do not claim a login-free build until the runtime source or replacement is built into `boot/select.img`.

## Target boot contract

1. Firmware starts GRUB.
2. GRUB loads the kernel and native SelectOS initramfs.
3. Native PID 1 mounts required virtual filesystems and discovers disks.
4. Runtime reads SelectOS configuration and detects installed operating systems.
5. Runtime presents the OS selector directly. There is no username/password prompt and no network/auth-server wait.
6. User selection boots the selected OS through the supported UEFI mechanism.
7. If no OS is found or detection fails, show a useful recovery/diagnostic screen; do not fall into an authentication prompt.

The runtime must not require a writable EFI System Partition for normal operation. Any persistent settings must use a separately identified writable location and safe mount options.

## Acceptance tests for the actual login-free milestone

- [ ] Image builds reproducibly from source inputs without downloading the legacy Gate initramfs.
- [ ] Artifact inspection confirms the native SelectOS initramfs is packaged.
- [ ] Cold boot reaches the SelectOS selector with Ethernet disconnected and Wi-Fi unavailable.
- [ ] No Gbox/Gate login prompt appears.
- [ ] No Debian auth server, PostgreSQL, DNS, or internet dependency exists in the boot path.
- [ ] OS detection handles missing disks, unreadable partitions, and unsupported filesystems without panic or hang.
- [ ] UEFI boot selection works on test hardware and handles unavailable EFI variables safely.
- [ ] PID 1 reaps child processes and supports controlled reboot/shutdown.
- [ ] Recovery/diagnostic behavior is tested.
- [ ] The generated image is boot-tested on at least one physical UEFI machine and one virtual UEFI machine.

## Future central authentication milestone

When authentication is reintroduced, the Debian server remains the source of truth and is accessed over the local network. It must be an optional feature, not a prerequisite for booting the selector. Its outage must never prevent the initial no-login SelectOS boot. The exact future policy (per-user desktop login versus admin-only settings) should be designed separately before adding it.
