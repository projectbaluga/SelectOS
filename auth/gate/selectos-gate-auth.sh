#!/bin/sh
# SelectOS Gate authentication hook.
# Exit 0 only after online or permitted offline authentication succeeds.
set -eu

CLIENT=${SELECTOS_AUTH_CLIENT:-/usr/bin/selectos-auth-client}
SERVER=${SELECTOS_AUTH_SERVER:-}
CACHE=${SELECTOS_AUTH_CACHE:-/var/lib/selectos-auth-client}

if [ -z "$SERVER" ]; then
  echo "[SelectOS] SELECTOS_AUTH_SERVER is not configured." >&2
  exit 2
fi
if [ ! -x "$CLIENT" ]; then
  echo "[SelectOS] Authentication client is missing or not executable: $CLIENT" >&2
  exit 2
fi
if [ ! -t 0 ]; then
  echo "[SelectOS] Authentication requires an interactive console." >&2
  exit 2
fi

printf "SelectOS username: "
IFS= read -r SELECTOS_USERNAME
if [ -z "$SELECTOS_USERNAME" ] || [ "${#SELECTOS_USERNAME}" -gt 64 ]; then
  echo "[SelectOS] Invalid username." >&2
  exit 2
fi

printf "Password: "
stty -echo
if ! IFS= read -r SELECTOS_PASSWORD; then
  stty echo
  printf "\n"
  exit 2
fi
stty echo
printf "\n"

export SELECTOS_PASSWORD
export SELECTOS_AUTH_CACHE="$CACHE"
if [ "${SERVER#https://}" = "$SERVER" ] && [ "${SELECTOS_AUTH_ALLOW_HTTP:-0}" != "1" ]; then
  unset SELECTOS_PASSWORD
  echo "[SelectOS] HTTPS is required. HTTP is allowed only for isolated-LAN testing with SELECTOS_AUTH_ALLOW_HTTP=1." >&2
  exit 2
fi

set +e
"$CLIENT" login --server "$SERVER" --username "$SELECTOS_USERNAME" --cache-dir "$CACHE"
status=$?
set -e
if [ "$status" -eq 0 ]; then
  unset SELECTOS_PASSWORD
  exit 0
fi

if [ "$status" -ne 3 ]; then
  unset SELECTOS_PASSWORD
  exit "$status"
fi

echo "[SelectOS] Server unavailable; checking the enrolled offline credential cache."
set +e
"$CLIENT" offline-check --username "$SELECTOS_USERNAME" --cache-dir "$CACHE"
status=$?
set -e
unset SELECTOS_PASSWORD
exit "$status"
