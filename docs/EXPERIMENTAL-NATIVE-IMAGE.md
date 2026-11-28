# Experimental native SelectOS image

This procedure builds a separate experimental x86_64 UEFI image. It does not
replace the legacy image or claim production readiness.

## Build dependencies

On Debian/Ubuntu:

\`\`\`sh
sudo apt-get update
sudo apt-get install -y gcc binutils gnu-efi busybox-static cpio gzip coreutils file parted dosfstools util-linux
sudo bash scripts/build-uefi-loader.sh
sudo bash scripts/build-native-initramfs.sh
sudo SELECTOS_BUILD_NATIVE_EXPERIMENTAL=1 bash scripts/build-native-image.sh
\`\`\`

The UEFI application is emitted as \`dist/native/selectos.efi\`. The native
image is emitted as \`dist/native/SelectOS-native-experimental.img\`. Both
builders refuse to overwrite existing outputs. Set
\`SELECTOS_UEFI_LOADER_OUT\` or \`SELECTOS_NATIVE_IMAGE_OUT\` to choose a
different path.

## UEFI-stage boot manager (experimental)

The GRUB menu includes an explicit **SelectOS UEFI Boot Manager
(experimental)** entry. The existing native Linux diagnostic boot remains the
default. Choosing the UEFI entry chainloads
\`EFI/SelectOS/selectos.efi\` while UEFI Boot Services are still available.

The application enumerates UEFI Simple File System volumes, opens only a
fixed allowlist of common loader paths read-only, and presents a firmware
keyboard menu. On selection it creates a file device path and calls UEFI
\`LoadImage\` and \`StartImage\` to hand control to the selected loader. It
does not pass input to a shell, construct arbitrary paths, change
\`BootOrder\`/\`BootNext\`, write NVRAM variables, or call \`ExitBootServices\`.
If a child loader returns, the menu is shown again.

This is an initial handoff implementation, not a complete production boot
manager. It scans UEFI-readable filesystem volumes rather than proving each
volume is a GPT EFI System Partition. It recognizes only the allowlisted
paths in \`src/uefi/boot-manager.c\`; vendor-specific layouts, other CPU
architectures, and all Linux distributions are not covered.

## Secure Boot

The generated GNU-EFI application is **unsigned**. With Secure Boot enabled,
firmware or an enforcing shim/GRUB may reject it unless signed by a key trusted
by that machine. SelectOS does not disable Secure Boot, bypass signature
checks, enroll keys, or modify MOK/NVRAM. A future signed release needs a
reproducible signing and key-management process. Do not disable Secure Boot
merely to test this experimental build.

The selected OS loader is passed through firmware \`LoadImage\`; this
application does not patch or bypass image authentication. Secure Boot
behavior has not been verified across firmware implementations.

## Validation and release limits

CI compiles the PE/COFF x86_64 EFI application, checks its file type, runs
source safety tests, and builds both image artifacts. This does not prove
compatibility with every UEFI implementation. QEMU/OVMF integration tests and
physical hardware testing remain required.

The manifest intentionally retains \`PRODUCTION_READY=false\`. Keep a known
working boot path and do not replace a production bootloader with this
experimental image.
