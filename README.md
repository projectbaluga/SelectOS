<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/branding/svg/selectos-logo-dark-bg.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/branding/svg/selectos-logo-light-bg.svg">
    <img alt="SelectOS" src="assets/branding/svg/selectos-logo-light-bg.svg" width="460">
  </picture>
</p>

Lightweight UEFI multi-boot selector. The project is transitioning away from the bundled legacy Gate/Gbox runtime toward a native SelectOS runtime.

**Authentication decision:** central authentication is deferred for a future release. The SelectOS boot path must not depend on the Debian authentication server, PostgreSQL, LAN, or internet access. The intended initial experience is a direct boot into the SelectOS operating-system selector without a login prompt.

> **Important current limitation:** the boot image still uses the prebuilt legacy Gate initramfs extracted from `gate-test.img.gz`. That binary contains the legacy Gate application and its login/UI behavior. Its full source and reproducible build recipe are not present here. Therefore the current image is **not login-free**, and this change does not claim to remove the Gbox login. See [the no-login runtime migration plan](docs/NO-LOGIN-MIGRATION.md).

## Current and target boot components

| Component | Current repository state | Target |
|---|---|---|
| UEFI / GRUB | Existing EFI binaries and GRUB configs | Retain, then validate |
| Kernel | Extracted from legacy source image | Retain initially |
| Initramfs / PID 1 | Prebuilt Gate/Gbox binary | Native SelectOS initramfs with no login gate |
| OS detection / selector | Legacy implementation | Native SelectOS implementation |
| Central authentication | Prototype server/client source only; not in boot path | Deferred; add later as optional LAN service |

## Authentication is deferred

- No authentication server, PostgreSQL service, login client, or network connection is required to boot or use the selector.
- Existing `auth/server` and `auth/client` source is retained for future work; it is not installed into the boot image.
- Do not add an authentication prompt or make boot conditional on successful authentication in the initial login-free milestone.
- The old Gate/Gbox login cannot be removed by editing GRUB or the image packager. The initramfs/PID 1 runtime must be replaced or rebuilt from complete source.

## Legacy image build (temporary compatibility only)

The current image builder can package the legacy runtime only when explicitly opted into. This is a transitional artifact, **not the login-free release**.

GitHub Actions:
1. Open **Actions**.
2. Select **Build legacy-compatible SelectOS image**.
3. Run the workflow on `main`.
4. The artifact is named `SelectOS-legacy-Gate-runtime` and includes a build manifest warning that the legacy login remains.

Local build on Ubuntu/Debian:

```bash
sudo apt-get update
sudo apt-get install -y parted dosfstools util-linux
sudo SELECTOS_ALLOW_LEGACY_GATE_RUNTIME=1 bash scripts/build-image.sh
```

The output is `dist/SelectOS.img`, a 512 MiB FAT32 UEFI image. The explicit opt-in is deliberate: it prevents the legacy binary from being mistaken for the future login-free runtime.

**Do not write a legacy artifact to a production USB expecting the Gbox login to be gone.**

## Server management dashboard

The LAN-only SelectOS Control Center is served by the Debian Go server at `/dashboard`. It includes administrator access, host health metrics, a device registry with per-device enrollment tokens and heartbeat agent, audit events, and PostgreSQL backup inventory/scripts. See [docs/CONTROL-CENTER.md](docs/CONTROL-CENTER.md) for deployment, TLS, enrollment, and recovery instructions. The dashboard is separate from PC boot authentication; it does not add a login requirement to SelectOS boot.

## No-login runtime implementation

Read [docs/NO-LOGIN-MIGRATION.md](docs/NO-LOGIN-MIGRATION.md) for the required source tree, boot contract, implementation phases, acceptance tests, and the precise reason a new login-free bootable image cannot yet be built from the files currently available.

## Configuration

Edit `config/select.json` for the current legacy selector's branding and OS presentation rules. The legacy initramfs reads the compatibility copy at `/gate.json`. A native runtime should define and document its own stable configuration schema rather than silently assuming these legacy paths.

## Project structure

```text
SelectOS/
├── .github/workflows/
├── auth/                    # future LAN authentication source; not in boot path
├── boot/                    # kernel, initramfs and GRUB files
├── config/select.json
├── docs/NO-LOGIN-MIGRATION.md
├── scripts/build-image.sh
└── src/                     # native runtime source to be implemented
```

## Release status

A buildable image is not automatically a verified bootable system. Every release needs a successful image build, artifact inspection, and real UEFI testing on representative hardware.

## License

See [LICENSE](LICENSE). Kernel, GRUB, MokManager, and other third-party binaries retain their respective upstream licenses.
