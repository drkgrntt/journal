package models

// UserPreferences is the typed shape of User.Metadata. Every preference lives
// in this one struct so that saving one can't drop the others: writing a
// narrower struct over the column replaces the whole document.
type UserPreferences struct {
	// HiddenPatterns holds the keys (dashboard.PatternKeys) of Patterns-tab
	// charts the user has opted out of.
	HiddenPatterns []string `json:"hiddenPatterns"`
	// AddAnotherEntry makes the new-entry form's "Add another entry" box start
	// checked. Unchecked is the default.
	AddAnotherEntry bool `json:"addAnotherEntry,omitempty"`
}

// Preferences reads the user's saved preferences; empty or unreadable
// metadata is just the defaults.
func (u *User) Preferences() UserPreferences {
	if u == nil || u.Base == nil {
		return UserPreferences{}
	}
	err, prefs := CastMetadata[UserPreferences](u.Metadata)
	if err != nil {
		return UserPreferences{}
	}
	return prefs
}

// SetPreferences writes prefs into u.Metadata. It does not save the user.
func (u *User) SetPreferences(prefs UserPreferences) error {
	metadata, err := EncodeMetadata(prefs)
	if err != nil {
		return err
	}
	u.Metadata = metadata
	return nil
}
