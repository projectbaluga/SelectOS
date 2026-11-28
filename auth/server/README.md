# SelectOS LAN Server and Control Center

Go API and administrator dashboard for a Debian server on the local network. It uses PostgreSQL and Docker Compose and has no cloud identity-provider dependency.

## Features

- PostgreSQL persistence, bcrypt admin accounts, signed 8-hour bearer tokens.
- Administrator-only dashboard at /dashboard, served locally by the Go API.
- Host health: uptime, load average, CPU core count, memory and root filesystem usage.
- Device enrollment using a shared bootstrap secret; each PC receives a unique random token whose SHA-256 hash is stored by the server.
- Linux heartbeat agent, device inventory, online/offline status and credential revocation.
- Application audit events, backup inventory, host-side backup and interactive restore scripts.
- Strict same-origin browser policy and security headers.

The dashboard administrator account is management-plane access. It is not a SelectOS workstation login and is not required for boot or OS selection. The existing legacy Gate/Gbox boot runtime remains a separate migration problem.

## Deploy (one command)

Install Docker Engine, the Docker Compose plugin, OpenSSL, and curl on Debian. From the repository root, run:

    cd ~/SelectOS
    bash auth/server/setup.sh

The setup script is designed to be re-run safely:

- On a genuinely new install, it generates independent database, signing, and enrollment secrets, creates a mode-0600 `.env`, and prints the initial administrator password once. Save that password securely.
- On an existing install, it keeps the current `.env`, starts PostgreSQL, creates a compressed database backup, then builds and updates the API.
- If a database volume exists but `.env` is missing or contains template placeholders, it stops instead of inventing a new database password that could disconnect the existing database.
- It checks the API health endpoint before reporting success.

Do not replace an existing `.env` with `.env.example`. The database password must remain the same one used when the PostgreSQL volume was initialized. Keep `.env` private and never commit it.

The default `SELECTOS_BIND_IP` is `127.0.0.1`. Put a TLS reverse proxy in front of the API before allowing browser or agent access from other LAN machines. A sample private-CA Caddy configuration is in `deploy/Caddyfile`. For a temporary isolated-LAN test only, binding directly to the server LAN IP over HTTP is possible, but passwords and bearer tokens are sent in cleartext. Never expose the API or PostgreSQL to the public internet.

## Enroll PCs

Build and install the Linux agent from auth/agent. Follow [the agent guide](../agent/README.md). Configure its server URL, unique device ID, and the same SELECTOS_DEVICE_ENROLLMENT_TOKEN configured on the server. The agent enrolls once and saves its unique device token in a mode-0600 state file. It sends heartbeats every 30 seconds; the dashboard considers a device online for two minutes after its last heartbeat.

The current agent reports hostname, platform, version, CPU, memory and disk use. Windows/macOS agents are not implemented. Revoking a device invalidates its credential; re-enrollment is needed to restore reporting.

## Backups

Create a compressed PostgreSQL backup on the Debian host:

    bash scripts/backup.sh

The script writes to auth/server/backups and prunes files older than 30 days. The dashboard lists backup files from the read-only bind mount. Store a second copy on separate storage.

Restore is destructive and requires typing RESTORE:

    bash scripts/restore.sh /path/to/selectos-YYYYMMDDTHHMMSSZ.sql.gz

Test backup integrity and restore into a disposable database before relying on the process. Do not run docker compose down -v unless intentionally deleting the database volume.

## API

- GET /healthz
- POST /v1/auth/login
- GET /v1/auth/me
- GET /dashboard/
- GET /v1/admin/overview and GET /v1/admin/metrics
- GET /v1/admin/devices
- PATCH /v1/admin/devices/{id}/status
- GET /v1/admin/events
- GET /v1/admin/backups
- POST /v1/devices/enroll (shared enrollment header)
- POST /v1/devices/heartbeat (per-device bearer token)

Every /v1/admin endpoint requires an administrator bearer token. Device enrollment and heartbeat use separate credentials. PostgreSQL is reachable only on the internal Compose network.

## Validation and remaining production work

The CI workflow builds/tests the server, client and Linux agent. A real deployment still needs TLS validation, firewall review, backup/restore drills, rate limiting on authentication/enrollment, monitoring/alerting, and endpoint hardware testing. The current server uses HTTP behind the recommended reverse proxy; configure TLS before remote LAN use. Remote shell/reboot/shutdown is intentionally not part of this milestone.
