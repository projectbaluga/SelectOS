# Identify the Debian server

Run these commands on the Debian machine and share the output (you may redact hostnames and IP addresses):

```sh
cat /etc/os-release
uname -m
dpkg --print-architecture
go version || true
ip -br address
```

Expected architecture mapping:
- `x86_64` / `amd64`: Go `GOARCH=amd64`
- `aarch64` / `arm64`: Go `GOARCH=arm64`

The Go client and server can be built natively on the Debian host, so a cross-compile is not required if Go is installed there.
