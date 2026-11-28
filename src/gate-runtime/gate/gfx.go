//go:build linux

package main

import (
	"image"
	"math"
)

type RGB struct{ R, G, B uint8 }

func lerp(a, b uint8, t float64) uint8 { return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5) }
func mix(a, b RGB, t float64) RGB {
	return RGB{lerp(a.R, b.R, t), lerp(a.G, b.G, t), lerp(a.B, b.B, t)}
}
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

type Canvas struct {
	img  *image.RGBA
	bg   *image.RGBA
	W, H int
}

func newCanvas(w, h int) *Canvas {
	return &Canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), W: w, H: h}
}

// makeBackground renders a deep-navy gradient with a soft accent glow (once).
func (c *Canvas) makeBackground(top, bottom, glow RGB) {
	c.bg = image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	gx, gy := float64(c.W)*0.5, -float64(c.H)*0.15
	R := float64(c.W) * 0.75
	for y := 0; y < c.H; y++ {
		ty := float64(y) / float64(c.H)
		for x := 0; x < c.W; x++ {
			base := mix(top, bottom, clamp01(ty*0.85+float64(x)/float64(c.W)*0.15))
			d := math.Hypot(float64(x)-gx, float64(y)-gy) / R
			if d < 1 {
				g := (1 - d) * (1 - d) * 0.30
				base = mix(base, glow, g)
			}
			i := (y*c.W + x) * 4
			c.bg.Pix[i], c.bg.Pix[i+1], c.bg.Pix[i+2], c.bg.Pix[i+3] = base.R, base.G, base.B, 255
		}
	}
}

func (c *Canvas) reset() { copy(c.img.Pix, c.bg.Pix) }

func (c *Canvas) px(x, y int, col RGB, a float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || a <= 0 {
		return
	}
	i := (y*c.W + x) * 4
	p := c.img.Pix
	if a >= 1 {
		p[i], p[i+1], p[i+2] = col.R, col.G, col.B
		return
	}
	p[i] = lerp(p[i], col.R, a)
	p[i+1] = lerp(p[i+1], col.G, a)
	p[i+2] = lerp(p[i+2], col.B, a)
}

func (c *Canvas) Rect(x, y, w, h int, col RGB, a float64) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c.px(i, j, col, a)
		}
	}
}

// RRect draws an anti-aliased rounded rectangle.
func (c *Canvas) RRect(x, y, w, h, r int, col RGB, a float64) {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	fr := float64(r)
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			cov := 1.0
			var cx, cy float64
			in := false
			switch {
			case i < x+r && j < y+r:
				cx, cy, in = float64(x+r), float64(y+r), true
			case i >= x+w-r && j < y+r:
				cx, cy, in = float64(x+w-r), float64(y+r), true
			case i < x+r && j >= y+h-r:
				cx, cy, in = float64(x+r), float64(y+h-r), true
			case i >= x+w-r && j >= y+h-r:
				cx, cy, in = float64(x+w-r), float64(y+h-r), true
			}
			if in {
				cov = clamp01(fr - math.Hypot(float64(i)+0.5-cx, float64(j)+0.5-cy) + 0.5)
			}
			c.px(i, j, col, a*cov)
		}
	}
}

// RRectBorder draws a filled rounded rect with a border of width bw.
func (c *Canvas) RRectBorder(x, y, w, h, r, bw int, fill, border RGB) {
	c.RRect(x, y, w, h, r, border, 1)
	c.RRect(x+bw, y+bw, w-2*bw, h-2*bw, r-bw, fill, 1)
}

// Shadow draws a soft drop shadow by stacking translucent rounded rects.
func (c *Canvas) Shadow(x, y, w, h, r, spread int, col RGB, strength float64) {
	for i := spread; i > 0; i-- {
		a := strength * (1 - float64(i)/float64(spread+1)) * 0.022
		c.RRect(x-i, y-i+spread/2, w+2*i, h+2*i, r+i, col, a)
	}
}

func (c *Canvas) Circle(cx, cy, r float64, col RGB, a float64) {
	for j := int(cy - r - 1); j <= int(cy+r+1); j++ {
		for i := int(cx - r - 1); i <= int(cx+r+1); i++ {
			cov := clamp01(r - math.Hypot(float64(i)+0.5-cx, float64(j)+0.5-cy) + 0.5)
			c.px(i, j, col, a*cov)
		}
	}
}

func (c *Canvas) Text(f *Font, x, y int, s string, scale int, col RGB) int {
	for _, r := range s {
		g := f.glyph(r)
		for row := 0; row < f.H; row++ {
			for cl := 0; cl < f.W; cl++ {
				if g[row*f.rowB+cl/8]&(0x80>>(cl%8)) != 0 {
					c.Rect(x+cl*scale, y+row*scale, scale, scale, col, 1)
				}
			}
		}
		x += f.W * scale
	}
	return x
}

func (f *Font) width(s string, scale int) int {
	n := 0
	for range s {
		n++
	}
	return n * f.W * scale
}

func (c *Canvas) TextC(f *Font, cx, y int, s string, scale int, col RGB) {
	c.Text(f, cx-f.width(s, scale)/2, y, s, scale, col)
}

// fit truncates s with "..." so it is at most maxW pixels wide.
func fit(f *Font, s string, scale, maxW int) string {
	if f.width(s, scale) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && f.width(string(r)+"...", scale) > maxW {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

// Image draws img scaled (box filter) to fit a size x size square centred on cx, cy.
func (c *Canvas) Image(img image.Image, cx, cy, size int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return
	}
	sc := float64(size) / float64(max(w, h))
	dw, dh := int(float64(w)*sc+0.5), int(float64(h)*sc+0.5)
	x0, y0 := cx-dw/2, cy-dh/2
	for y := 0; y < dh; y++ {
		sy0 := int(float64(y) / sc)
		sy1 := max(int(float64(y+1)/sc), sy0+1)
		for x := 0; x < dw; x++ {
			sx0 := int(float64(x) / sc)
			sx1 := max(int(float64(x+1)/sc), sx0+1)
			var r, g, bl, a, n uint64
			for sy := sy0; sy < sy1 && sy < h; sy++ {
				for sx := sx0; sx < sx1 && sx < w; sx++ {
					pr, pg, pb, pa := img.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r, g, bl, a, n = r+uint64(pr), g+uint64(pg), bl+uint64(pb), a+uint64(pa), n+1
				}
			}
			if n == 0 || a == 0 {
				continue
			}
			col := RGB{uint8(r * 255 / a), uint8(g * 255 / a), uint8(bl * 255 / a)}
			c.px(x0+x, y0+y, col, float64(a)/float64(n)/65535)
		}
	}
}
