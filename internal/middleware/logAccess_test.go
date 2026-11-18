package middleware

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewAnalyticIsAnonymous(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	a := newAnalytic("GET", "journal.example", "/journal/new", 200, "a1b2c3d4", now)

	if a.Method != "GET" || a.Domain != "journal.example" || a.Page != "/journal/new" || a.Status != 200 || a.RequestID != "a1b2c3d4" {
		t.Fatalf("unexpected row: %+v", a)
	}
	if !a.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %v, want %v", a.CreatedAt, now)
	}
	if a.CreatorID == nil || *a.CreatorID != uuid.Nil || a.LastUpdaterID == nil || *a.LastUpdaterID != uuid.Nil {
		t.Fatalf("log rows must not carry an account ID: creator=%v updater=%v", a.CreatorID, a.LastUpdaterID)
	}
}

// The request log is anonymous by design (see the privacy policy). This fails
// if someone adds a column that could identify a person, so the change has to
// be a deliberate decision that also updates the policy.
func TestAnalyticHasNoIdentifyingFields(t *testing.T) {
	allowed := map[string]bool{"Base": true, "Method": true, "Domain": true, "Page": true, "Status": true, "RequestID": true}
	typ := reflect.TypeOf(newAnalytic("", "", "", 0, "", time.Time{})).Elem()
	for i := 0; i < typ.NumField(); i++ {
		if name := typ.Field(i).Name; !allowed[name] {
			t.Errorf("Analytic.%s is not in the allowed anonymous set; update the privacy policy if this is intended", name)
		}
	}
}
