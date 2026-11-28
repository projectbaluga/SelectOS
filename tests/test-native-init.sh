#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
INIT="$ROOT/src/native-initramfs/init"
BUILD="$ROOT/scripts/build-native-initramfs.sh"
INSTALLER="$ROOT/auth/agent/install-control-center-agent.sh"
bash -n "$INIT"
bash -n "$BUILD"
bash -n "$INSTALLER"
grep -q 'Native runtime — experimental' "$INIT"
grep -q 'No username/password login is configured' "$INIT"
grep -q 'No LAN, internet, PostgreSQL, or auth server is required' "$INIT"
if grep -Eiq 'SELECTOS_AUTH|POSTGRES|curl|wget|https?://' "$INIT"; then
  echo "Unexpected auth/network reference in native PID 1." >&2
  exit 1
fi
grep -q 'PRODUCTION_READY=false' "$BUILD"
grep -q 'SELECTOS_SERVER_URL' "$INSTALLER"
grep -q 'SELECTOS_ENROLLMENT_TOKEN' "$INSTALLER"
echo "Native initramfs and Control Center installer source checks passed."
