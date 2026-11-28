//go:build linux

package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var (
	cBGTop    = RGB{8, 12, 26}
	cBGBot    = RGB{16, 27, 58}
	cGlow     = RGB{59, 130, 246}
	cCard     = RGB{19, 28, 49}
	cCardSel  = RGB{26, 39, 70}
	cBorder   = RGB{40, 54, 88}
	cInput    = RGB{11, 18, 34}
	cText     = RGB{232, 238, 255}
	cMuted    = RGB{139, 151, 184}
	cDim      = RGB{92, 104, 136}
	cAccent   = RGB{79, 140, 255}
	cAccentDk = RGB{52, 86, 160}
	cGood     = RGB{52, 211, 153}
	cBad      = RGB{248, 113, 113}
	cWin      = RGB{0, 164, 239}
	cUbuntu   = RGB{233, 84, 32}
	cWhite    = RGB{255, 255, 255}
)

var allowedSuffix = "@gbox.ncf.edu.ph"

type osItem struct {
	ID    uint16
	Name  string
	Sub   string
	Kind  int // 0 generic, 1 windows, 2 ubuntu, 3 other linux
	Img   image.Image
	Order int
	Known bool
}

type UI struct {
	c          *Canvas
	fS, fM, fL *Font
	s          float64
	ts         int
	hw         string
	secure     bool
	brand1     string
	logo       image.Image
	brand2     string
}

func newUI(w, h int, cfg *Config) *UI {
	u := &UI{c: newCanvas(w, h), brand1: cfg.Brand.Name, brand2: cfg.Brand.Subtitle}
	u.s = math.Min(1.6, math.Max(0.6, float64(h)/1080))
	u.ts = 1
	if h >= 1000 {
		u.ts = 2
	}
	u.fS = loadFont("Uni3-Terminus16")
	u.fM = loadFont("Uni3-TerminusBold24x12")
	u.fL = loadFont("Uni3-TerminusBold32x16")
	u.c.makeBackground(cBGTop, cBGBot, cGlow)
	u.logo = loadIcon(espPath, cfg.Brand.Logo)
	u.hw = hardwareName()
	u.secure = secureBootOn()
	return u
}

func (u *UI) sc(v float64) int { return int(v*u.s + 0.5) }

