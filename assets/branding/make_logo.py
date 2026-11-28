"""SelectOS logo generator. Run: python3 make_logo.py  (needs: pip install cairosvg fonttools)"""
import argparse
import os
import sys

import cairosvg
from fontTools.ttLib import TTFont
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen

FONT = None
FONT_MEDIUM = None
BLUE_LIGHT, BLUE_DARK = "#5b9bff", "#2f5fe0"

def text_path(text, size, x, baseline, tracking=0, font=None):
    """Convert text to an SVG path (so the logo needs no installed font)."""
    f = TTFont(font or FONT); gs = f.getGlyphSet(); cmap = f.getBestCmap(); upm = f["head"].unitsPerEm
    sc = size / upm; pen = SVGPathPen(gs); cx = x
    for ch in text:
        g = cmap[ord(ch)]
        gs[g].draw(TransformPen(pen, (sc, 0, 0, -sc, cx, baseline)))
        cx += gs[g].width * sc + tracking
    return pen.getCommands(), cx

def width_of(text, size, tracking=0):
    f = TTFont(FONT); gs = f.getGlyphSet(); cmap = f.getBestCmap(); sc = size / f["head"].unitsPerEm
    return sum(gs[cmap[ord(c)]].width * sc + tracking for c in text)

# ---- the mark: three OS "cards", the middle one selected -------------------------------
def mark_body(v="cursor"):
    base = f"""
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="{BLUE_LIGHT}"/><stop offset="1" stop-color="{BLUE_DARK}"/>
    </linearGradient>
    <radialGradient id="shine" cx="0.25" cy="0.15" r="0.8">
      <stop offset="0" stop-color="#fff" stop-opacity="0.28"/><stop offset="1" stop-color="#fff" stop-opacity="0"/>
    </radialGradient>
  </defs>
  <rect x="32" y="32" width="448" height="448" rx="112" fill="url(#bg)"/>
  <rect x="32" y="32" width="448" height="448" rx="112" fill="url(#shine)"/>
  <rect x="96"  y="192" width="80"  height="128" rx="24" fill="#fff" fill-opacity="0.38"/>
  <rect x="336" y="192" width="80"  height="128" rx="24" fill="#fff" fill-opacity="0.38"/>
  <rect x="200" y="176" width="112" height="176" rx="30" fill="#0b1c5c" fill-opacity="0.22"/>
  <rect x="200" y="166" width="112" height="176" rx="30" fill="#fff"/>
"""
    if v == "check":
        top = f'<path d="M226 258 L249 281 L288 232" fill="none" stroke="{BLUE_DARK}" stroke-width="20" stroke-linecap="round" stroke-linejoin="round"/>'
    else:  # cursor: a pointer arrow that is clicking the middle card
        top = (f'<g transform="translate(221 197) scale(2.45)">'
               f'<path d="M0 0 L0 40 L10.5 30 L18 47 L26 43.5 L18.5 27.5 L32 27.5 Z" fill="{BLUE_DARK}" '
               f'stroke="{BLUE_DARK}" stroke-width="2.4" stroke-linejoin="round"/></g>')
    return base + top

def mark_svg(size=512, v="cursor"):
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="32 32 448 448" width="{size}" height="{size}">{mark_body(v)}</svg>'

def lockup(dark=True, v="cursor"):
    t1, t2 = ("#f2f6ff", BLUE_LIGHT) if dark else ("#0b1430", BLUE_DARK)
    size, tr = 150, -2
    d1, x2 = text_path("Select", size, 270, 205, tr)
    d2, xe = text_path("OS", size, x2, 205, tr)
    w = int(xe + 20)
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} 300" width="{w}" height="300">'
            f'<svg x="10" y="40" width="220" height="220" viewBox="32 32 448 448">{mark_body(v)}</svg>'
            f'<path d="{d1}" fill="{t1}"/><path d="{d2}" fill="{t2}"/></svg>'), w

def build(v, root):
    os.makedirs(f"{root}/png", exist_ok=True); os.makedirs(f"{root}/svg", exist_ok=True)
    open(f"{root}/svg/selectos-mark.svg", "w").write(mark_svg(512, v))
    for px in (1024, 512, 256, 128, 64, 32):
        cairosvg.svg2png(bytestring=mark_svg(512, v).encode(), write_to=f"{root}/png/selectos-mark-{px}.png", output_width=px)
    for dark in (True, False):
        svg, w = lockup(dark, v); tag = "dark" if dark else "light"
        open(f"{root}/svg/selectos-logo-{tag}-bg.svg", "w").write(svg)
        cairosvg.svg2png(bytestring=svg.encode(), write_to=f"{root}/png/selectos-logo-{tag}-bg.png", output_width=w*2)

