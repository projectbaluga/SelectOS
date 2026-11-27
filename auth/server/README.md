# SelectOS Local Authentication Server

Go HTTP API for a Debian machine on the same LAN as SelectOS clients. The service uses PostgreSQL and Docker Compose; it does not depend on an internet identity provider.

## Stack and current scope

- PostgreSQL 17 database with persistent Docker volume.
- bcrypt password hashes and case-insensitive unique usernames.
- Bootstrap first admin through environment variables.
- Bearer tokens signed with an installation-specific HMAC key (8-hour expiry).
- Admin endpoints to list/create/disable users and reset passwords.
- Health endpoint and basic audit events.
- No passwords are logged or returned by the API.

## Deploy on Debian with Docker Compose

Install Docker Engine and the Docker Compose plugin first. Then:

```sh
git clone --branch feature/local-auth https://github.com/projectbaluga/SelectOS.git
cd SelectOS/auth/server
cp .env.example .env
```

Generate unique secrets and edit `.env` (keep it private):

```sh
openssl rand -hex 32
openssl rand -hex 32
chmod 600 .env
nano .env
```

Use the first generated value for `SELECTOS_DB_PASSWORD`, the second for `SELECTOS_AUTH_TOKEN_KEY`, and set a unique `SELECTOS_ADMIN_PASSWORD` of at least 12 characters. Do not commit or share `.env`.

Start the stack:

```sh
docker compose up -d --build
docker compose ps
docker compose logs -f auth
```

The API listens on TCP port 8088 by default. Clients on the same LAN can use `http://SERVER-LAN-IP:8088` during isolated-LAN testing. PostgreSQL is not published to the host network; only the API port is exposed. Change the port with `SELECTOS_AUTH_PORT`.

Check health:

```sh
curl http://127.0.0.1:8088/healthz
```

## Data persistence and backup

PostgreSQL data is stored in the named Docker volume `selectos-postgres`, so recreating the containers does not remove the database.

Create a backup:

```sh
docker compose exec -T postgres pg_dump -U selectos selectos > selectos-backup.sql
```

Restore into an empty database:

```sh
cat selectos-backup.sql | docker compose exec -T postgres psql -U selectos -d selectos
```

Do not run `docker compose down -v` unless you intentionally want to delete the database volume.

## Security notes

The API currently uses HTTP, not TLS. **Do not expose port 8088 to the public internet or an untrusted network.** For production, put the API behind TLS (for example, a private-CA reverse proxy), restrict inbound access with the Debian firewall to the client LAN, and use unique strong secrets. The PostgreSQL port is intentionally internal to the Compose network.

A production rollout still needs rate limiting/lockout, secure device enrollment and revocation, tested backup/restore, and monitoring. The offline login cache is not yet integrated into this server milestone. The existing Gate initramfs is a prebuilt binary with no source in this repository, so this API does not yet replace its Gbox login screen; a client/login component and initramfs integration are still required.

## API

- `GET /healthz`
- `POST /v1/auth/login` JSON `{"username":"...","password":"..."}`
- `GET /v1/auth/me` with `Authorization: Bearer TOKEN`
- Admin: `GET /v1/admin/users`
- Admin: `POST /v1/admin/users` JSON `{"username":"student01","password":"long password","role":"user"}`
- Admin: `PATCH /v1/admin/users/{id}/status` JSON `{"enabled":false}`
- Admin: `POST /v1/admin/users/{id}/password` JSON `{"password":"new long password"}`

## Local build (optional)

Requires Go 1.23+ and a reachable PostgreSQL instance. Set `SELECTOS_DATABASE_URL`, `SELECTOS_AUTH_TOKEN_KEY`, and admin bootstrap variables, then:

```sh
go mod download
go test ./...
go build -o selectos-auth .
```
