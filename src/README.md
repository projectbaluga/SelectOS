# Runtime sources

| Path | What it is |
| ---- | ---------- |
| `gate-runtime/` | Go source of the `/init` binary inside `boot/select.img` (graphical OS selector, `select.json`/`gate.json` config, SelectOS logo support). `gate-runtime/tools/build-image.sh` rebuilds `gate-test.img.gz`-style images from it. See `gate-runtime/README.md`. |
| `native-initramfs/` | Experimental shell-based native PID 1 (diagnostic bring-up only). |