def nest(svg, w, x, y, width):
    inner = svg.split(">", 1)[1].rsplit("</svg>", 1)[0]
    return f'<svg x="{x}" y="{y}" width="{width}" height="{width*300/w}" viewBox="0 0 {w} 300">{inner}</svg>'
def tile(x, y, n, v):
    return f'<svg x="{x}" y="{y}" width="{n}" height="{n}" viewBox="32 32 448 448">{mark_body(v)}</svg>'
def sheet(v, root):
    d, wd = lockup(True, v); l, wl = lockup(False, v)
    t = lambda x, y, n: tile(x, y, n, v)
    out = f'''<svg xmlns="http://www.w3.org/2000/svg" width="1600" height="640" viewBox="0 0 1600 640">
<defs><linearGradient id="g" x1="0" y1="0" x2="0.4" y2="1"><stop offset="0" stop-color="#0b1426"/><stop offset="1" stop-color="#14224a"/></linearGradient></defs>
<rect width="800" height="640" fill="url(#g)"/><rect x="800" width="800" height="640" fill="#f5f7fc"/>
{nest(d, wd, 60, 110, 680)}{nest(l, wl, 860, 110, 680)}
{t(100,400,150)}{t(300,430,96)}{t(450,454,48)}{t(560,466,32)}
{t(900,400,150)}{t(1100,430,96)}{t(1250,454,48)}{t(1360,466,32)}
</svg>'''
    open(f"{root}/svg/brand-sheet.svg", "w").write(out)
    cairosvg.svg2png(bytestring=out.encode(), write_to=f"{root}/png/brand-sheet.png", output_width=1600)

def social(v, root):
    """1280x640 GitHub social-preview card (Settings > Social preview)."""
    d, w = lockup(True, v)
    medium = FONT_MEDIUM
    tag = "Lightweight UEFI multi-boot selector"
    tw = width_of(tag, 40)  # approximate; centred below
    p, _ = text_path(tag, 40, (1280 - 0.93 * tw) / 2, 520, 0, medium)
    out = f'''<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="640" viewBox="0 0 1280 640">
<defs><linearGradient id="g" x1="0" y1="0" x2="0.5" y2="1"><stop offset="0" stop-color="#0a1226"/><stop offset="1" stop-color="#162a63"/></linearGradient>
<radialGradient id="glow" cx="0.5" cy="0" r="0.8"><stop offset="0" stop-color="#4f8cff" stop-opacity="0.35"/><stop offset="1" stop-color="#4f8cff" stop-opacity="0"/></radialGradient></defs>
<rect width="1280" height="640" fill="url(#g)"/><rect width="1280" height="640" fill="url(#glow)"/>
{nest(d, w, 150, 150, 980)}
<path d="{p}" fill="#a9bdf0"/>
</svg>'''
    open(f"{root}/svg/social-preview.svg", "w").write(out)
    cairosvg.svg2png(bytestring=out.encode(), write_to=f"{root}/png/social-preview.png", output_width=1280)

def main():
    parser = argparse.ArgumentParser(description="Generate SelectOS logo SVG and PNG assets.")
    parser.add_argument("--font-bold", default=os.environ.get("SELECTOS_FONT_BOLD"),
                        help="Path to a licensed Poppins Bold TTF (or SELECTOS_FONT_BOLD).")
    parser.add_argument("--font-medium", default=os.environ.get("SELECTOS_FONT_MEDIUM"),
                        help="Path to a licensed Poppins Medium TTF (or SELECTOS_FONT_MEDIUM).")
    parser.add_argument("--output-dir", default=".", help="Output directory (default: current directory).")
    parser.add_argument("--alt", action="store_true", help="Also generate the check-mark variant.")
    args = parser.parse_args()

    for label, path in (("Poppins Bold", args.font_bold), ("Poppins Medium", args.font_medium)):
        if not path or not os.path.isfile(path):
            parser.error(f"{label} font file is required and must exist; pass the matching --font-* option.")
    global FONT, FONT_MEDIUM
    FONT, FONT_MEDIUM = args.font_bold, args.font_medium

    build("cursor", args.output_dir)
    sheet("cursor", args.output_dir)
    social("cursor", args.output_dir)
    if args.alt:
        alt_dir = os.path.join(args.output_dir, "alt-check")
        build("check", alt_dir)
        sheet("check", alt_dir)
    print(f"Brand assets written to {os.path.abspath(args.output_dir)}")


if __name__ == "__main__":
    main()
