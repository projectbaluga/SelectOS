# Experimental native image build

This procedure creates a **separate experimental UEFI image** using the native
PID 1/initramfs bring-up prototype. It does not replace the legacy image and it
does not claim to boot Windows or Linux.

## Inputs

The build expects these source inputs to be staged in the repository:

- \`boot/vmlinuz\`: compatible x86_64 Linux kernel
- \`EFI/BOOT/BOOTX64.EFI\`, \`grubx64.efi\`, \`mmx64.efi\`
- GRUB configuration files under \`EFI/BOOT\` and \`boot/grub\`
- \`config/select.json\`
- \`dist/native/select-native.img\`, produced by \`scripts/build-native-initramfs.sh\`

The native initramfs build requires static BusyBox with the \`blkid\` applet,
cpio, gzip, coreutils, and file. The image packaging step additionally
requires parted, dosfstools, util-linux, and root privileges. The booted kernel
must support the target disk controller and FAT/VFAT filesystems; unsupported
hardware or filesystems are skipped without writes.

## Build

On Debian/Ubuntu:

\`\`\`sh
sudo apt-get update
sudo apt-get install -y busybox-static cpio gzip coreutils file parted dosfstools util-linux
sudo bash scripts/build-native-initramfs.sh
sudo SELECTOS_BUILD_NATIVE_EXPERIMENTAL=1 bash scripts/build-native-image.sh
\`\`\`

Output: \`dist/native/SelectOS-native-experimental.img\`.

The image builder refuses to overwrite an existing output. To choose another
path, set \`SELECTOS_NATIVE_IMAGE_OUT\`; to change the image size, set
\`SELECTOS_NATIVE_IMAGE_SIZE_MB\` (minimum 128 MiB). It never removes the
repository's \`dist\` directory or the legacy \`dist/SelectOS.img\`.

## Phase 2 behavior

At native PID 1 startup, SelectOS enumerates partition nodes in sysfs, asks
BusyBox \`blkid\` for the GPT partition type, and considers only the standard EFI
System Partition GUID. It mounts each candidate temporarily with
\`ro,nosuid,nodev,noexec\`, checks a fixed allowlist of common Windows/Linux EFI
loader paths, and unmounts it immediately. It does not repair, format, or write
to any partition.

If known loaders are found, a keyboard menu accepts a numeric index or \`q\`.
The selected item is printed as a diagnostic record. Keyboard input is never
treated as a path or shell command.

## Important limitations

The menu is integrated into the initramfs, but **selection does not boot the
chosen operating system yet**. Linux has already taken control from UEFI by
the time PID 1 runs; an EFI executable cannot safely be launched as if UEFI
Boot Services were still active. A separate, audited EFI-stage loader/handoff
is required. This build does not write \`BootNext\`, use efivarfs, change NVRAM,
or execute discovered EFI programs.

The scanner recognizes only a fixed list of common loader paths and depends on
kernel block-device and VFAT support. It does not yet parse vendor-specific
boot variables, Secure Boot policy, LUKS, RAID, or every possible EFI layout.
The image manifest explicitly marks the build as not production-ready.

Do not flash it as a replacement for a working multi-boot system. Validate it
with QEMU/OVMF and then representative physical UEFI hardware after a real
EFI-stage handoff implementation is available.

## Safety guarantees for this phase

- Discovery is read-only and ESP mounts include \`ro,nosuid,nodev,noexec\`.
- Only allowlisted loader filenames are displayed.
- Menu input is a numeric entry index or cancel; it is never executed.
- No UEFI variable, partition table, filesystem content, or installed OS is
  modified.
- Control Center, LAN, internet, authentication, and PostgreSQL are not boot
  dependencies.
