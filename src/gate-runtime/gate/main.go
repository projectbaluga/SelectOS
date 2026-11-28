//go:build linux

// Gate: boot-time login + OS selector. Graphical (framebuffer) UI with a text fallback.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	dry := flag.Bool("dry-run", false, "do not write BootNext or reboot")
	text := flag.Bool("text", false, "force the text-mode UI")
	flag.Parse()

	if os.Getpid() == 1 { // running as init inside the Gate initramfs
		setupInit()
		defer func() {
			emergencyShell()
			for {
				time.Sleep(time.Hour)
			}
		}()
	}
	// In initramfs efivarfs is not mounted yet; EBUSY means already mounted.
	if err := syscall.Mount("efivarfs", efivarsDir, "efivarfs", 0, ""); err != nil && err != syscall.EBUSY {
		fmt.Println("warning: efivarfs mount:", err)
	}
	cfg := loadConfig(espPath)
	allowedSuffix = ""
	if d := strings.TrimPrefix(cfg.Domain, "@"); d != "" { // empty domain = any name, no suffix shown
		allowedSuffix = "@" + d
	}
	if !*text && safeGUI(*dry, cfg) {
		return
	}
	runText(*dry)
}

// safeGUI runs the graphical UI; on any panic it prints the error and falls back to text mode.
func safeGUI(dry bool, cfg *Config) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\ngate: GUI crashed:", r)
			os.Stdout.Write(debug.Stack())
			time.Sleep(5 * time.Second)
			ok = false
		}
	}()
	return runGUI(dry, cfg)
}

func rebootNow() {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
}

func runGUI(dry bool, cfg *Config) bool {
	fb, err := openFB()
	if err != nil {
		fmt.Println("no framebuffer, using text mode:", err)
		return false
	}
	entries, err := ListBootEntries()
	if b, _ := os.ReadFile("/proc/cmdline"); strings.Contains(string(b), "gate.demo") { // design preview
		entries = []BootEntry{{1, "Windows Boot Manager"}, {2, "ubuntu"}, {3, "Kali Linux"}, {4, "Fedora"}, {5, "UEFI: PXE IPv4"}}
		err = nil
		dry = true // demo cards are fake: never write BootNext
	}
	items, sel := classify(entries, cfg, espPath)
	if err != nil || len(items) == 0 {
		fb.Close()
		return false
	}
	fmt.Println("gate: starting graphical UI")
	fmt.Printf("gate: framebuffer %dx%d stride=%d shifts=%d/%d/%d items=%d\n", fb.W, fb.H, fb.Stride, fb.rs, fb.gs, fb.bs, len(items))
	restore := ttyGraphics()
	defer restore()
	ui := newUI(fb.W, fb.H, cfg)
	keys := make(chan Key, 64)
	go readKeys(keys)

	local, errMsg, screen, caret := "", "", 0, true
	draw := func() {
		if screen == 0 {
			ui.drawLogin(local, errMsg, caret)
		} else {
			ui.drawPicker(local, items, sel)
		}
		fb.Flush(ui.c.img)
	}
	status := func(title, sub string, frames int, bad bool) {
		for f := 0; f < frames; f++ {
			ui.drawBoot(title, sub, f, bad)
			fb.Flush(ui.c.img)
			time.Sleep(110 * time.Millisecond)
		}
		for len(keys) > 0 {
			<-keys
		}
	}
	t0 := time.Now()
	draw()
	fmt.Printf("gate: first frame drawn in %v\n", time.Since(t0))
	os.Stdout.WriteString("\033[?25l") // hide the blinking text cursor over the UI
	tick := time.NewTicker(550 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if screen == 0 {
				caret = !caret
				draw()
			}
			continue
		case k := <-keys:
			if screen == 0 {
				switch {
				case k.Kind == KRune:
					c := k.R
					if c >= 'A' && c <= 'Z' {
						c += 32
					}
					if (c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') && len(local) < 40 {
						local += string(c)
						errMsg = ""
					}
				case k.Kind == KBack && len(local) > 0:
					local = local[:len(local)-1]
					errMsg = ""
				case k.Kind == KEnter:
					if len(local) < 3 {
						errMsg = "Please enter your Gbox username (at least 3 characters)."
					} else {
						screen = 1
					}
				}
				caret = true
			} else {
				switch {
				case k.Kind == KLeft || (k.Kind == KTab && false):
					sel = (sel + len(items) - 1) % len(items)
				case k.Kind == KRight || k.Kind == KTab:
					sel = (sel + 1) % len(items)
				case k.Kind == KEsc:
					screen, local = 0, ""
				case k.Kind == KRune && k.R >= '1' && int(k.R-'0') <= len(items):
					sel = int(k.R - '1')
				case k.Kind == KEnter:
					it := items[sel]
					if dry {
						status("Starting "+it.Name, "Dry run: BootNext would be set to "+it.Sub, 14, false)
						screen, local = 0, ""
						break
					}
					if err := SetBootNext(it.ID); err != nil {
						status("Could not start "+it.Name, err.Error(), 30, true)
						break
					}
					status("Starting "+it.Name, "Please wait...", 14, false)
					rebootNow()
					select {}
				}
			}
			draw()
		}
	}
}

// ---- text-mode fallback -----------------------------------------------------

func runText(dry bool) {
	in := bufio.NewReader(os.Stdin)
	fmt.Println("=== NCF CCS Lab Gate ===")
	email := ""
	for {
		fmt.Print("gbox email: ")
		line, _ := in.ReadString('\n')
		email = strings.ToLower(strings.TrimSpace(line))
		if strings.HasSuffix(email, allowedSuffix) {
			break
		}
		fmt.Println("Gbox email lang po (" + allowedSuffix + ")")
	}
	fmt.Println("Hello,", email)
	entries, err := ListBootEntries()
	if err != nil || len(entries) == 0 {
		fmt.Println("Walang boot entries:", err)
		return
	}
	var choices []BootEntry
	for _, e := range entries {
		if !strings.Contains(strings.ToLower(e.Description), "gate") {
			choices = append(choices, e)
		}
	}
	for {
		fmt.Println("\nPumili ng OS:")
		for i, e := range choices {
			fmt.Printf("  [%d] %s (Boot%04X)\n", i+1, e.Description, e.ID)
		}
		fmt.Print("> ")
		line, _ := in.ReadString('\n')
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || n < 1 || n > len(choices) {
			continue
		}
		c := choices[n-1]
		if dry {
			fmt.Printf("[dry-run] BootNext=%04X (%s)\n", c.ID, c.Description)
			return
		}
		if err := SetBootNext(c.ID); err != nil {
			fmt.Println("BootNext failed:", err)
			return
		}
		fmt.Println("Booting", c.Description, "...")
		rebootNow()
		return
	}
}
