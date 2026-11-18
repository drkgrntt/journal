package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func requestIDApp() *fiber.App {
	app := fiber.New()
	app.Use(RequestID)
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendString("fine") })
	app.Get("/text500", func(c *fiber.Ctx) error {
		return c.Status(http.StatusInternalServerError).SendString("Error creating thing")
	})
	app.Get("/json500", func(c *fiber.Ctx) error {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating thing"})
	})
	app.Get("/text404", func(c *fiber.Ctx) error {
		return c.Status(http.StatusNotFound).SendString("Not here")
	})
	app.Get("/empty500", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusInternalServerError) })
	app.Get("/err", func(c *fiber.Ctx) error { return errors.New("boom") })
	return app
}

func get(t *testing.T, app *fiber.App, path string, headers ...string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}$`)

func TestRequestIDHeader(t *testing.T) {
	app := requestIDApp()
	res, _ := get(t, app, "/ok")
	if id := res.Header.Get("X-Request-ID"); !idPattern.MatchString(id) {
		t.Fatalf("X-Request-ID = %q, want 8 hex chars", id)
	}
}

func TestRequestIDIsPerRequest(t *testing.T) {
	app := requestIDApp()
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		res, _ := get(t, app, "/ok")
		id := res.Header.Get("X-Request-ID")
		if seen[id] {
			t.Fatalf("request id %q repeated: ids must not link requests", id)
		}
		seen[id] = true
	}
}

// A client-supplied id could forge or inject into logs, so it is never trusted.
func TestRequestIDIgnoresClientValue(t *testing.T) {
	res, _ := get(t, requestIDApp(), "/ok", "X-Request-ID", "attacker\nforged")
	if id := res.Header.Get("X-Request-ID"); !idPattern.MatchString(id) {
		t.Fatalf("X-Request-ID = %q, want a server-generated id", id)
	}
}

func TestServerErrorBodiesGetTheReference(t *testing.T) {
	tests := []struct {
		name, path string
		want       func(body, id string) bool
	}{
		{"plain text 500 gets the reference appended", "/text500", func(b, id string) bool {
			return b == "Error creating thing (reference: "+id+")"
		}},
		{"empty 500 gets the standard text plus reference", "/empty500", func(b, id string) bool {
			return strings.HasSuffix(b, "(reference: "+id+")") && strings.HasPrefix(b, "Internal Server Error")
		}},
		{"JSON 500 is left as valid JSON (the header carries the id)", "/json500", func(b, id string) bool {
			return b == `{"message":"Error creating thing"}`
		}},
		{"4xx is not touched", "/text404", func(b, id string) bool { return b == "Not here" }},
		{"200 is not touched", "/ok", func(b, id string) bool { return b == "fine" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := get(t, requestIDApp(), tt.path)
			id := res.Header.Get("X-Request-ID")
			if !tt.want(body, id) {
				t.Fatalf("body = %q (id %q)", body, id)
			}
		})
	}
}

// An error returned from a handler becomes a 500 after this middleware runs, so
// the id must still be on the response.
func TestRequestIDOnReturnedError(t *testing.T) {
	res, _ := get(t, requestIDApp(), "/err")
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if id := res.Header.Get("X-Request-ID"); !idPattern.MatchString(id) {
		t.Fatalf("X-Request-ID = %q", id)
	}
}
