# SelectOS Local Authentication Server

Go HTTP API for a Debian machine on the same LAN as SelectOS clients. It does not depend on an internet identity provider.

## Current scope

- SQLite user database; bcrypt password hashes.
- Bootstrap first admin through environment variables.
- Bearer tokens signed with an installation-specific HMAC key (8-hour expiry).
- Admin endpoints to list/create/disable users and reset passwords.
- Health endpoint and basic audit events.
- No passwords are logged or returned by the API.

## Important security/deployment notes

This first server implementation is HTTP-only and is intended for a trusted isolated LAN during development. **Do not expose port 8088 to the public internet or an untrusted network.** Before production deployment, put it behind TLS (for example, a private-CA reverse proxy), restrict inbound access with the Debian firewall, and provision a unique random token key. A production rollout also needs rate limiting/lockout, CSRF protection if a browser admin UI is added, backup/restore, and device enrollment.

The offline login cache is deliberately not enabled in this server-only milestone. A safe cache requires device enrollment and a per-device encryption key protected by the client OS; do not store reusable password hashes in a shared boot image. The existing Gate initramfs is a prebuilt binary with no source in this repository, so this API cannot replace its Gbox screen until a new client/login component is built and integrated into the initramfs.

## Build

Requires Go 1.23+ and network access once to download modules:

```sh
cd auth/server
go mod download
go build -o selectos-auth .
```

## Run manually

```sh
install -d -m 700 data
export SELECTOS_AUTH_TOKEN_KEY="$(openssl rand -hex 32)"
export SELECTOS_ADMIN_USER="admin"
read -rsp "Initial admin password: " SELECTOS_ADMIN_PASSWORD; echo
export SELECTOS_ADMIN_PASSWORD
./selectos-auth
```

Listen address defaults to `0.0.0.0:8088`; set `SELECTOS_AUTH_ADDR` to change it. Database defaults to `./data/selectos-auth.db`.

## API

- `GET /healthz`
- `POST /v1/auth/login` JSON `{"username":"...","password":"..."}`
- `GET /v1/auth/me` with `Authorization: Bearer TOKEN`
- Admin: `GET /v1/admin/users`
- Admin: `POST /v1/admin/users` JSON `{"username":"student01","password":"long password","role":"user"}`
- Admin: `PATCH /v1/admin/users/{id}/status` JSON `{"enabled":false}`
- Admin: `POST /v1/admin/users/{id}/password` JSON `{"password":"new long password"}`

## Debian service

Review and adapt `systemd/selectos-auth.service`. Create a dedicated system user, install the binary at `/opt/selectos-auth/selectos-auth`, and put environment values in root-owned `/etc/selectos-auth.env` (mode 0600). The service unit uses `/var/lib/selectos-auth` for state; set `SELECTOS_AUTH_DB=/var/lib/selectos-auth/selectos-auth.db`.
