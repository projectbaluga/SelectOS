//go:build linux

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf16"
)

const (
	efiGlobalGUID = "8be4df61-93ca-11d2-aa0d-00e098032b8c"
	efivarsDir    = "/sys/firmware/efi/efivars"
	attrNVBSRT    = 0x07 // NON_VOLATILE | BOOTSERVICE_ACCESS | RUNTIME_ACCESS
)

type BootEntry struct {
	ID          uint16
	Description string
}

// ListBootEntries reads Boot#### UEFI variables and returns their descriptions.
func ListBootEntries() ([]BootEntry, error) {
	pattern := filepath.Join(efivarsDir, "Boot[0-9A-F][0-9A-F][0-9A-F][0-9A-F]-"+efiGlobalGUID)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var out []BootEntry
	for _, f := range files {
		var tmp uint32
		if _, err := fmt.Sscanf(filepath.Base(f)[4:8], "%04X", &tmp); err != nil {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil || len(data) < 4+4+2+2 {
			continue
		}
		// 4 bytes var attrs | EFI_LOAD_OPTION: attrs(4) pathLen(2) description(UTF-16, NUL-terminated)
		raw := data[4+6:]
		var u []uint16
		for i := 0; i+1 < len(raw); i += 2 {
			c := binary.LittleEndian.Uint16(raw[i:])
			if c == 0 {
				break
			}
			u = append(u, c)
		}
		out = append(out, BootEntry{ID: uint16(tmp), Description: string(utf16.Decode(u))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// SetBootNext sets the one-time next boot entry (cleared by firmware after use).
func SetBootNext(id uint16) error {
	path := filepath.Join(efivarsDir, "BootNext-"+efiGlobalGUID)
	buf := make([]byte, 6)
	binary.LittleEndian.PutUint32(buf[0:4], attrNVBSRT)
	binary.LittleEndian.PutUint16(buf[4:6], id)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buf) // efivarfs needs attrs + data in ONE write()
	return err
}
