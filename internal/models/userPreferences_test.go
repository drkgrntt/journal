package models

import (
	"reflect"
	"testing"
)

func prefsUser() *User { return &User{Base: &Base{}} }

func TestUserPreferencesDefaults(t *testing.T) {
	if got := prefsUser().Preferences(); got.AddAnotherEntry || len(got.HiddenPatterns) != 0 {
		t.Errorf("empty metadata should be defaults, got %+v", got)
	}
	if got := (&User{}).Preferences(); got.AddAnotherEntry {
		t.Errorf("user with no base should be defaults, got %+v", got)
	}
}

func TestUserPreferencesRoundTrip(t *testing.T) {
	u := prefsUser()
	want := UserPreferences{HiddenPatterns: []string{"calendar", "year"}, AddAnotherEntry: true}
	if err := u.SetPreferences(want); err != nil {
		t.Fatal(err)
	}
	if got := u.Preferences(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// Metadata written before AddAnotherEntry existed only has hiddenPatterns.
func TestUserPreferencesReadsLegacyMetadata(t *testing.T) {
	u := prefsUser()
	if err := u.Metadata.UnmarshalJSON([]byte(`{"hiddenPatterns":["calendar"]}`)); err != nil {
		t.Fatal(err)
	}
	got := u.Preferences()
	if !reflect.DeepEqual(got.HiddenPatterns, []string{"calendar"}) || got.AddAnotherEntry {
		t.Errorf("legacy metadata misread: %+v", got)
	}
}

// Changing one preference must keep the other.
func TestUserPreferencesChangeOneKeepsTheOther(t *testing.T) {
	u := prefsUser()
	if err := u.SetPreferences(UserPreferences{HiddenPatterns: []string{"calendar"}}); err != nil {
		t.Fatal(err)
	}
	prefs := u.Preferences()
	prefs.AddAnotherEntry = true
	if err := u.SetPreferences(prefs); err != nil {
		t.Fatal(err)
	}
	got := u.Preferences()
	if !got.AddAnotherEntry || !reflect.DeepEqual(got.HiddenPatterns, []string{"calendar"}) {
		t.Errorf("lost a preference: %+v", got)
	}
}
