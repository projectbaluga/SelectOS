# Gate authentication hook — integration contract

This directory contains a console authentication hook for the existing SelectOS auth CLI. It is an integration component, **not a replacement for the Gate initramfs**.

## Files and responsibilities

- `selectos-gate-auth.sh`: prompts for username/password without echo, attempts online authentication, and tries offline verification only when the client returns exit status 3 (network/service unavailable).
- `../client`: performs API validation and maintains the encrypted device-local offline cache.

## Build and install

Build the client for the target initramfs architecture. On x86_64:

```sh
cd auth/client
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o selectos-auth-client .
```

Install both files into the initramfs at stable paths, for example:

```sh
install -D -m 0755 selectos-auth-client /usr/bin/selectos-auth-client
install -D -m 0755 auth/gate/selectos-gate-auth.sh /usr/lib/selectos/selectos-gate-auth
```

Set the runtime configuration before calling the hook:

- `SELECTOS_AUTH_SERVER=https://auth.example.lan`
- `SELECTOS_AUTH_CACHE=/var/lib/selectos-auth-client` (must be on persistent writable storage)
- `SELECTOS_AUTH_CLIENT=/usr/bin/selectos-auth-client`

For isolated-LAN testing against the current HTTP API only, set `SELECTOS_AUTH_ALLOW_HTTP=1`. Do not use HTTP on public or untrusted networks.

The hook exits 0 only after online or offline authentication succeeds. Any other exit code must stop the protected login/session transition. In particular, invalid credentials (HTTP 401) must never trigger offline fallback. Exit code 3 from the CLI is reserved for unavailable network/service conditions.

## Critical integration limitation

The repository's current `boot/select.img` is a prebuilt legacy Gate initramfs, and the repository does not contain its complete source or reproducible build recipe. Therefore this hook cannot be injected into the shipped image by the existing image builder alone. The actual Gate `/init` source must call this hook before launching the OS-selection UI/session, and the initramfs build must include the client binary, hook, CA trust bundle, networking tools/modules, and a persistent writable cache mount.

Do not replace `boot/select.img`, alter `rdinit=/init`, or publish an image claiming enforced authentication until the complete Gate source is available and a rebuilt image passes hardware boot tests.

## Required boot sequence

1. Mount proc, sysfs, devtmpfs and required network/device modules.
2. Discover and mount the configured persistent data partition with restrictive permissions.
3. Bring up the network with a bounded timeout; DNS and time synchronization must be available for TLS validation.
4. Invoke this hook on the local console.
5. On success, continue to the existing Gate UI/session.
6. On rejection, remain at the login prompt; never fall through to the protected UI.
7. Preserve a separately documented, physically controlled recovery path for administrators.

## Validation matrix

| Scenario | Expected result |
|---|---|
| Server reachable, valid credentials | Online success; refresh cache |
| Server reachable, wrong password | Denied; no offline fallback |
| Account disabled on server | Denied; no offline fallback |
| Server unreachable, valid unexpired cache | Offline success |
| Server unreachable, wrong password | Denied |
| Server unreachable, expired/corrupt cache | Denied |
| Cache directory/key permissions unsafe | Denied |
| Missing client/configuration/TTY | Fail closed |

Offline cache revocation is not immediate: server-side password changes or account disabling cannot invalidate an already enrolled device while it remains disconnected. The 72-hour lease is a maximum, not a revocation mechanism. Production deployment still requires device enrollment/revocation, TLS/private CA, rate limiting, audited recovery, and tested persistent storage.
