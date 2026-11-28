#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "select-boot-entry.py"
spec = importlib.util.spec_from_file_location("select_boot_entry", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class SelectorSafetyTests(unittest.TestCase):
    def test_only_allowlisted_loader_paths_are_exposed(self):
        doc = {"partitions": [{"device": "/dev/nvme0n1p1", "loaders": [
            "EFI/Microsoft/Boot/bootmgfw.efi", "../../etc/passwd", "EFI/custom/run.efi"
        ]}]}
        entries = module.safe_entries(doc)
        self.assertEqual(len(entries), 1)
        self.assertEqual(entries[0]["os"], "Windows")

    def test_rejects_device_path_traversal_and_non_device(self):
        doc = {"partitions": [
            {"device": "/dev/../etc/sda", "loaders": ["EFI/Boot/bootx64.efi"]},
            {"device": "sda1", "loaders": ["EFI/Boot/bootx64.efi"]},
        ]}
        self.assertEqual(module.safe_entries(doc), [])

    def test_does_not_trust_missing_or_wrong_types(self):
        self.assertEqual(module.safe_entries({}), [])
        self.assertEqual(module.safe_entries({"partitions": [{"device": 7, "loaders": ["EFI/Boot/bootx64.efi"]}]}), [])

if __name__ == "__main__":
    unittest.main()
