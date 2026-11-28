#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$ROOT/src/uefi/boot-manager.c"
OUT="${SELECTOS_UEFI_LOADER_OUT:-$ROOT/dist/native/selectos.efi}"
WORK="$(mktemp -d)"
TMP_OUT="${OUT}.tmp.${BASHPID}"
trap 'rm -rf "$WORK"; rm -f -- "$TMP_OUT"' EXIT
for cmd in gcc ld objcopy file find; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "ERROR: Missing UEFI build dependency: $cmd" >&2
    echo "Debian/Ubuntu: sudo apt-get install gcc binutils gnu-efi" >&2
    exit 1
  }
done
[[ -s "$SRC" ]] || { echo "ERROR: UEFI boot-manager source is missing." >&2; exit 1; }
[[ -d /usr/include/efi/x86_64 ]] || { echo "ERROR: GNU-EFI headers not found; install gnu-efi." >&2; exit 1; }
LDS="$(find /usr/lib /usr/lib64 -type f -name elf_x86_64_efi.lds -print -quit 2>/dev/null)"
CRT="$(find /usr/lib /usr/lib64 -type f -name crt0-efi-x86_64.o -print -quit 2>/dev/null)"
[[ -n "$LDS" && -n "$CRT" ]] || { echo "ERROR: GNU-EFI linker script or x86_64 CRT object not found." >&2; exit 1; }
LIBDIR="$(dirname "$LDS")"
[[ -s "$LIBDIR/libefi.a" && -s "$LIBDIR/libgnuefi.a" ]] || { echo "ERROR: GNU-EFI libraries not found beside $LDS." >&2; exit 1; }
mkdir -p "$(dirname "$OUT")"
[[ ! -e "$OUT" ]] || { echo "ERROR: Refusing to overwrite UEFI loader: $OUT" >&2; exit 1; }
gcc -I/usr/include/efi -I/usr/include/efi/x86_64 -I/usr/include/efi/protocol \
  -fpic -fshort-wchar -mno-red-zone -maccumulate-outgoing-args -fno-stack-protector \
  -DEFI_FUNCTION_WRAPPER -c "$SRC" -o "$WORK/boot-manager.o"
ld -nostdlib -znocombreloc -T "$LDS" -shared -Bsymbolic \
  "$CRT" "$WORK/boot-manager.o" -L"$LIBDIR" -lefi -lgnuefi -o "$WORK/boot-manager.so"
objcopy -j .text -j .sdata -j .data -j .dynamic -j .dynsym \
  -j .rel -j .rela -j .reloc --target=efi-app-x86_64 "$WORK/boot-manager.so" "$TMP_OUT"
file "$TMP_OUT" | grep -q 'PE32+.*EFI application.*x86-64' || {
  echo "ERROR: Output is not an x86_64 UEFI application." >&2
  file "$TMP_OUT" >&2
  exit 1
}
mv -- "$TMP_OUT" "$OUT"
echo "UEFI boot manager built: $OUT"
file "$OUT"
