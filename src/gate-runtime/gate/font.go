//go:build linux

package main

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/binary"
	"io"
	"unicode/utf8"
)

//go:embed assets/*.psf.gz
var fontFS embed.FS

// Font is a PSF1/PSF2 bitmap font (Terminus, SIL OFL) with its unicode map.
type Font struct {
	W, H   int
	rowB   int
	glyphs [][]byte
	idx    map[rune]int
}

func loadFont(name string) *Font {
	raw, err := fontFS.ReadFile("assets/" + name + ".psf.gz")
	if err != nil {
		panic(err)
	}
	zr, _ := gzip.NewReader(bytes.NewReader(raw))
	d, _ := io.ReadAll(zr)
	f := &Font{idx: map[rune]int{}}
	var n, charsize, hdr int
	hasTab := false
	psf2 := false
	if len(d) > 4 && d[0] == 0x72 && d[1] == 0xb5 && d[2] == 0x4a && d[3] == 0x86 {
		psf2 = true
		hdr = int(binary.LittleEndian.Uint32(d[8:]))
		hasTab = binary.LittleEndian.Uint32(d[12:])&1 != 0
		n = int(binary.LittleEndian.Uint32(d[16:]))
		charsize = int(binary.LittleEndian.Uint32(d[20:]))
		f.H = int(binary.LittleEndian.Uint32(d[24:]))
		f.W = int(binary.LittleEndian.Uint32(d[28:]))
	} else { // PSF1
		mode := d[2]
		n = 256
		if mode&1 != 0 {
			n = 512
		}
		hasTab = mode&2 != 0
		hdr = 4
		charsize = int(d[3])
		f.H, f.W = charsize, 8
	}
	f.rowB = (f.W + 7) / 8
	for i := 0; i < n; i++ {
		f.glyphs = append(f.glyphs, d[hdr+i*charsize:hdr+(i+1)*charsize])
	}
	if hasTab {
		p := hdr + n*charsize
		for i := 0; i < n && p < len(d); i++ {
			if psf2 {
				start := p
				for p < len(d) && d[p] != 0xff {
					p++
				}
				seg := d[start:p]
				if k := bytes.IndexByte(seg, 0xfe); k >= 0 {
					seg = seg[:k]
				}
				for len(seg) > 0 {
					r, sz := utf8.DecodeRune(seg)
					seg = seg[sz:]
					if _, ok := f.idx[r]; !ok {
						f.idx[r] = i
					}
				}
				p++
			} else {
				for p+1 < len(d) {
					v := binary.LittleEndian.Uint16(d[p:])
					p += 2
					if v == 0xffff {
						break
					}
					if v == 0xfffe {
						for p+1 < len(d) && binary.LittleEndian.Uint16(d[p:]) != 0xffff {
							p += 2
						}
						continue
					}
					if _, ok := f.idx[rune(v)]; !ok {
						f.idx[rune(v)] = i
					}
				}
			}
		}
	} else {
		for i := 0; i < n && i < 256; i++ {
			f.idx[rune(i)] = i
		}
	}
	return f
}

func (f *Font) glyph(r rune) []byte {
	if i, ok := f.idx[r]; ok {
		return f.glyphs[i]
	}
	if i, ok := f.idx['?']; ok {
		return f.glyphs[i]
	}
	return f.glyphs[0]
}

func (f *Font) has(r rune) bool { _, ok := f.idx[r]; return ok }
