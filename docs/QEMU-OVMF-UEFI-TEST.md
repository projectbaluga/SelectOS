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

The test compiles both EFI applications for the baseline x86-64 ISA (without
SSE4.1/SSE4.2 or AVX dependencies). The boot manager constructs its file device
path locally rather than relying on a distro-built GNU-EFI helper that may use
newer SIMD instructions.

The test uses a temporary GPT/FAT32 EFI System Partition image and does not
modify host EFI variables or NVRAM. The production native image remains marked `PRODUCTION_READY=false`.


### Failure diagnostics

The QEMU harness compiles the boot manager with
`SELECTOS_QEMU_TEST_DIAGNOSTICS`. This test-only build writes
`selectos-debug.log` to the disposable ESP, recording that the manager
started and how many UEFI Simple File System handles and supported loader
entries it discovered. The normal image build does not define this macro and
does not write this diagnostic file. When the child marker is missing, the
test prints the diagnostic log if the manager was able to create it.
