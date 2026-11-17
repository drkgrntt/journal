package models

import "testing"

func TestIsValidAppearance(t *testing.T) {
	tests := []struct {
		name      string
		theme     string
		font      string
		wantTheme bool
		wantFont  bool
	}{
		{"defaults", "", "", true, true},
		{"known", "lavender", "serif", true, true},
		{"unknown theme", "neon", "serif", false, true},
		{"unknown font", "sky", "comic", true, false},
		{"case sensitive", "Sky", "Serif", false, false},
		{"injection", `x" onload="y`, "serif;}", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTheme(tt.theme); got != tt.wantTheme {
				t.Errorf("IsValidTheme(%q) = %v", tt.theme, got)
			}
			if got := IsValidFontFamily(tt.font); got != tt.wantFont {
				t.Errorf("IsValidFontFamily(%q) = %v", tt.font, got)
			}
		})
	}
}

func TestResolveAppearance(t *testing.T) {
	tests := []struct {
		name       string
		subscribed bool
		theme      string
		font       string
		wantTheme  string
		wantFont   string
	}{
		{"subscribed keeps choice", true, "rose", "serif", "rose", "serif"},
		{"lapsed gets defaults", false, "rose", "serif", "", ""},
		{"subscribed default", true, "", "", "", ""},
		{"invalid stored value falls back", true, "neon", "comic", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, font := resolveAppearance(tt.subscribed, tt.theme, tt.font)
			if theme != tt.wantTheme || font != tt.wantFont {
				t.Errorf("got (%q, %q), want (%q, %q)", theme, font, tt.wantTheme, tt.wantFont)
			}
		})
	}
}
