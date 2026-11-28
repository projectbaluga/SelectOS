//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config is gate.json, kept on the Gate partition/USB so it can be edited with Notepad.
// Every field is optional; missing fields fall back to sensible defaults.
type Config struct {
	Brand struct {
		Name     string `json:"name"`
		Subtitle string `json:"subtitle"`
		Logo     string `json:"logo"` // optional PNG on the Gate partition, e.g. icons/selectos.png
	} `json:"brand"`
	Domain         string   `json:"domain"`
	ShowAllEntries bool     `json:"show_all_entries"` // also show firmware/other entries
	Hide           []string `json:"hide"`             // hide entries whose name contains any of these
	Default        string   `json:"default"`          // pre-select the OS matching this text
	OS             []OSRule `json:"os"`
}

// OSRule customises one operating system. Only "match" is required.
type OSRule struct {
	Match    string `json:"match"`    // text found in the UEFI entry name (or "Boot0003" to pin by ID)
	Name     string `json:"name"`     // title shown on the card
	Subtitle string `json:"subtitle"` // small text under the title
	Icon     string `json:"icon"`     // windows|ubuntu|linux|terminal|disk or a PNG path like icons/kali.png
	Order    int    `json:"order"`    // 1 = first. Unset rules sort after ordered ones.
	Hidden   bool   `json:"hidden"`
}

func defaultConfig() *Config {
	c := &Config{Domain: "gbox.ncf.edu.ph"}
	c.Brand.Name, c.Brand.Subtitle = "NCF", "CCS Laboratory"
	return c
}

func loadConfig(base string) *Config {
	c := defaultConfig()
	paths := []string{"gate.json"}
	if base != "" {
		paths = []string{filepath.Join(base, "gate.json"), "gate.json"}
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var n Config
		n.Domain = c.Domain
		n.Brand = c.Brand
		if err := json.Unmarshal(b, &n); err != nil {
			fmt.Println("gate: gate.json is invalid, using defaults:", err)
			return c
		}
		fmt.Println("gate: loaded", p)
		return &n
	}
	return c
}

type distro struct{ key, name string }

var distros = []distro{
	{"debian", "Debian"}, {"fedora", "Fedora"}, {"kali", "Kali Linux"}, {"arch", "Arch Linux"},
	{"manjaro", "Manjaro"}, {"opensuse", "openSUSE"}, {"suse", "openSUSE"}, {"zorin", "Zorin OS"},
	{"elementary", "elementary OS"}, {"rocky", "Rocky Linux"}, {"almalinux", "AlmaLinux"},
	{"centos", "CentOS"}, {"redhat", "Red Hat"}, {"proxmox", "Proxmox"}, {"tails", "Tails"},
	{"clonezilla", "Clonezilla"}, {"parrot", "Parrot OS"}, {"nixos", "NixOS"}, {"gentoo", "Gentoo"},
	{"freebsd", "FreeBSD"}, {"linux", "Linux"},
}

func tokens(s string) map[string]bool {
	m := map[string]bool{}
	for _, t := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		m[t] = true
	}
	return m
}

// detect recognises common operating systems from a UEFI boot entry name.
func detect(desc string) (key, name string, kind int) {
	t := tokens(desc)
	switch {
	case t["windows"]:
		return "windows", "Windows", 1
	case t["ubuntu"]:
		return "ubuntu", "Ubuntu", 2
	case strings.Contains(strings.ToLower(desc), "linux mint") || t["mint"]:
		return "mint", "Linux Mint", 3
	}
	for _, d := range distros {
		if t[d.key] {
			return d.key, d.name, 3
		}
	}
	return "", desc, 0
}

func isFirmwareEntry(d string) bool {
	d = strings.ToLower(d)
	for _, p := range []string{"uefi ", "uefi:", "uiapp", "efi ", "pxe", "ipv4", "ipv6", "setup", "boot menu", "shell"} {
		if strings.HasPrefix(d, p) || strings.Contains(d, p) && strings.HasPrefix(d, "uefi") {
			return true
		}
	}
	return false
}

func loadIcon(base, rel string) image.Image {
	if base == "" || rel == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(base, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}

var builtinIcons = map[string]int{"windows": 1, "ubuntu": 2, "linux": 3, "terminal": 3, "disk": 0, "generic": 0}

// classify turns firmware boot entries into the cards shown to the student.
// Returns the cards and the index to pre-select.
func classify(es []BootEntry, cfg *Config, base string) ([]osItem, int) {
	var all []osItem
	for _, e := range es {
		d := strings.ToLower(e.Description)
		if strings.Contains(d, "gate") {
			continue
		}
		skip := false
		for _, h := range cfg.Hide {
			if h != "" && strings.Contains(d, strings.ToLower(h)) {
				skip = true
			}
		}
		if skip {
			continue
		}
		id := fmt.Sprintf("boot%04x", e.ID)
		key, name, kind := detect(e.Description)
		it := osItem{ID: e.ID, Name: name, Sub: e.Description, Kind: kind, Order: 1 << 20}
		if kind == 0 {
			it.Sub = fmt.Sprintf("Boot%04X", e.ID)
		}
		it.Known = kind != 0
		var rule *OSRule
		for i := range cfg.OS {
			m := strings.ToLower(strings.TrimSpace(cfg.OS[i].Match))
			if m != "" && (m == id || strings.Contains(d, m)) {
				rule = &cfg.OS[i]
				break
			}
		}
		if rule != nil {
			it.Known = true
			if rule.Hidden {
				continue
			}
			if rule.Name != "" {
				it.Name = rule.Name
			}
			if rule.Subtitle != "" {
				it.Sub = rule.Subtitle
			}
			if rule.Order > 0 {
				it.Order = rule.Order
			}
			if k, ok := builtinIcons[strings.ToLower(rule.Icon)]; ok {
				it.Kind = k
			} else if img := loadIcon(base, rule.Icon); img != nil {
				it.Img = img
			}
		}
		if it.Img == nil { // convention: icons/<key>.png is picked up automatically
			for _, k := range []string{key, strings.ToLower(strings.Fields(it.Name + " x")[0])} {
				if k == "" {
					continue
				}
				if img := loadIcon(base, "icons/"+k+".png"); img != nil {
					it.Img = img
					break
				}
			}
		}
		all = append(all, it)
	}
	// Show recognised/configured OSes only, unless nothing is recognised or show_all_entries is set.
	var shown []osItem
	for _, it := range all {
		if it.Known || cfg.ShowAllEntries {
			shown = append(shown, it)
		}
	}
	if len(shown) == 0 {
		for _, it := range all {
			if !isFirmwareEntry(it.Sub) && !isFirmwareEntry(it.Name) {
				shown = append(shown, it)
			}
		}
		if len(shown) == 0 {
			shown = all
		}
	}
	sort.SliceStable(shown, func(i, j int) bool {
		if shown[i].Order != shown[j].Order {
			return shown[i].Order < shown[j].Order
		}
		return shown[i].ID < shown[j].ID
	})
	if len(shown) > 6 {
		shown = shown[:6]
	}
	sel := 0
	if cfg.Default != "" {
		for i, it := range shown {
			if strings.Contains(strings.ToLower(it.Name+" "+it.Sub), strings.ToLower(cfg.Default)) {
				sel = i
				break
			}
		}
	}
	return shown, sel
}
