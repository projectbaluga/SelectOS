#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$(mktemp -d)"
cleanup() { rm -rf -- "$WORK"; }
trap cleanup EXIT

for cmd in gcc ld objcopy qemu-system-x86_64 mkfs.vfat mmd mcopy mtype python3 timeout; do
  command -v "$cmd" >/dev/null 2>&1 || {
    echo "ERROR: Missing QEMU/OVMF test dependency: $cmd" >&2
    exit 1
  }
done
OVMF_CODE="${SELECTOS_OVMF_CODE:-}"
OVMF_VARS="${SELECTOS_OVMF_VARS:-}"
if [[ -z "$OVMF_CODE" || -z "$OVMF_VARS" ]]; then
  for suffix in _4M ""; do
    candidate_code="/usr/share/OVMF/OVMF_CODE${suffix}.fd"
    candidate_vars="/usr/share/OVMF/OVMF_VARS${suffix}.fd"
    if [[ -s "$candidate_code" && -s "$candidate_vars" ]]; then
      OVMF_CODE="$candidate_code"
      OVMF_VARS="$candidate_vars"
      break
    fi
  done
fi
[[ -s "$OVMF_CODE" && -s "$OVMF_VARS" ]] || {
  echo "ERROR: Matching OVMF_CODE and OVMF_VARS firmware images were not found." >&2
  exit 1
}
cp "$OVMF_VARS" "$WORK/OVMF_VARS.fd"

LDS="$(find /usr/lib /usr/lib64 -type f -name elf_x86_64_efi.lds -print -quit 2>/dev/null)"
CRT="$(find /usr/lib /usr/lib64 -type f -name crt0-efi-x86_64.o -print -quit 2>/dev/null)"
[[ -n "$LDS" && -n "$CRT" ]] || { echo "ERROR: GNU-EFI linker files not found." >&2; exit 1; }
LIBDIR="$(dirname "$LDS")"
[[ -s "$LIBDIR/libefi.a" && -s "$LIBDIR/libgnuefi.a" ]] || { echo "ERROR: GNU-EFI libraries missing." >&2; exit 1; }

build_efi() {
  local src="$1" out="$2" name="$3"
  local extra_flags=()
  if [[ "$name" == "boot-manager" ]]; then
    extra_flags+=(-DSELECTOS_QEMU_TEST_DIAGNOSTICS)
  fi
  gcc -I/usr/include/efi -I/usr/include/efi/x86_64 -I/usr/include/efi/protocol \
    -fpic -fshort-wchar -mno-red-zone -maccumulate-outgoing-args -fno-stack-protector \
    -march=x86-64 -mtune=generic -mno-sse4.1 -mno-sse4.2 -mno-avx -mno-avx2 \
    -DEFI_FUNCTION_WRAPPER "${extra_flags[@]}" -c "$src" -o "$WORK/$name.o"
  ld -nostdlib -znocombreloc -T "$LDS" -shared -Bsymbolic \
    "$CRT" "$WORK/$name.o" -L"$LIBDIR" -lefi -lgnuefi -o "$WORK/$name.so"
  objcopy -j .text -j .sdata -j .data -j .dynamic -j .dynsym \
    -j .rel -j .rela -j .reloc -I elf64-x86-64 -O efi-app-x86_64 \
    "$WORK/$name.so" "$out"
  file "$out" | grep -q 'PE32+.*EFI.*application.*x86-64'
}
build_efi "$ROOT/src/uefi/boot-manager.c" "$WORK/selectos.efi" boot-manager
build_efi "$ROOT/tests/uefi-handoff-child.c" "$WORK/grubx64.efi" handoff-child

# Use the known-good GRUB EFI binary extracted from the repository's
# compatibility image as the firmware fallback. GRUB then chainloads the
# freshly-built SelectOS manager, isolating the manager test from OVMF's
# direct loading of a GNU-EFI PE/COFF binary.
[[ -s "$ROOT/EFI/BOOT/grubx64.efi" ]] || {
  echo "ERROR: Expected GRUB EFI bootstrap binary is missing." >&2
  exit 1
}
cp "$ROOT/EFI/BOOT/grubx64.efi" "$WORK/BOOTX64.EFI"
cat > "$WORK/grub.cfg" <<'GRUB'
set timeout=0
set default=0
echo SELECTOS_OVMF_GRUB_CONFIG_LOADED
set root=(hd0)
echo SELECTOS_GRUB_ROOT=$root
ls /EFI/SelectOS/
chainloader /EFI/SelectOS/selectos.efi
boot
GRUB

