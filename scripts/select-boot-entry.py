#!/usr/bin/env python3
"""Safe, keyboard-driven prototype for choosing an enumerated EFI loader.

Reads discovery JSON from stdin or --input. It never executes a loader or
accepts a path from keyboard input; the only accepted choice is a menu index.
"""
from __future__ import annotations
import argparse
import json
import sys
from pathlib import PurePosixPath

ALLOWED = {
    "EFI/Microsoft/Boot/bootmgfw.efi",
    "EFI/Boot/bootx64.efi",
    "EFI/Boot/bootaa64.efi",
    "EFI/ubuntu/shimx64.efi",
    "EFI/ubuntu/grubx64.efi",
    "EFI/debian/shimx64.efi",
    "EFI/debian/grubx64.efi",
    "EFI/fedora/shimx64.efi",
    "EFI/fedora/grubx64.efi",
    "EFI/systemd/systemd-bootx64.efi",
    "EFI/systemd/systemd-bootaa64.efi",
}

def safe_entries(document: dict) -> list[dict]:
    result = []
    for partition in document.get("partitions", []):
        device = partition.get("device")
        if not isinstance(device, str) or not device.startswith("/dev/"):
            continue
        if ".." in PurePosixPath(device).parts:
            continue
        for loader in partition.get("loaders", []):
            if loader not in ALLOWED:
                continue
            family = "Windows" if loader.startswith("EFI/Microsoft/") else "Linux / EFI"
            result.append({
                "device": device,
                "partition_uuid": str(partition.get("partition_uuid", "")),
                "label": str(partition.get("label", "")),
                "os": family,
                "loader": loader,
            })
    return result

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", help="discovery JSON file (default: stdin)")
    args = parser.parse_args()
    try:
        source = open(args.input, encoding="utf-8") if args.input else sys.stdin
        with source if args.input else _nullcontext(source) as stream:
            document = json.load(stream)
    except (OSError, json.JSONDecodeError) as exc:
        print(f"selectos-selector: cannot read discovery data: {exc}", file=sys.stderr)
        return 2

    entries = safe_entries(document)
    if not entries:
        print("Walang kilalang EFI boot entry. Bumalik sa firmware boot menu.")
        return 1
    print("SelectOS — piliin ang operating system")
    for index, entry in enumerate(entries, 1):
        label = f" ({entry['label']})" if entry["label"] else ""
        print(f"  {index}. {entry['os']} — {entry['device']}{label} — {entry['loader']}")
    print("  q. Kanselahin")
    try:
        choice = input("Piliin ang numero: ").strip().lower()
    except EOFError:
        return 2
    if choice == "q":
        return 0
    if not choice.isdigit() or not (1 <= int(choice) <= len(entries)):
        print("Hindi valid ang pinili; walang binagong boot setting.", file=sys.stderr)
        return 2
    # Output is a selection record only. A separate, audited UEFI handoff is
    # required; this prototype deliberately does not execute or chainload it.
    print(json.dumps(entries[int(choice) - 1], ensure_ascii=False))
    return 0

class _nullcontext:
    def __init__(self, value):
        self.value = value
    def __enter__(self):
        return self.value
    def __exit__(self, *exc):
        return False

if __name__ == "__main__":
    raise SystemExit(main())