func readTrim(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func hardwareName() string {
	v := readTrim("/sys/class/dmi/id/sys_vendor")
	p := readTrim("/sys/class/dmi/id/product_name")
	if p == "" {
		return "Lab PC"
	}
	if v != "" && !strings.HasPrefix(strings.ToLower(p), strings.ToLower(v)) {
		return v + " " + p
	}
	return p
}

func secureBootOn() bool {
	b, err := os.ReadFile(filepath.Join(efivarsDir, "SecureBoot-"+efiGlobalGUID))
	return err == nil && len(b) >= 5 && b[4] == 1
}

// ---- shared chrome -------------------------------------------------------

func (u *UI) brand() {
	c := u.c
	x, y := u.sc(48), u.sc(40)
	tx := x + u.sc(40)
	if u.logo != nil { // brand.logo from gate.json: the SelectOS mark
		sz := u.sc(46)
		c.Image(u.logo, x+sz/2, y+u.fM.H/2, sz)
		tx = x + sz + u.sc(14)
	} else {
		c.Circle(float64(x+u.sc(14)), float64(y+u.fM.H/2), float64(u.sc(14)), cAccent, 1)
		c.Circle(float64(x+u.sc(14)), float64(y+u.fM.H/2), float64(u.sc(6)), cBGTop, 1)
	}
	c.Text(u.fM, tx, y, u.brand1, 1, cText)
	c.Text(u.fM, tx+u.fM.width(u.brand1+"  ", 1), y, u.brand2, 1, cMuted)
}

func (u *UI) footer() {
	c := u.c
	y := u.c.H - u.sc(48)
	c.Text(u.fS, u.sc(48), y, u.hw, 1, cDim)
	right := "Gate v0.2"
	rx := c.W - u.sc(48) - u.fS.width(right, 1)
	c.Text(u.fS, rx, y, right, 1, cDim)
	sb := "Secure Boot: OFF"
	col := cBad
	if u.secure {
		sb, col = "Secure Boot: ON", cGood
	}
	sx := rx - u.sc(28) - u.fS.width(sb, 1)
	c.Circle(float64(sx-u.sc(12)), float64(y+u.fS.H/2), float64(u.sc(4)), col, 1)
	c.Text(u.fS, sx, y, sb, 1, cMuted)
}

func (u *UI) privacy() {
	u.c.TextC(u.fS, u.c.W/2, u.c.H-u.sc(84), "Username is a local session label; this screen does not authenticate accounts.", 1, cDim)
}

func (u *UI) keycap(x, y int, label string) int {
	f := u.fS
	w := f.width(label, 1) + u.sc(20)
	h := f.H + u.sc(12)
	u.c.RRectBorder(x, y, w, h, u.sc(7), 2, cInput, cBorder)
	u.c.Text(f, x+u.sc(10), y+u.sc(6), label, 1, cText)
	return x + w
}

// hints draws a centered row like: [Enter] start   [Esc] sign out
func (u *UI) hints(pairs [][2]string) {
	f := u.fS
	total := 0
	for _, p := range pairs {
		total += f.width(p[0], 1) + u.sc(20) + u.sc(10) + f.width(p[1], 1) + u.sc(28)
	}
	x := (u.c.W - total + u.sc(28)) / 2
	y := u.c.H - u.sc(124)
	for _, p := range pairs {
		x = u.keycap(x, y, p[0]) + u.sc(10)
		u.c.Text(f, x, y+u.sc(6), p[1], 1, cMuted)
		x += f.width(p[1], 1) + u.sc(28)
	}
}

// ---- icons -----------------------------------------------------------------

func (u *UI) icon(it osItem, cx, cy, size int) {
	c := u.c
	h := size / 2
	if it.Img != nil {
		c.Image(it.Img, cx, cy, size)
		return
	}
	switch it.Kind {
	case 1: // Windows
		g := u.sc(5)
		s := (size - g) / 2
		x0, y0 := cx-h, cy-h
		for i := 0; i < 2; i++ {
			for j := 0; j < 2; j++ {
				c.RRect(x0+i*(s+g), y0+j*(s+g), s, s, u.sc(4), cWin, 1)
			}
		}
	case 2: // Ubuntu
		r := float64(h)
		c.Circle(float64(cx), float64(cy), r, cUbuntu, 1)
		c.Circle(float64(cx), float64(cy), r*0.50, cWhite, 1)
		c.Circle(float64(cx), float64(cy), r*0.36, cUbuntu, 1)
		for _, a := range []float64{210, 330, 90} {
			rad := a * math.Pi / 180
			px, py := float64(cx)+math.Cos(rad)*r*0.68, float64(cy)-math.Sin(rad)*r*0.68
			c.Circle(px, py, r*0.20, cUbuntu, 1)
			c.Circle(px, py, r*0.13, cWhite, 1)
		}
	case 3: // other Linux: terminal
		c.RRectBorder(cx-h, cy-size*36/100, size, size*72/100, u.sc(12), 3, cInput, cAccent)
		c.Text(u.fL, cx-u.fL.width(">_", 1)/2, cy-u.fL.H/2, ">_", 1, cAccent)
	default: // generic disk
		c.RRectBorder(cx-h, cy-size*30/100, size, size*60/100, u.sc(14), 3, cInput, cMuted)
		c.Circle(float64(cx+h-u.sc(22)), float64(cy+size*30/100-u.sc(18)), float64(u.sc(5)), cGood, 1)
		c.Rect(cx-h+u.sc(20), cy-u.sc(4), size-u.sc(40), u.sc(8), cBorder, 1)
	}
}

// ---- screens ---------------------------------------------------------------

func (u *UI) drawLogin(local, errMsg string, caret bool) {
	c := u.c
	c.reset()
	u.brand()
	const subtitle = "Enter your Gbox username to continue."
	pad := u.sc(52)
	cw, ch := u.sc(660), u.sc(470)
	if need := u.fM.width(subtitle, 1) + 2*pad + u.sc(16); cw < need {
		cw = need // fonts are fixed-size, so never let text overflow the card
	}
	if cw > c.W-u.sc(40) {
		cw = c.W - u.sc(40)
	}
	x0, y0 := (c.W-cw)/2, (c.H-ch)/2-u.sc(20)
	c.Shadow(x0, y0, cw, ch, u.sc(26), u.sc(44), RGB{2, 5, 14}, 1)
	c.RRectBorder(x0, y0, cw, ch, u.sc(26), 2, cCard, cBorder)

	y := y0 + pad
	c.Text(u.fL, x0+pad, y, "Welcome", u.ts, cText)
	y += u.fL.H*u.ts + u.sc(10)
	c.Text(u.fM, x0+pad, y, subtitle, 1, cMuted)
	y += u.fM.H + u.sc(34)
	c.Text(u.fS, x0+pad, y, "GBOX USERNAME", 1, cDim)
	y += u.fS.H + u.sc(10)

	iw, ih := cw-2*pad, u.sc(72)
	border := cBorder
	if errMsg != "" {
		border = cBad
	} else {
		border = cAccent
	}
	c.RRectBorder(x0+pad, y, iw, ih, u.sc(14), 2, cInput, border)
	tx, ty := x0+pad+u.sc(22), y+(ih-u.fM.H)/2
	nx := c.Text(u.fM, tx, ty, local, 1, cText)
	if caret {
		c.Rect(nx+1, ty-u.sc(2), u.sc(2), u.fM.H+u.sc(4), cAccent, 1)
	}
	c.Text(u.fM, nx+u.sc(4), ty, allowedSuffix, 1, cDim)
	y += ih + u.sc(12)
	if errMsg != "" {
		c.Text(u.fS, x0+pad, y, errMsg, 1, cBad)
	}

	bh := u.sc(64)
	by := y0 + ch - pad - bh
	bcol := cAccent
	if len(local) < 3 {
		bcol = cAccentDk
	}
	c.RRect(x0+pad, by, iw, bh, u.sc(14), bcol, 1)
	c.TextC(u.fM, x0+cw/2, by+(bh-u.fM.H)/2, "Continue", 1, cWhite)

	u.privacy()
	u.hints([][2]string{{"Enter", "continue"}})
	u.footer()
}

func (u *UI) drawPicker(local string, items []osItem, sel int) {
	c := u.c
	c.reset()
	u.brand()

	// user chip (top right)
	email := fit(u.fS, local+allowedSuffix, 1, u.sc(360))
	ew := u.fS.width(email, 1)
	cx := c.W - u.sc(48) - ew - u.sc(44)
	cy := u.sc(40) + u.fM.H/2
	c.Circle(float64(cx+u.sc(14)), float64(cy), float64(u.sc(15)), cAccent, 1)
	in := strings.ToUpper(string([]rune(local)[:1]))
	c.TextC(u.fS, cx+u.sc(14), cy-u.fS.H/2, in, 1, cWhite)
	c.Text(u.fS, cx+u.sc(38), cy-u.fS.H/2, email, 1, cMuted)

	hy := int(float64(c.H) * 0.17)
	head := fit(u.fL, "Hello, "+local, u.ts, c.W-u.sc(160))
	c.TextC(u.fL, c.W/2, hy, head, u.ts, cText)
	c.TextC(u.fM, c.W/2, hy+u.fL.H*u.ts+u.sc(12), "Choose an operating system to start your session.", 1, cMuted)

	n := len(items)
	cw, chh, gap := max(u.sc(270), 240), u.sc(320), u.sc(30)
	if min := u.sc(198) + u.fL.H + u.fS.H*2 + u.sc(64); chh < min {
		chh = min
	}
	maxW := c.W - 2*u.sc(48)
	if n*cw+(n-1)*gap > maxW {
		cw = (maxW - (n-1)*gap) / n
	}
	total := n*cw + (n-1)*gap
	sx := (c.W - total) / 2
	sy := c.H/2 - chh/2 + u.sc(40)
	for i, it := range items {
		x := sx + i*(cw+gap)
		if i == sel {
			c.Shadow(x, sy, cw, chh, u.sc(22), u.sc(30), cAccent, 0.9)
			c.RRectBorder(x, sy, cw, chh, u.sc(22), 3, cCardSel, cAccent)
		} else {
			c.Shadow(x, sy, cw, chh, u.sc(22), u.sc(26), RGB{2, 5, 14}, 1)
			c.RRectBorder(x, sy, cw, chh, u.sc(22), 2, cCard, cBorder)
		}
		bx, by := x+u.sc(30), sy+u.sc(30)
		c.Circle(float64(bx), float64(by), float64(u.sc(15)), cInput, 1)
		c.TextC(u.fS, bx, by-u.fS.H/2, fmt.Sprint(i+1), 1, cMuted)
		u.icon(it, x+cw/2, sy+u.sc(120), u.sc(96))
		nf := u.fL
		if nf.width(it.Name, 1) > cw-u.sc(32) {
			nf = u.fM // long names: use the smaller font before truncating
		}
		c.TextC(nf, x+cw/2, sy+u.sc(198)+(u.fL.H-nf.H)/2, fit(nf, it.Name, 1, cw-u.sc(32)), 1, cText)
		c.TextC(u.fS, x+cw/2, sy+u.sc(198)+u.fL.H+u.sc(8), fit(u.fS, it.Sub, 1, cw-u.sc(32)), 1, cDim)
		if i == sel {
			c.TextC(u.fS, x+cw/2, sy+chh-u.sc(44), "Press Enter to start", 1, cAccent)
		}
	}
	u.hints([][2]string{{"<", "previous"}, {">", "next"}, {"Enter", "start"}, {"Esc", "back"}})
	u.footer()
}

func (u *UI) drawBoot(title, sub string, frame int, bad bool) {
	c := u.c
	c.reset()
	u.brand()
	col := cText
	if bad {
		col = cBad
	}
	y := c.H/2 - u.sc(120)
	c.TextC(u.fL, c.W/2, y, fit(u.fL, title, u.ts, c.W-u.sc(120)), u.ts, col)
	c.TextC(u.fM, c.W/2, y+u.fL.H*u.ts+u.sc(14), fit(u.fM, sub, 1, c.W-u.sc(120)), 1, cMuted)
	if !bad {
		cx, cy := float64(c.W/2), float64(y+u.fL.H*u.ts+u.sc(130))
		for i := 0; i < 8; i++ {
			a := float64(i) * math.Pi / 4
			age := ((frame-i)%8 + 8) % 8
			c.Circle(cx+math.Cos(a)*float64(u.sc(34)), cy+math.Sin(a)*float64(u.sc(34)), float64(u.sc(7)), cAccent, 0.2+0.8*float64(7-age)/7)
		}
	}
	u.footer()
}
