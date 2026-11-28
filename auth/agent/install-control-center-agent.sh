#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run with: sudo bash auth/agent/install-control-center-agent.sh"
  exit 1
fi
for cmd in go systemctl install useradd getent; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Missing required command: $cmd" >&2; exit 1; }
done
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
AGENT_DIR="$ROOT/auth/agent"
[[ -f "$AGENT_DIR/main.go" ]] || { echo "Run this script from a SelectOS repository checkout." >&2; exit 1; }
[[ -f "$AGENT_DIR/selectos-agent.service" ]] || { echo "Agent systemd unit is missing." >&2; exit 1; }

read -r -p "Control Center HTTPS URL (e.g. https://selectos-server.lan): " SERVER_URL
SERVER_URL="${SERVER_URL%/}"
if [[ "$SERVER_URL" != https://* ]]; then
  if [[ "$SERVER_URL" == http://* && "${SELECTOS_ALLOW_HTTP:-0}" == "1" ]]; then
    echo "WARNING: HTTP is allowed only for an isolated test LAN."
  else
    echo "HTTPS is required. For an isolated lab only, explicitly rerun with SELECTOS_ALLOW_HTTP=1." >&2
    exit 1
  fi
fi
read -r -p "Unique device ID (default: $(hostname)): " DEVICE_ID
DEVICE_ID="${DEVICE_ID:-$(hostname)}"
if [[ ! "$DEVICE_ID" =~ ^[A-Za-z0-9._-]{3,64}$ ]]; then
  echo "Device ID must be 3-64 letters, numbers, dots, underscores, or hyphens." >&2
  exit 1
fi
read -r -s -p "Server enrollment token: " ENROLLMENT_TOKEN
echo
if [[ "${#ENROLLMENT_TOKEN}" -lt 32 ]]; then
  echo "Enrollment token appears too short; use the server's long random secret." >&2
  exit 1
fi

if ! getent passwd selectos-agent >/dev/null; then
  useradd --system --home /var/lib/selectos-agent --shell /usr/sbin/nologin selectos-agent
fi
install -d -o selectos-agent -g selectos-agent -m 0700 /var/lib/selectos-agent
TMP_BIN="$(mktemp)"
trap 'rm -f "$TMP_BIN"; unset ENROLLMENT_TOKEN' EXIT
(
  cd "$AGENT_DIR"
  go build -trimpath -o "$TMP_BIN" .
)
install -o root -g root -m 0755 "$TMP_BIN" /usr/local/bin/selectos-agent
umask 077
{
  printf 'SELECTOS_SERVER_URL=%s\n' "$SERVER_URL"
  printf 'SELECTOS_DEVICE_ID=%s\n' "$DEVICE_ID"
  printf 'SELECTOS_ENROLLMENT_TOKEN=%s\n' "$ENROLLMENT_TOKEN"
  printf 'SELECTOS_STATE_FILE=/var/lib/selectos-agent/credentials.json\n'
} > /etc/selectos-agent.env
chown root:root /etc/selectos-agent.env
chmod 0600 /etc/selectos-agent.env
install -o root -g root -m 0644 "$AGENT_DIR/selectos-agent.service" /etc/systemd/system/selectos-agent.service
systemctl daemon-reload
systemctl enable --now selectos-agent
unset ENROLLMENT_TOKEN
echo
echo "SelectOS Control Center agent installed."
echo "Check status: sudo systemctl status selectos-agent"
echo "View logs:    sudo journalctl -u selectos-agent -f"
echo "Then confirm this device appears in Control Center > Devices."
echo "The PC continues working offline; only dashboard heartbeats require the server."
