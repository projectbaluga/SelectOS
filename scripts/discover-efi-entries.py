#!/usr/bin/env python3
"""Read-only GPT EFI System Partition and boot-loader discovery prototype.

This is a host-side phase-2 tool; it is not invoked by PID 1 yet. By default it
only reads sysfs and blkid metadata. Pass --mount to temporarily mount ESPs
read-only and enumerate known loader paths.
"""
from __future__ import annotations
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

ESP_GUID = "c12a7328-f81f-11d2-ba4b-00a0c93ec93b"
KNOWN_LOADERS = (
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
)

def parse_export(text: str) -> dict[str, str]:
    result = {}
    for line in text.splitlines():
        if "=" in line:
            key, value = line.split("=", 1)
            result[key.strip()] = value.strip().strip('"')
    return result

def is_esp(metadata: dict[str, str]) -> bool:
    return metadata.get("PART_ENTRY_TYPE", "").strip("{}").lower() == ESP_GUID

def discover(sys_block: Path, dev_root: Path, do_mount: bool = False) -> list[dict]:
    results = []
    for node in sorted(sys_block.iterdir()) if sys_block.exists() else []:
        if not (node / "partition").exists():
            continue
        device = dev_root / node.name
        probe = subprocess.run(
            ["blkid", "-o", "export", str(device)],
            text=True, capture_output=True, check=False,
        )
        metadata = parse_export(probe.stdout)
        if not is_esp(metadata):
            continue
        item = {
            "device": str(device),
            "partition_uuid": metadata.get("PART_ENTRY_UUID", ""),
            "filesystem": metadata.get("TYPE", ""),
            "label": metadata.get("LABEL", ""),
            "loaders": [],
            "mounted_read_only": False,
        }
        if do_mount:
            with tempfile.TemporaryDirectory(prefix="selectos-esp-") as temp:
                mounted = subprocess.run(
                    ["mount", "-o", "ro,nosuid,nodev,noexec", str(device), temp],
                    text=True, capture_output=True, check=False,
                )
                if mounted.returncode == 0:
                    item["mounted_read_only"] = True
                    root = Path(temp)
                    item["loaders"] = [
                        path for path in KNOWN_LOADERS if (root / path).is_file()
                    ]
                    subprocess.run(["umount", temp], check=False, capture_output=True)
                else:
                    item["mount_error"] = mounted.stderr.strip() or "mount failed"
        results.append(item)
    return results

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sys-block", default="/sys/class/block")
    parser.add_argument("--dev-root", default="/dev")
    parser.add_argument("--mount", action="store_true",
                        help="temporarily mount detected ESPs read-only to list known loaders")
    args = parser.parse_args()
    try:
        found = discover(Path(args.sys_block), Path(args.dev_root), args.mount)
    except (OSError, ValueError) as exc:
        print(f"selectos-efi-discover: {exc}", file=sys.stderr)
        return 1
    print(json.dumps({"esp_count": len(found), "partitions": found}, indent=2))
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
