# gate-runtime

Source of the selector that runs as PID 1 inside the boot initramfs: framebuffer UI, UEFI boot-entry
discovery (`efivarfs`), `BootNext` + reboot, and `gate.json` configuration (brand, logo, OS rules, icons).
Standard library only; the Terminus bitmap fonts (SIL OFL) are embedded from `gate/assets/`.

```
cd src/gate-runtime
go vet ./gate && go test ./gate            # renders preview PNGs to /tmp
sudo bash tools/build-image.sh             # -> dist/gate-test.img (UEFI, Secure Boot capable)
EXTRA="gate.demo" sudo bash tools/build-image.sh   # demo cards, dry-run (never writes BootNext)
```

The build uses Ubuntu's signed shim, GRUB and kernel, so Secure Boot can stay enabled. Configuration comes from
`config/select.json` (copied to `gate.json`) and `icons/` in the repository root; set `CONFIG=<dir>` to override.

Note: the current UI still contains the legacy sign-in screen; removing it is tracked in `docs/NO-LOGIN-MIGRATION.md`.
