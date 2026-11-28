//go:build linux

package main

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	KRune = iota
	KEnter
	KBack
	KLeft
	KRight
	KUp
	KDown
	KEsc
	KTab
)

type Key struct {
	Kind int
	R    rune
}

// ttyGraphics puts the console in raw mode (no echo) for the graphical UI.
func ttyGraphics() (restore func()) {
	fd := os.Stdin.Fd()
	var old syscall.Termios
	haveTerm := ioctlPtr(fd, syscall.TCGETS, unsafe.Pointer(&old)) == nil
	if haveTerm {
		t := old
		t.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ISIG
		t.Iflag &^= syscall.ICRNL | syscall.IXON
		t.Cc[syscall.VMIN] = 1
		t.Cc[syscall.VTIME] = 0
		ioctlPtr(fd, syscall.TCSETS, unsafe.Pointer(&t))
	}
	// Note: we deliberately do NOT switch the VT to KD_GRAPHICS: if anything goes wrong the
	// console (and any error message) stays visible instead of a silent black screen.
	return func() {
		os.Stdout.WriteString("\033[?25h") // show the text cursor again
		if haveTerm {
			ioctlPtr(fd, syscall.TCSETS, unsafe.Pointer(&old))
		}
	}
}

func readKeys(ch chan<- Key) {
	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		for i := 0; i < n; {
			b := buf[i]
			switch {
			case b == 0x1b:
				if i+2 < n && (buf[i+1] == '[' || buf[i+1] == 'O') {
					j := i + 2
					for j < n-1 && ((buf[j] >= '0' && buf[j] <= '9') || buf[j] == ';') {
						j++
					}
					switch buf[j] {
					case 'A':
						ch <- Key{Kind: KUp}
					case 'B':
						ch <- Key{Kind: KDown}
					case 'C':
						ch <- Key{Kind: KRight}
					case 'D':
						ch <- Key{Kind: KLeft}
					}
					i = j + 1
				} else {
					ch <- Key{Kind: KEsc}
					i = n
				}
				continue
			case b == '\r' || b == '\n':
				ch <- Key{Kind: KEnter}
			case b == 0x7f || b == 0x08:
				ch <- Key{Kind: KBack}
			case b == '\t':
				ch <- Key{Kind: KTab}
			case b >= 0x20 && b < 0x7f:
				ch <- Key{Kind: KRune, R: rune(b)}
			}
			i++
		}
	}
}
