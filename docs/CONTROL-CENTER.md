# SelectOS Control Center

The Control Center is an administrator-only web console for the Debian server on the local network. It is separate from the SelectOS PC boot/login path: a PC boots and selects an OS without this server, without internet, and without the dashboard.

## Implemented in this milestone

- Responsive web dashboard at /dashboard, served by the Go API (no external JavaScript/CDN dependency).
- Existing server administrator login; administrator role is checked by every protected API request.
- Server overview: hostname, OS/architecture, CPU core count, uptime, load average, memory, root filesystem use, database health, and registered/online device counts.
- Device enrollment with a shared bootstrap secret and unique per-device random bearer credential. Only a SHA-256 hash of the device credential is stored server-side.
- Linux device heartbeat agent, 30-second reporting interval, host/IP/platform/version and memory/disk metrics.
- Device inventory and revoke action. Revocation invalidates the credential; a device must enroll again to report.
- Recent application audit events.
- Backup inventory and host-side PostgreSQL backup/restore scripts. Restore requires an explicit typed confirmation.
- Browser security headers and same-origin static assets.

## Not included / intentionally deferred

- No PC user login, boot gate, remote desktop, remote shell, file browsing, restart, or shutdown.
- No remote command channel. This prevents a dashboard compromise from becoming a fleet-wide arbitrary command execution path.
- No internet/cloud dependency. The dashboard and API are intended for the private LAN.
- The current agent reports memory and disk utilization; its CPU metric is currently zero/not sampled. Per-PC live CPU graphs are not claimed.
- Audit events are application events, not a full journalctl/syslog viewer.
- Dashboard administrator authentication is management-plane access; it does not authenticate a person at a SelectOS workstation.

## Deploy the server

1. On Debian, install Docker Engine and the Compose plugin.
2. Clone the branch containing this feature, then enter auth/server.
3. Copy .env.example to .env. Generate three independent secrets:
   
       openssl rand -hex 32
       openssl rand -hex 32
       openssl rand -hex 32

4. Set SELECTOS_DB_PASSWORD, SELECTOS_AUTH_TOKEN_KEY, SELECTOS_ADMIN_PASSWORD, and SELECTOS_DEVICE_ENROLLMENT_TOKEN in .env. Use unique values. The admin password must be at least 12 characters. Set SELECTOS_BIND_IP to 127.0.0.1 (recommended when placing a TLS reverse proxy on the host). Protect the file:

       chmod 600 .env

5. Start the services:

       docker compose up -d --build
       docker compose ps
       curl http://127.0.0.1:8088/healthz

6. Put a TLS reverse proxy in front of the API before allowing browser or agent access from other LAN machines. A sample Caddy configuration is provided in deploy/Caddyfile. Use a private DNS name and install/trust the private CA certificate on the administrator browser and endpoint PCs. Do not expose port 8088 directly to untrusted networks.

For a small isolated test LAN only, SELECTOS_BIND_IP may be set to the Debian server's LAN IP and the dashboard opened at http://SERVER-LAN-IP:8088/dashboard/. HTTP exposes credentials and bearer tokens to anyone able to observe the LAN; do not use this mode on shared/untrusted networks.

## Enroll Linux PCs

1. Install Go 1.23+ on a build machine, then build auth/agent:

       cd auth/agent
       go build -trimpath -o selectos-agent .

2. Create a dedicated service account and install the agent:

       sudo useradd --system --home /var/lib/selectos-agent --shell /usr/sbin/nologin selectos-agent
       sudo install -d -o selectos-agent -g selectos-agent -m 0700 /var/lib/selectos-agent
       sudo install -m 0755 selectos-agent /usr/local/bin/selectos-agent

3. Create /etc/selectos-agent.env with the server URL, a unique SELECTOS_DEVICE_ID, the enrollment secret, and state file path. The URL must use the same trusted HTTPS reverse proxy for production. Protect the file with chmod 600.
4. Install selectos-agent.service to /etc/systemd/system/, then run:

       sudo systemctl daemon-reload
       sudo systemctl enable --now selectos-agent
       sudo journalctl -u selectos-agent -f

The agent enrolls on first run and writes its device token to /var/lib/selectos-agent/credentials.json with restrictive permissions. Remove SELECTOS_ENROLLMENT_TOKEN from the service environment after enrollment if you want to reduce secret exposure; the current agent only reads it when no stored device token exists.

The dashboard marks a device online when its last heartbeat is less than two minutes old. Devices do not appear until the agent is installed and can reach the server.

## Backups and restore

Create a compressed backup on the Debian host:

    cd auth/server
    bash scripts/backup.sh

Backups are written to auth/server/backups, which is mounted read-only into the API container for inventory display. The script retains the most recent 30 days of compressed backup files. Store an additional copy on separate storage; a Docker volume or same-disk backup is not disaster recovery.

Restore is destructive and interactive:

    bash scripts/restore.sh /path/to/selectos-YYYYMMDDTHHMMSSZ.sql.gz

The script requires typing RESTORE. Stop dependent services if necessary, verify the backup source, and validate the application after restore. Never run docker compose down -v unless you intend to delete the PostgreSQL volume.

## API endpoints

- Public health: GET /healthz
- Admin login: POST /v1/auth/login
- Admin identity: GET /v1/auth/me
- Admin overview and host metrics: GET /v1/admin/overview, GET /v1/admin/metrics
- Device inventory and revoke: GET /v1/admin/devices, PATCH /v1/admin/devices/{id}/status
- Audit and backup inventory: GET /v1/admin/events, GET /v1/admin/backups
- Device enrollment: POST /v1/devices/enroll with X-SelectOS-Enrollment-Token
- Device heartbeat: POST /v1/devices/heartbeat with per-device Bearer token

All /v1/admin endpoints require an administrator bearer token. Device enrollment requires the shared enrollment secret. Heartbeats require the per-device token. PostgreSQL is not published to the host network.

## Operational checks

- Verify the admin login and that non-admin users receive HTTP 403 on admin endpoints.
- Enroll one test PC and confirm it appears online, then stop the agent and confirm it becomes offline after two minutes.
- Revoke the test device and confirm the previous token receives HTTP 401.
- Create a backup, verify gzip integrity, and test restoration into a disposable database before relying on it.
- Test dashboard access from an authorized LAN client over TLS.
- Confirm booting and OS selection still works with the server powered off and the network disconnected.
