package controllers

import (
	"fmt"
	"io"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/testutil"
	"journal/internal/utils"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// The rating calendar only draws a dot per day, so the entry text has no reason
// to be in the page.
func TestRatingCalendarSendsRatingWithoutEntryText(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	testutil.Tx(t, func(tx *gorm.DB) {
		middleware.SetDB(tx)
		user := testutil.User(t, tx, "calendar@example.com")
		rating := &models.Rating{BaseType: &models.BaseType{Base: baseFor(user), ID: 902, Code: "calendar-test", Name: "calendar-test"}, Value: 4}
		if err := tx.Create(rating).Error; err != nil {
			t.Fatal(err)
		}
		when := time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC)
		j := &models.Journal{Base: baseFor(user), Date: &when, Entry: "entry " + secretA, JournalTypeID: 1, RatingID: rating.ID}
		if err := tx.Create(j).Error; err != nil {
			t.Fatal(err)
		}

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(utils.LocalCurrentUser, user)
			return c.Next()
		})
		c := &DashboardController{}
		c.Init(tx, app)
		c.RegisterViewRoutes()

		status, body := serve(t, app, "/dashboard/calendar?month=3&year=2026")

		if status != http.StatusOK {
			t.Fatalf("status %d: %s", status, body)
		}
		_, data, found := strings.Cut(body, `id="calendar-journals"`)
		if !found {
			t.Fatal("no calendar-journals data in the page")
		}
		data, _, _ = strings.Cut(data, "</script>")
		for _, want := range []string{`"date":"2026-03-10T12:00:00Z"`, fmt.Sprintf(`"ratingId":%d`, rating.ID), `"value":4`} {
			if !strings.Contains(data, want) {
				t.Errorf("data is missing %s: %s", want, data)
			}
		}
		if strings.Contains(body, secretA) {
			t.Errorf("entry text reached the page: %s", data)
		}
	})
}

// The dashboard page draws the current month's calendar inline, from its own
// route, so it needs the same guarantee.
func TestDashboardPageSendsNoEntryText(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	testutil.Tx(t, func(tx *gorm.DB) {
		middleware.SetDB(tx)
		user := testutil.User(t, tx, "dashboard@example.com")
		id := journalFor(t, tx, user)

		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals(utils.LocalCurrentUser, user)
			return c.Next()
		})
		c := &DashboardController{}
		c.Init(tx, app)
		c.RegisterViewRoutes()

		status, body := serve(t, app, "/dashboard?tab=patterns")
		if status != http.StatusOK {
			t.Fatalf("status %d: %s", status, body)
		}
		_, data, found := strings.Cut(body, `id="calendar-journals"`)
		data, _, _ = strings.Cut(data, "</script>")
		if !found || !strings.Contains(data, `"ratingId":1`) {
			t.Fatalf("the entry's day (journal %s) isn't in the calendar data: %s", id, data)
		}
		if strings.Contains(body, secretA) {
			t.Errorf("entry text reached the page: %s", data)
		}
	})
}

// serve GETs path over a real listener: templ refuses to render under
// app.Test, whose request context reads as canceled.
func serve(t *testing.T, app *fiber.App, path string) (int, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go app.Listener(ln)
	t.Cleanup(func() { app.Shutdown() })

	res, err := http.Get("http://" + ln.Addr().String() + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}
