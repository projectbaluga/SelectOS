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

The test uses a temporary whole-device FAT32 removable-media image and a
known-good GRUB bootstrap extracted from the compatibility image. This isolates
SelectOS manager loading from the firmware's direct PE/COFF loading path; it is
not a test of GPT ESP discovery. The test does not modify host EFI variables or
NVRAM. The production native image remains marked `PRODUCTION_READY=false`.


### Failure diagnostics

The QEMU harness compiles the boot manager with
`SELECTOS_QEMU_TEST_DIAGNOSTICS`. This test-only build writes
`selectos-debug.log` to the disposable ESP, recording that the manager started and how many UEFI Simple File System
handles and supported loader entries it discovered. The test build also emits
a console marker at the first line of efi_main, before GNU-EFI library
initialization, to distinguish entrypoint execution from a failure during
initialization. Diagnostic files are attempted on every discovered Simple File
System volume rather than assuming the image's LoadedImage->DeviceHandle is
the writable volume. The normal image build does not define this macro and
does not write this diagnostic file. When the child marker is missing, the
test prints the diagnostic log if the manager was able to create it.


### Current debugging observation

- Workflow run #74 confirmed that the manager reached its entrypoint, completed GNU-EFI initialization, and returned from filesystem scanning and post-scan diagnostics. The application then returned EFI status 14 (`EFI_NOT_FOUND`) without creating the child marker.
- A follow-up attempt added variadic `LocalPrint` calls inside the scanner. Run #76 reached initialization but did not emit the post-scan marker, so those extra formatter calls were removed from the test-only scan path.
- The current test-only scan diagnostics use direct UEFI `OutputString` markers to distinguish: `LocateHandleBuffer` failure, no Simple File System handles, handles found, protocol/open-volume failures, a matched allowlisted loader, and zero/nonzero entry count.
- Runs #75 and #76 did not complete the child handoff. Runs #77 and #78 are validating the direct-marker instrumentation and the corrected diagnostic string escapes.
- No successful `LoadImage`/`StartImage` handoff is claimed yet.
