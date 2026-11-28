# SelectOS

Lightweight UEFI multi-boot selector transitioning from the bundled legacy Gate/Gbox runtime toward a native SelectOS runtime.

**Authentication decision:** central authentication is deferred. Boot and local OS selection must not depend on the Debian auth server, PostgreSQL, LAN, or internet access.

> **Legacy limitation:** the compatibility image still uses the prebuilt Gate/Gbox initramfs from `gate-test.img.gz`, including its old login/UI behavior. Its complete source and reproducible build recipe are not available, so that image is not the login-free release. See [the migration plan](docs/NO-LOGIN-MIGRATION.md).

## Boot components

| Component | Current state |
|---|---|
| UEFI / GRUB | Existing EFI binaries and GRUB configuration |
| Kernel | Extracted from legacy source image |
| Native initramfs | Experimental BusyBox PID 1 and read-only diagnostic scanner |
| UEFI selector | Experimental GNU-EFI app with fixed loader allowlist and LoadImage/StartImage handoff |
| Authentication | Deferred; not a boot dependency |

## Legacy compatibility image

The legacy image builder requires explicit opt-in:

```bash
sudo apt-get update
sudo apt-get install -y parted dosfstools util-linux
sudo SELECTOS_ALLOW_LEGACY_GATE_RUNTIME=1 bash scripts/build-image.sh
```

Do not flash this compatibility artifact expecting the legacy Gbox login to be gone.

## Experimental native image

The image workflow builds a separate `SelectOS-native-experimental` artifact. GRUB keeps the native Linux diagnostic runtime as its default entry and adds an explicit **SelectOS UEFI Boot Manager (experimental)** option. The UEFI app enumerates UEFI-readable filesystems, checks a fixed allowlist of common Windows/Linux loader paths, and launches the selected image using UEFI `LoadImage` and `StartImage`.

The generated UEFI app is unsigned. Secure Boot may reject it unless signed by a key trusted by the machine. SelectOS does not disable Secure Boot or enroll keys. The artifact manifest remains `PRODUCTION_READY=false`; QEMU/OVMF and physical UEFI tests remain required. See [the experimental build guide](docs/EXPERIMENTAL-NATIVE-IMAGE.md).

## Control Center

The LAN-only SelectOS Control Center is separate from boot and local OS selection. It does not add a boot login requirement. See [docs/CONTROL-CENTER.md](docs/CONTROL-CENTER.md).

## Runtime migration and release status

Read [docs/NO-LOGIN-MIGRATION.md](docs/NO-LOGIN-MIGRATION.md). A successful image build is not proof of firmware compatibility; each release requires artifact inspection, QEMU/OVMF integration tests, and physical UEFI testing.

## License

See [LICENSE](LICENSE). Kernel, GRUB, MokManager, and other third-party binaries retain their respective upstream licenses.
