//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// espPath is where the Gate partition (the FAT volume holding gate.marker and gate.json) is mounted.
var espPath string

func loadModule(name string) error {
	data, err := os.ReadFile(filepath.Join("/modules", name+".ko"))
	if err != nil {
		return err
	}
	params, _ := syscall.BytePtrFromString("")
	_, _, e := syscall.Syscall(syscall.SYS_INIT_MODULE, uintptr(unsafePtr(data)), uintptr(len(data)), uintptr(unsafePtrB(params)))
	if e != 0 && e != syscall.EEXIST {
		return e
	}
	return nil
}

// setupInit prepares the minimal system when Gate runs as PID 1.
func setupInit() {
	syscall.Mount("proc", "/proc", "proc", 0, "")
	syscall.Mount("sysfs", "/sys", "sysfs", 0, "")
	syscall.Mount("devtmpfs", "/dev", "devtmpfs", 0, "")
	if f, err := os.Open("/modules/order"); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if m := strings.TrimSpace(sc.Text()); m != "" {
				loadModule(m) // best effort: errors (already built in / no such hardware) are fine
			}
		}
		f.Close()
	}
	time.Sleep(2 * time.Second) // let USB keyboards and disks enumerate
	espPath = findESP()
	if f, err := os.OpenFile("/dev/console", os.O_RDWR, 0); err == nil {
		os.Stdin, os.Stdout, os.Stderr = f, f, f
	}
}

// emergencyShell keeps PID 1 alive and gives a shell for debugging.
func emergencyShell() {
	for {
		fmt.Println("\n[Gate] Emergency shell. Type 'exit' to return to Gate.")
		c := exec.Command("/bin/sh")
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		c.Run()
		return
	}
}

// findESP mounts (read-only) the FAT volume that contains gate.marker, waiting a few seconds
// for USB/SATA/NVMe devices to appear. Returns "" if none is found (Gate then uses defaults).
func findESP() string {
	os.MkdirAll("/esp", 0o755)
	tried := map[string]bool{}
	for deadline := time.Now().Add(6 * time.Second); time.Now().Before(deadline); time.Sleep(400 * time.Millisecond) {
		ents, _ := os.ReadDir("/sys/class/block")
		for _, e := range ents {
			n := e.Name()
			if strings.HasPrefix(n, "loop") || strings.HasPrefix(n, "ram") || tried[n] {
				continue
			}
			dev := "/dev/" + n
			if _, err := os.Stat(dev); err != nil {
				continue
			}
			if syscall.Mount(dev, "/esp", "vfat", syscall.MS_RDONLY, "iocharset=cp437") == nil {
				if _, err := os.Stat("/esp/gate.marker"); err == nil {
					return "/esp"
				}
				syscall.Unmount("/esp", 0)
			}
			tried[n] = true
		}
	}
	return ""
}
