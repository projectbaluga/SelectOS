# QEMU/OVMF UEFI handoff integration test

The CI integration test boots the built SelectOS UEFI application under QEMU
with OVMF firmware. A test-only EFI child is placed at the allowlisted Ubuntu
GRUB path. The test injects the menu selection through QEMU's QMP/HMP keyboard
path and verifies that the child wrote a marker file to the FAT volume.

A passing test demonstrates, in this specific emulated environment, that OVMF
can boot the SelectOS application, the menu accepts a keyboard selection, and
the application's `LoadImage`/`StartImage` path can start an EFI child that
returns control. It does not validate real operating-system boot, Secure Boot,
physical hardware, all firmware implementations, or production readiness.

The test uses a temporary FAT image and does not modify host EFI variables or
NVRAM. The production native image remains marked `PRODUCTION_READY=false`.