truncate -s 64M "$WORK/uefi-test.img"
# A whole-device FAT32 removable-media volume exercises the UEFI fallback
# path without depending on GPT/ESP partition auto-discovery.
FAT_IMAGE="$WORK/uefi-test.img"
mkfs.vfat -F 32 -n SELECTOS "$FAT_IMAGE"
mmd -i "$FAT_IMAGE" ::/EFI
mmd -i "$FAT_IMAGE" ::/EFI/BOOT
mmd -i "$FAT_IMAGE" ::/EFI/SelectOS
mmd -i "$FAT_IMAGE" ::/EFI/ubuntu
mmd -i "$FAT_IMAGE" ::/boot
mmd -i "$FAT_IMAGE" ::/boot/grub
mcopy -i "$FAT_IMAGE" "$WORK/BOOTX64.EFI" ::/EFI/BOOT/BOOTX64.EFI
mcopy -i "$FAT_IMAGE" "$WORK/grub.cfg" ::/EFI/BOOT/grub.cfg
mcopy -i "$FAT_IMAGE" "$WORK/grub.cfg" ::/boot/grub/grub.cfg
mcopy -i "$FAT_IMAGE" "$WORK/selectos.efi" ::/EFI/SelectOS/selectos.efi
mcopy -i "$FAT_IMAGE" "$WORK/grubx64.efi" ::/EFI/ubuntu/grubx64.efi
mdir -i "$FAT_IMAGE" ::/EFI/BOOT
mdir -i "$FAT_IMAGE" ::/EFI/SelectOS

python3 - "$WORK/uefi-test.img" "$OVMF_CODE" "$WORK/OVMF_VARS.fd" "$WORK/serial.log" <<'PY'
import json, os, socket, subprocess, sys, tempfile, time
image, firmware, variables, serial_log = sys.argv[1:5]
qmp_path = os.path.join(tempfile.gettempdir(), f"selectos-qmp-{os.getpid()}.sock")
cmd = [
    "qemu-system-x86_64", "-machine", "pc,accel=tcg", "-m", "512",
    "-drive", f"if=pflash,format=raw,readonly=on,file={firmware}",
    "-drive", f"if=pflash,format=raw,file={variables}",
    "-display", "none", "-serial", f"file:{serial_log}",
    "-monitor", "none", "-qmp", f"unix:{qmp_path},server=on,wait=off",
    # Whole-device FAT32 USB media exercises the standard UEFI
    # removable-media fallback path (\\EFI\\BOOT\\BOOTX64.EFI).
    "-device", "qemu-xhci,id=xhci",
    "-drive", f"if=none,id=stick,format=raw,file={image}",
    "-device", "usb-storage,drive=stick,bus=xhci.0,bootindex=1",
    "-boot", "order=c",
    "-no-reboot", "-no-shutdown",
]
proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
sock.settimeout(10)
try:
    deadline = time.time() + 20
    while time.time() < deadline:
        if proc.poll() is not None:
            raise RuntimeError("QEMU exited before QMP became ready")
        try:
            sock.connect(qmp_path)
            break
        except (FileNotFoundError, ConnectionRefusedError):
            time.sleep(0.2)
    else:
        raise RuntimeError("Timed out waiting for QEMU QMP socket")
    stream = sock.makefile("rwb", buffering=0)
    greeting = json.loads(stream.readline())
    if "QMP" not in greeting:
        raise RuntimeError(f"Unexpected QMP greeting: {greeting}")
    def qmp(command, arguments=None):
        payload = {"execute": command}
        if arguments is not None:
            payload["arguments"] = arguments
        stream.write((json.dumps(payload) + "\r\n").encode())
        while True:
            response = json.loads(stream.readline())
            if "return" in response:
                return response["return"]
            if "error" in response:
                raise RuntimeError(f"QMP {command} failed: {response['error']}")
    qmp("qmp_capabilities")
    # Allow OVMF to enumerate the GPT disk and launch the fallback EFI application.
    time.sleep(12)
    try:
        qmp("human-monitor-command", {"command-line": "sendkey 1"})
        time.sleep(0.3)
        qmp("human-monitor-command", {"command-line": "sendkey ret"})
    except (BrokenPipeError, ConnectionResetError, TimeoutError):
        # The FAT marker is the authoritative assertion that the selected child ran.
        pass
    # The child app writes a marker to the same FAT volume before returning.
    time.sleep(5)
finally:
    try:
        sock.close()
    except Exception:
        pass
    if proc.poll() is None:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)
    _, stderr = proc.communicate()
    if stderr:
        print("QEMU stderr: " + stderr.decode(errors="replace"), file=sys.stderr)
    if os.path.exists(serial_log):
        with open(serial_log, "r", errors="replace") as serial:
            output = serial.read()
        if output:
            print("QEMU serial output:\\n" + output, file=sys.stderr)
    try:
        os.unlink(qmp_path)
    except FileNotFoundError:
        pass
PY

if ! mtype -i "$FAT_IMAGE" ::/handoff.ok > "$WORK/handoff.txt" 2>/dev/null; then
  echo "ERROR: QEMU/OVMF did not create the child-loader handoff marker." >&2
  echo "The firmware boot, menu input, or LoadImage/StartImage path did not complete." >&2
  if mtype -i "$FAT_IMAGE" ::/selectos-debug.log > "$WORK/manager-debug.txt" 2>/dev/null; then
    echo "SelectOS test-only filesystem diagnostics:" >&2
    cat "$WORK/manager-debug.txt" >&2
  else
    echo "No SelectOS manager diagnostic file was written; firmware may not have launched BOOTX64.EFI." >&2
  fi
  exit 1
fi
grep -qx 'SELECTOS_QEMU_OVMF_HANDOFF_OK' "$WORK/handoff.txt"
echo "QEMU/OVMF UEFI handoff integration test passed."
