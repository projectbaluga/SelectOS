#!/usr/bin/env bash
# One-command, idempotent setup/update for the SelectOS LAN server.
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"
ENV_FILE="$ROOT/.env"

die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"; }
need docker
need openssl
need curl
docker compose version >/dev/null 2>&1 || die "Install the Docker Compose plugin first."

# Never invent a new database password for an existing PostgreSQL data volume.
EXISTING_VOLUME="$(docker volume ls --format '{{.Name}}' | grep -E '(^|_)selectos-postgres$' | head -n1 || true)"
if [[ ! -f "$ENV_FILE" ]]; then
  [[ -z "$EXISTING_VOLUME" ]] || die "Existing database volume '$EXISTING_VOLUME' found but .env is missing. Restore the original auth/server/.env first; generating a new DB password could make the database inaccessible."
fi

if [[ -f "$ENV_FILE" ]] && grep -Eq 'REPLACE_WITH|^SELECTOS_(DB_PASSWORD|AUTH_TOKEN_KEY|ADMIN_PASSWORD|DEVICE_ENROLLMENT_TOKEN)=$' "$ENV_FILE"; then
  [[ -z "$EXISTING_VOLUME" ]] || die "The existing .env contains template placeholders while database volume '$EXISTING_VOLUME' already exists. Restore the original .env; do not generate replacement database credentials."
  mv "$ENV_FILE" "$ENV_FILE.example-backup.$(date -u +%Y%m%dT%H%M%SZ)"
  rm -f "$ENV_FILE"
fi

if [[ ! -f "$ENV_FILE" ]]; then
  umask 077
  ADMIN_PASSWORD="$(openssl rand -hex 24)"
  cat > "$ENV_FILE" <<EOF
SELECTOS_DB_PASSWORD=$(openssl rand -hex 32)
SELECTOS_AUTH_TOKEN_KEY=$(openssl rand -hex 32)
SELECTOS_ADMIN_USER=admin
SELECTOS_ADMIN_PASSWORD=$ADMIN_PASSWORD
SELECTOS_AUTH_PORT=8088
SELECTOS_BIND_IP=127.0.0.1
SELECTOS_DEVICE_ENROLLMENT_TOKEN=$(openssl rand -hex 32)
EOF
  chmod 600 "$ENV_FILE"
  printf '\nNEW INSTALL: generated private credentials in %s\n' "$ENV_FILE"
  printf 'Save this initial dashboard admin password now: %s\n' "$ADMIN_PASSWORD"
  printf 'The password is not shown again by this script.\n\n'
else
  chmod 600 "$ENV_FILE"
  for key in SELECTOS_DB_PASSWORD SELECTOS_AUTH_TOKEN_KEY SELECTOS_ADMIN_PASSWORD SELECTOS_DEVICE_ENROLLMENT_TOKEN; do
    grep -Eq "^$key=.+$" "$ENV_FILE" || die "$key is missing or empty in .env."
  done
fi

if [[ -n "$EXISTING_VOLUME" ]]; then
  printf 'Existing PostgreSQL volume detected: %s\n' "$EXISTING_VOLUME"
  printf 'Starting database only so a safety backup can be made before updating the API...\n'
  docker compose up -d postgres
  ready=0
  for _ in $(seq 1 40); do
    if docker compose exec -T postgres pg_isready -U selectos -d selectos >/dev/null 2>&1; then ready=1; break; fi
    sleep 2
  done
  [[ "$ready" -eq 1 ]] || die "PostgreSQL did not become ready. No API update was attempted."
  bash "$ROOT/scripts/backup.sh"
fi

printf 'Building and starting SelectOS Server...\n'
docker compose up -d --build

healthy=0
for _ in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:${SELECTOS_AUTH_PORT:-8088}/healthz" >/dev/null 2>&1; then healthy=1; break; fi
  sleep 2
done
[[ "$healthy" -eq 1 ]] || { docker compose ps; die "Health check failed. Inspect logs with: docker compose logs --tail=100 auth"; }

printf '\nSelectOS Server is healthy.\n'
docker compose ps
printf '\nDashboard endpoint (local server): http://127.0.0.1:${SELECTOS_AUTH_PORT:-8088}/dashboard/\n'
printf 'For LAN access, configure the documented TLS reverse proxy; do not expose plain HTTP on an untrusted network.\n'
