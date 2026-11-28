//go:build linux

package main

import (
	"fmt"
	"image/png"
	"os"
	"testing"
	"time"
)

func save(u *UI, name string) {
	f, _ := os.Create(name)
	png.Encode(f, u.c.img)
	f.Close()
}

func TestRender(t *testing.T) {
	for _, res := range [][2]int{{1280, 800}, {1920, 1080}, {1366, 768}} {
		t0 := time.Now()
		cfg := defaultConfig()
		if _, err := os.Stat(os.Getenv("LOGO_PNG")); err == nil { // optional: preview with the brand logo
			os.MkdirAll("/tmp/espt/icons", 0o755)
			b, _ := os.ReadFile(os.Getenv("LOGO_PNG"))
			os.WriteFile("/tmp/espt/icons/selectos.png", b, 0o644)
			espPath, cfg.Brand.Logo = "/tmp/espt", "icons/selectos.png"
			cfg.Brand.Name, cfg.Brand.Subtitle = "SelectOS", "Multi-boot Selector"
		}
		u := newUI(res[0], res[1], cfg)
		fmt.Println("init", res, time.Since(t0))
		es := []BootEntry{{1, "Windows Boot Manager"}, {2, "ubuntu"}, {3, "Kali Linux"}, {4, "Fedora"}, {5, "UEFI: PXE IPv4"}}
		items, _ := classify(es, cfg, "")
		t0 = time.Now()
		u.drawLogin("juan.delacruz", "", true)
		fmt.Println("login draw", time.Since(t0))
		save(u, fmt.Sprintf("/tmp/ui_login_%d.png", res[0]))
		t0 = time.Now()
		u.drawPicker("juan.delacruz", items, 1)
		fmt.Println("picker draw", time.Since(t0))
		save(u, fmt.Sprintf("/tmp/ui_picker_%d.png", res[0]))
		u.drawBoot("Starting Ubuntu", "Please wait...", 3, false)
		save(u, fmt.Sprintf("/tmp/ui_boot_%d.png", res[0]))
	}
}
