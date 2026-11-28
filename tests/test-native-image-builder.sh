#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf -- "$TMP"' EXIT

# The experimental packager must refuse to run without explicit opt-in.
if (cd "$ROOT" && env -u SELECTOS_BUILD_NATIVE_EXPERIMENTAL bash scripts/build-native-image.sh >"$TMP/out" 2>&1); then
  echo "ERROR: native image builder accepted a build without opt-in" >&2
  exit 1
fi
grep -q 'This image is experimental' "$TMP/out"

# The legacy packager must preserve dist instead of recursively deleting it.
if grep -Fq 'rm -rf "$DIST"' "$ROOT/scripts/build-image.sh"; then
  echo "ERROR: legacy image builder still deletes the whole dist directory" >&2
  exit 1
fi
grep -Fq 'Refusing to overwrite existing image' "$ROOT/scripts/build-image.sh"
grep -Fq 'Refusing to overwrite existing image' "$ROOT/scripts/build-native-image.sh"

echo "Native image builder safety checks passed."
