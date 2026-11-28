//go:build linux

package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"syscall"
	"unsafe"
)

type FB struct {
	f          *os.File
	mem        []byte
	buf        []byte
	W, H       int
	Stride     int
	rs, gs, bs uint
}

func ioctlPtr(fd, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

func ioctlVal(fd, req, val uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, val); e != 0 {
		return e
	}
	return nil
}

func openFB() (*FB, error) {
	var f *os.File
	var err error
	for _, p := range []string{"/dev/fb0", "/dev/fb/0"} {
		if f, err = os.OpenFile(p, os.O_RDWR, 0); err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	var v [40]uint32 // struct fb_var_screeninfo
	if err := ioctlPtr(f.Fd(), 0x4600, unsafe.Pointer(&v[0])); err != nil {
		return nil, err
	}
	var fix [80]byte // struct fb_fix_screeninfo
	if err := ioctlPtr(f.Fd(), 0x4602, unsafe.Pointer(&fix[0])); err != nil {
		return nil, err
	}
	if v[6] != 32 {
		return nil, fmt.Errorf("unsupported bpp %d", v[6])
	}
	stride := int(binary.LittleEndian.Uint32(fix[48:52]))
	size := int(binary.LittleEndian.Uint32(fix[24:28]))
	if need := stride * int(v[1]); size < need {
		size = need
	}
	mem, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return &FB{f: f, mem: mem, W: int(v[0]), H: int(v[1]), Stride: stride,
		rs: uint(v[8]), gs: uint(v[11]), bs: uint(v[14])}, nil
}

// Flush converts the canvas to the framebuffer pixel format and pushes it with
// write(): unlike plain mmap stores, this always marks the DRM fbdev buffer dirty,
// so the frame actually reaches the display.
func (fb *FB) Flush(img *image.RGBA) {
	if len(fb.buf) != fb.Stride*fb.H {
		fb.buf = make([]byte, fb.Stride*fb.H)
	}
	fast := fb.rs == 16 && fb.gs == 8 && fb.bs == 0
	for y := 0; y < fb.H; y++ {
		src := img.Pix[y*img.Stride : y*img.Stride+fb.W*4]
		dst := fb.buf[y*fb.Stride : y*fb.Stride+fb.W*4]
		if fast {
			for x := 0; x < len(src); x += 4 {
				dst[x], dst[x+1], dst[x+2], dst[x+3] = src[x+2], src[x+1], src[x], 0
			}
			continue
		}
		for x := 0; x < fb.W; x++ {
			v := uint32(src[x*4])<<fb.rs | uint32(src[x*4+1])<<fb.gs | uint32(src[x*4+2])<<fb.bs
			binary.LittleEndian.PutUint32(dst[x*4:], v)
		}
	}
	if _, err := fb.f.WriteAt(fb.buf, 0); err != nil { // fall back to the mmap
		copy(fb.mem, fb.buf)
	}
}

func (fb *FB) Close() { syscall.Munmap(fb.mem); fb.f.Close() }
