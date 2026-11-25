package models

import (
	"fmt"
	"math"
)

// Appearance options for the paid "Themes" feature. The empty string is
// always the default (the original mint look / the sans-serif face), so an
// unset column and an explicit default are the same thing.
var (
	// ThemePalettes must match the [data-palette] blocks in
	// assets/css/theme/appearance.css.
	ThemePalettes = []string{"", "lavender", "sky", "rose", "sand"}
	// FontFamilies must match the [data-font] rules in appearance.css.
	FontFamilies = []string{"", "serif"}
)

func contains(options []string, v string) bool {
	for _, o := range options {
		if o == v {
			return true
		}
	}
	return false
}

func IsValidTheme(v string) bool      { return contains(ThemePalettes, v) }
func IsValidFontFamily(v string) bool { return contains(FontFamilies, v) }

// resolveAppearance returns what actually gets rendered: the saved choice for
// someone with an active plan, the default for everyone else. The saved choice
// is never deleted, so it comes back if the plan does.
func resolveAppearance(subscribed bool, theme, font string) (string, string) {
	if !subscribed {
		return "", ""
	}
	if !IsValidTheme(theme) {
		theme = ""
	}
	if !IsValidFontFamily(font) {
		font = ""
	}
	return theme, font
}

// EffectiveAppearance is the palette and font family to render for this user.
func (u *User) EffectiveAppearance() (theme, font string) {
	return resolveAppearance(u.IsSubscribed(), u.Theme, u.FontFamily)
}

// paletteHues must match --palette-hue in assets/css/theme/appearance.css.
var paletteHues = map[string]float64{"lavender": 255, "sky": 205, "rose": 345, "sand": 38}

// DefaultThemeColor is the original mint, used when no palette applies.
const DefaultThemeColor = "#7EEAD2"

// PaletteThemeColor is the installed app's theme color (the Android system
// bar) for a palette: the light-mode --primary-color, hsl(hue 85% 85%).
func PaletteThemeColor(palette string) string {
	hue, ok := paletteHues[palette]
	if !ok {
		return DefaultThemeColor
	}
	const s, l = 0.85, 0.85
	c := (1 - math.Abs(2*l-1)) * s
	h := hue / 60
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	var r, g, b float64
	switch {
	case h < 1:
		r, g, b = c, x, 0
	case h < 2:
		r, g, b = x, c, 0
	case h < 3:
		r, g, b = 0, c, x
	case h < 4:
		r, g, b = 0, x, c
	case h < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	to := func(v float64) int { return int(math.Round((v + m) * 255)) }
	return fmt.Sprintf("#%02X%02X%02X", to(r), to(g), to(b))
}
