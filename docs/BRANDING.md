# SelectOS branding

The mark represents three operating-system cards with the middle one selected: *choose your OS*.

## Source assets

| File | Use |
| --- | --- |
| `assets/branding/svg/selectos-logo-dark-bg.svg` / `selectos-logo-light-bg.svg` | Full wordmarks; text is converted to vector paths. |
| `assets/branding/svg/selectos-mark.svg` | Square mark for icons, avatars, and favicons. |
| `assets/branding/svg/brand-sheet.svg` | Side-by-side brand reference sheet. |
| `assets/branding/svg/social-preview.svg` | Vector source for the repository social-preview artwork. |
| `assets/branding/make_logo.py` | Regenerates SVG and PNG variants. |

The repository currently tracks the SVG sources and generator. PNG exports are generated locally and are not part of this change.

## Regenerate the assets

Install the Python dependencies:

```sh
python3 -m pip install cairosvg fonttools
```

The generator requires **Poppins Bold** and **Poppins Medium** TrueType font files. Supply paths to font files that you are licensed to use; the font files are not bundled:

```sh
python3 assets/branding/make_logo.py \
  --font-bold /path/to/Poppins-Bold.ttf \
  --font-medium /path/to/Poppins-Medium.ttf
```

Alternatively, set `SELECTOS_FONT_BOLD` and `SELECTOS_FONT_MEDIUM`. Use `--output-dir DIR` to write the generated `svg/` and `png/` directories elsewhere, and add `--alt` to generate the optional check-mark variant under `alt-check/`.

The wordmarks already contain outlined text, so viewing or using the committed SVG files does not require installing Poppins. The font files are needed only when regenerating the artwork.

## Colours and usage

| Token | Value |
| --- | --- |
| Blue (light) | `#5b9bff` |
| Blue (dark) | `#2f5fe0` |
| Navy text | `#0b1430` |

Minimum size: mark 32 px, full logo 120 px wide. Keep clear space of at least one quarter of the mark's height around it. Do not stretch, recolour the cards, or add effects.

## GitHub social preview

To use a raster export as the repository social preview, open **Settings → General → Social preview** and upload the generated `png/social-preview.png`. GitHub repository settings are not changed by committing these source files.
