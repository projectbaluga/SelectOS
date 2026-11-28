#!/usr/bin/env python3
import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "discover-efi-entries.py"
spec = importlib.util.spec_from_file_location("discover_efi", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class DiscoveryParsingTests(unittest.TestCase):
    def test_export_parser_preserves_values(self):
        data = module.parse_export('TYPE="vfat"\nPART_ENTRY_TYPE=c12a7328-f81f-11d2-ba4b-00a0c93ec93b\nLABEL="EFI SYSTEM"\n')
        self.assertEqual(data["TYPE"], "vfat")
        self.assertEqual(data["LABEL"], "EFI SYSTEM")

    def test_recognizes_esp_guid_case_insensitively(self):
        self.assertTrue(module.is_esp({"PART_ENTRY_TYPE": "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"}))

    def test_rejects_non_esp_partition(self):
        self.assertFalse(module.is_esp({"PART_ENTRY_TYPE": "0fc63daf-8483-4772-8e79-3d69d8477de4"}))
        self.assertFalse(module.is_esp({}))

    def test_empty_sysfs_returns_no_partitions(self):
        with tempfile.TemporaryDirectory() as temp:
            self.assertEqual(module.discover(Path(temp) / "missing", Path(temp)), [])

if __name__ == "__main__":
    unittest.main()
