package controllers

import (
	"journal/internal/models"
	"journal/internal/testutil"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func prefsOf(t *testing.T, tx *gorm.DB, u *models.User) models.UserPreferences {
	t.Helper()
	var fresh models.User
	if err := tx.First(&fresh, "id = ?", u.ID).Error; err != nil {
		t.Fatal(err)
	}
	return fresh.Preferences()
}

func TestUpdatePreferences(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		a := compedUser(t, tx, uuid.NewString()+"@example.com")
		b := compedUser(t, tx, uuid.NewString()+"@example.com")
		if err := a.SetPreferences(models.UserPreferences{HiddenPatterns: []string{"calendar"}}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Model(a).Update("metadata", a.Metadata).Error; err != nil {
			t.Fatal(err)
		}

		appA := newAuthedTestApp(t, tx, a, &ProfileController{})
		code, body := send(t, appA, http.MethodPut, "/api/profile/preferences", form("addAnotherEntry", "true"))
		if !is2xx(code) {
			t.Fatalf("PUT: %d %s", code, body)
		}
		got := prefsOf(t, tx, a)
		if !got.AddAnotherEntry {
			t.Error("preference not saved")
		}
		if len(got.HiddenPatterns) != 1 || got.HiddenPatterns[0] != "calendar" {
			t.Errorf("hidden patterns lost: %+v", got.HiddenPatterns)
		}
		if prefsOf(t, tx, b).AddAnotherEntry {
			t.Error("another user's preference changed")
		}

		// The form's script sends the box's state either way.
		code, body = send(t, appA, http.MethodPut, "/api/profile/preferences", form("addAnotherEntry", "false"))
		if !is2xx(code) {
			t.Fatalf("PUT unchecked: %d %s", code, body)
		}
		if prefsOf(t, tx, a).AddAnotherEntry {
			t.Error("preference not cleared")
		}
	})
}

func TestUpdatePreferencesNeedsLogin(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		code, _ := send(t, newTestApp(t, tx, &ProfileController{}), http.MethodPut, "/api/profile/preferences", form("addAnotherEntry", "true"))
		if code != http.StatusUnauthorized {
			t.Errorf("status %d, want 401", code)
		}
	})
}
