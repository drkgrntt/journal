package controllers

import (
	"encoding/json"
	"io"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// filterResult is what filterApp reports back: the entries' dates (UTC, in
// the order shown), the paging locals, and the prev/next day links.
type filterResult struct {
	Dates   []string
	HasMore bool
	Prev    string
	Next    string
}

// filterApp serves getJournals as the given user and reports its locals as
// JSON, so the list's filtering can be checked without rendering templates.
func filterApp(db *gorm.DB, u *models.User) *fiber.App {
	jc := &JournalController{db: db}
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		c.Locals(utils.LocalCurrentUser, u)
		return c.Next()
	}, jc.getJournals, func(c *fiber.Ctx) error {
		var r filterResult
		for _, j := range *utils.GetLocal[[]*models.Journal](c, utils.LocalJournals) {
			r.Dates = append(r.Dates, j.Date.UTC().Format("2006-01-02 15:04"))
		}
		if v := utils.GetLocal[bool](c, utils.LocalHasMore); v != nil {
			r.HasMore = *v
		}
		if v := utils.GetLocal[string](c, utils.LocalPrevDate); v != nil {
			r.Prev = *v
		}
		if v := utils.GetLocal[string](c, utils.LocalNextDate); v != nil {
			r.Next = *v
		}
		return c.JSON(r)
	})
	return app
}

func getFiltered(t *testing.T, app *fiber.App, query string) filterResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	req.AddCookie(&http.Cookie{Name: "tz", Value: "America/Chicago"})
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /?%s: %d %s", query, resp.StatusCode, b)
	}
	var r filterResult
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	return r
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestJournalListFilters(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	db, newUser := committedUsers(t)
	u, other := newUser(), newUser()

	// The lookup tables are empty in the test database.
	mkType := func(code string) models.JournalType {
		jt := models.JournalType{BaseType: &models.BaseType{Base: &models.Base{CreatorID: &uuid.Nil, LastUpdaterID: &uuid.Nil}, Code: code, Name: code}}
		if err := db.Where("code = ?", code).FirstOrCreate(&jt).Error; err != nil {
			t.Fatal(err)
		}
		return jt
	}
	custom, builtin := mkType("custom"), mkType("general")
	rating := models.Rating{BaseType: &models.BaseType{Base: &models.Base{CreatorID: &uuid.Nil, LastUpdaterID: &uuid.Nil}, Code: "ok", Name: "ok"}}
	if err := db.Where("code = ?", "ok").FirstOrCreate(&rating).Error; err != nil {
		t.Fatal(err)
	}
	cjt := &models.CustomJournalType{Base: &models.Base{CreatorID: &u.ID, LastUpdaterID: &u.ID}, Name: "Mine " + u.ID.String()[:6]}
	if err := db.Create(cjt).Error; err != nil {
		t.Fatal(err)
	}

	add := func(owner *models.User, at string, mutate func(*models.Journal)) {
		t.Helper()
		d, err := time.Parse("2006-01-02 15:04", at)
		if err != nil {
			t.Fatal(err)
		}
		j := &models.Journal{Base: &models.Base{CreatorID: &owner.ID, LastUpdaterID: &owner.ID}, Date: &d, Entry: "x", JournalTypeID: builtin.ID, RatingID: rating.ID}
		if mutate != nil {
			mutate(j)
		}
		if err := db.Create(j).Error; err != nil {
			t.Fatal(err)
		}
	}
	asCustom := func(j *models.Journal) { j.JournalTypeID = custom.ID; j.CustomJournalTypeID = cjt.ID }
	bookmarked := func(j *models.Journal) { n := time.Now(); j.BookmarkedAt = &n }

	// Chicago is UTC-6 in January: 2030-01-10 03:00Z is the evening of the 9th.
	add(u, "2030-01-10 03:00", nil)
	add(u, "2030-01-10 18:00", asCustom)
	add(u, "2030-01-12 15:00", bookmarked)
	add(u, "2030-01-14 15:00", asCustom)
	add(other, "2030-01-14 16:00", asCustom) // someone else's, never visible
	app := filterApp(db, u)

	t.Run("custom topic matches only its own entries", func(t *testing.T) {
		r := getFiltered(t, app, "topic="+cjt.SubID())
		if want := []string{"2030-01-14 15:00", "2030-01-10 18:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("dates = %v, want %v", r.Dates, want)
		}
	})

	t.Run("a topic nobody owns matches nothing", func(t *testing.T) {
		if r := getFiltered(t, app, "topic=000000000000"); len(r.Dates) != 0 {
			t.Errorf("dates = %v, want none", r.Dates)
		}
	})

	t.Run("by date shows one local day and its more button honours a custom topic", func(t *testing.T) {
		r := getFiltered(t, app, "sort=date&topic="+cjt.SubID())
		if want := []string{"2030-01-14 15:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("dates = %v, want %v", r.Dates, want)
		}
		if !r.HasMore {
			t.Error("hasMore = false, want true (an earlier custom-topic day exists)")
		}
		r = getFiltered(t, app, "sort=date&topic="+cjt.SubID()+"&page=1")
		if want := []string{"2030-01-10 18:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("page 1 dates = %v, want %v", r.Dates, want)
		}
		if r.HasMore {
			t.Error("hasMore = true on the last day")
		}
	})

	t.Run("by date groups by the user's local day", func(t *testing.T) {
		// 03:00Z on the 10th is the 9th in Chicago, so it is a day of its own.
		r := getFiltered(t, app, "sort=date&page=2")
		if want := []string{"2030-01-10 18:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("dates = %v, want %v", r.Dates, want)
		}
		r = getFiltered(t, app, "sort=date&page=3")
		if want := []string{"2030-01-10 03:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("dates = %v, want %v", r.Dates, want)
		}
	})

	t.Run("prev and next are the neighbouring local days", func(t *testing.T) {
		r := getFiltered(t, app, "date=2030-01-10")
		if r.Prev != "2030-01-09" || r.Next != "2030-01-12" {
			t.Errorf("prev/next = %q/%q, want 2030-01-09/2030-01-12", r.Prev, r.Next)
		}
	})

	t.Run("prev and next respect bookmarked-only", func(t *testing.T) {
		r := getFiltered(t, app, "sort=bookmark&date=2030-01-12")
		if want := []string{"2030-01-12 15:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("dates = %v, want %v", r.Dates, want)
		}
		if r.Prev != "" || r.Next != "" {
			t.Errorf("prev/next = %q/%q, want none", r.Prev, r.Next)
		}
	})

	t.Run("by date with a specific day shows that day, not an offset one", func(t *testing.T) {
		r := getFiltered(t, app, "sort=date&date=2030-01-12")
		if want := []string{"2030-01-12 15:00"}; !eqStrings(r.Dates, want) {
			t.Errorf("dates = %v, want %v", r.Dates, want)
		}
	})

	t.Run("a malformed date is a 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?date=nope", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", resp.StatusCode)
		}
	})
}
