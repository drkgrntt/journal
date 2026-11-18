package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"journal/internal/models"
	"journal/internal/utils"

	"github.com/gofiber/fiber/v2"
)

func ago(d time.Duration) *time.Time   { t := time.Now().UTC().Add(-d); return &t }
func ahead(d time.Duration) *time.Time { t := time.Now().UTC().Add(d); return &t }

func userWith(status string, trialEnd, periodEnd *time.Time) *models.User {
	return &models.User{SubscriptionStatus: status, TrialEndsAt: trialEnd, CurrentPeriodEnd: periodEnd}
}

// run drives mw with an optional current user and returns the response.
func run(t *testing.T, mw fiber.Handler, user *models.User, header map[string]string) *http.Response {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if user != nil {
			c.Locals(utils.LocalCurrentUser, user)
		}
		return c.Next()
	})
	app.Get("/", mw, func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestRequireSubscription(t *testing.T) {
	const day = 24 * time.Hour
	tests := []struct {
		name string
		user *models.User
		want int
	}{
		{"no user", nil, 403},
		{"trial future", userWith("trialing", ahead(10*day), nil), 200},
		{"trial in grace", userWith("trialing", ago(3*day), nil), 200},
		{"trial past grace", userWith("trialing", ago(15*day), nil), 403},
		{"trial no end", userWith("trialing", nil, nil), 403},
		{"active future", userWith("active", nil, ahead(10*day)), 200},
		{"active no end", userWith("active", nil, nil), 200},
		{"past_due future", userWith("past_due", nil, ahead(day)), 200},
		{"past_due no end", userWith("past_due", nil, nil), 200},
		{"past_due past grace", userWith("past_due", nil, ago(20*day)), 403},
		{"canceled in grace", userWith("canceled", nil, ago(2*day)), 200},
		{"canceled past grace", userWith("canceled", nil, ago(15*day)), 403},
		{"canceled no end", userWith("canceled", nil, nil), 403},
		{"lifetime", userWith("lifetime", nil, nil), 200},
		{"comped", userWith("comped", ago(100*day), ago(100*day)), 200},
		{"empty status", userWith("", nil, nil), 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(t, RequireSubscription, tt.user, nil).StatusCode; got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	res := run(t, RequireAuth, nil, nil)
	if res.StatusCode != fiber.StatusFound || res.Header.Get("Location") != "/auth/login" {
		t.Errorf("no user: got %d -> %q, want 302 -> /auth/login", res.StatusCode, res.Header.Get("Location"))
	}
	// RequireAuth has no htmx special case: it still sends a plain redirect.
	res = run(t, RequireAuth, nil, map[string]string{"HX-Request": "true"})
	if res.StatusCode != fiber.StatusFound || res.Header.Get("HX-Redirect") != "" {
		t.Errorf("htmx: got %d HX-Redirect=%q", res.StatusCode, res.Header.Get("HX-Redirect"))
	}
	if got := run(t, RequireAuth, &models.User{}, nil).StatusCode; got != 200 {
		t.Errorf("with user: got %d, want 200", got)
	}
}

func TestFilterBots(t *testing.T) {
	app := fiber.New()
	app.Use(FilterBots)
	app.All("/*", func(c *fiber.Ctx) error { return c.SendString("ok") })
	tests := []struct {
		name, path, ua string
		want           int
	}{
		{"browser", "/", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36", 200},
		{"bot", "/", "Googlebot/2.1 (+http://www.google.com/bot.html)", 403},
		{"curl", "/", "curl/8.0", 403},
		{"bot on webhook", "/api/stripe/webhook", "curl/8.0", 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("User-Agent", tt.ua)
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.want)
			}
		})
	}
}
