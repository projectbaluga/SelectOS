# SelectOS authentication client (prototype)

This is a Go command-line reference client for testing the LAN authentication API and device-local offline credential cache. It is **not yet wired into the graphical SelectOS/Gate login screen or initramfs**.

## Build

```sh
cd auth/client
go mod download
go build -o selectos-auth-client .
```

## Online login and cache enrollment

After the server is running and reachable over the trusted LAN:

```sh
sudo install -d -m 0700 -o root -g root /var/lib/selectos-auth-client
read -rsp "Password: " SELECTOS_PASSWORD; export SELECTOS_PASSWORD; echo
sudo --preserve-env=SELECTOS_PASSWORD ./selectos-auth-client login --server http://DEBIAN-LAN-IP:8088 --username YOUR_USER
unset SELECTOS_PASSWORD
```

The command caches an Argon2id password verifier encrypted with AES-256-GCM and a random per-device key. Keep the cache directory root-owned and mode 0700. A separate cache must be enrolled on each device; never copy a device key or cache from one computer to another.

## Offline verification

```sh
read -rsp "Password: " SELECTOS_PASSWORD; export SELECTOS_PASSWORD; echo
sudo --preserve-env=SELECTOS_PASSWORD ./selectos-auth-client offline-check --username YOUR_USER
unset SELECTOS_PASSWORD
```

The cache expires after 72 hours and is refreshed only by a successful online login. The client rejects unsafe cache-directory/key/cache-file permissions, validates cache timestamps and file types, writes cache updates atomically, limits request time, and refuses redirects so login credentials are not forwarded to another host. This is a prototype: disabling an account or changing its password on the server cannot revoke an already-enrolled offline cache until it expires or the device reconnects. For production, shorten the lease to your policy, implement device enrollment/revocation and TPM-backed key protection where available, and enforce a TLS endpoint with a private CA. The current example API listens over HTTP, so do not use it over untrusted networks.

Run the client tests with `go test ./...` from this directory. They cover cache enrollment/verification, wrong credentials, expiry, unsafe key permissions, and invalid lifetimes.\n\n## Security model and limitations

The local device key protects the cache at rest against casual file disclosure, but a root compromise can access it. Argon2id slows offline password guessing; it does not make weak passwords safe. This tool verifies credentials only. It does not start a desktop session, unlock encrypted disks, or replace the legacy Gbox UI. The prebuilt initramfs currently in the SelectOS repository has no source/build recipe here, so graphical integration requires rebuilding that runtime or supplying its source.
