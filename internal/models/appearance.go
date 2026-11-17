package models

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
