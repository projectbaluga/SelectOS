# SelectOS authentication client

Go command-line reference client for the LAN authentication API and device-local offline credential cache. It is not yet wired into the shipped graphical SelectOS/Gate login screen; see [the Gate hook integration contract](../gate/README.md).

## Build

```sh
cd auth/client
go mod tidy
go vet ./...
go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o selectos-auth-client .
```

Choose `GOARCH=arm64` for a 64-bit ARM initramfs. The binary must match the kernel/userspace architecture.

## Online login and cache enrollment

For isolated-LAN testing against the current HTTP server:

```sh
read -rsp "Password: " SELECTOS_PASSWORD; export SELECTOS_PASSWORD; echo
SELECTOS_AUTH_ALLOW_HTTP=1 sudo --preserve-env=SELECTOS_PASSWORD,SELECTOS_AUTH_ALLOW_HTTP ./selectos-auth-client login --server http://DEBIAN-LAN-IP:8088 --username YOUR_USER
unset SELECTOS_PASSWORD
```

For production, use HTTPS with a trusted private CA and do not set `SELECTOS_AUTH_ALLOW_HTTP`.

The command caches an Argon2id password verifier encrypted with AES-256-GCM and a random per-device key. Keep the cache directory root-owned and mode 0700. Enroll each device separately; never copy a device key/cache to another computer.

## Offline verification

```sh
read -rsp "Password: " SELECTOS_PASSWORD; export SELECTOS_PASSWORD; echo
sudo --preserve-env=SELECTOS_PASSWORD ./selectos-auth-client offline-check --username YOUR_USER
unset SELECTOS_PASSWORD
```

The cache expires after 72 hours and is refreshed only after successful online authentication. Cache directory, key and cache-file types/permissions and timestamps are checked; cache writes are atomic.

## Exit status and fallback policy

- `0`: authentication accepted.
- `1`: rejected credentials or offline verification denied. **Do not fall back to offline after an online credential rejection.**
- `2`: invalid invocation/configuration or missing required input.
- `3`: authentication server/network unavailable (or server returns HTTP 5xx). The Gate hook may try offline verification only for this status.

Redirects are not followed. HTTPS is required by default. Set `SELECTOS_AUTH_ALLOW_HTTP=1` only for isolated-LAN testing against the current HTTP-only server.

## Security limitations

The local device key protects the cache at rest against casual file disclosure, but root compromise can access it. Argon2id slows offline password guessing; it does not make weak passwords safe. Server-side password changes/account disabling cannot revoke an already-enrolled offline cache while a device remains disconnected. Production requires TLS/private CA, device enrollment/revocation, rate limiting, monitored and tested backups, and persistent writable cache storage.

This tool verifies credentials only. It does not start a desktop session, unlock encrypted disks, or replace the legacy Gbox UI. The repository's shipped Gate initramfs is prebuilt and has no complete source/build recipe here.
