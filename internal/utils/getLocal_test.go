package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// runHandler runs fn inside a real request so it gets a live *fiber.Ctx.
func runHandler(t *testing.T, target string, fn func(*fiber.Ctx) error) {
	t.Helper()
	app := fiber.New()
	app.Get("/", fn)
	resp, err := app.Test(httptest.NewRequest("GET", target, nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestGetLocal(t *testing.T) {
	type thing struct{ N int }
	tests := []struct {
		name  string
		setup func(*fiber.Ctx)
		check func(*testing.T, *fiber.Ctx)
	}{
		{
			name:  "present",
			setup: func(c *fiber.Ctx) { c.Locals("k", &thing{N: 7}) },
			check: func(t *testing.T, c *fiber.Ctx) {
				if got := GetLocal[thing](c, "k"); got == nil || got.N != 7 {
					t.Errorf("got %v", got)
				}
			},
		},
		{
			name:  "missing key",
			setup: func(c *fiber.Ctx) {},
			check: func(t *testing.T, c *fiber.Ctx) {
				if got := GetLocal[thing](c, "k"); got != nil {
					t.Errorf("got %v", got)
				}
			},
		},
		{
			name:  "wrong type",
			setup: func(c *fiber.Ctx) { c.Locals("k", &thing{}) },
			check: func(t *testing.T, c *fiber.Ctx) {
				if got := GetLocal[string](c, "k"); got != nil {
					t.Errorf("got %v", got)
				}
			},
		},
		{
			name:  "non-pointer value is a type mismatch",
			setup: func(c *fiber.Ctx) { c.Locals("k", thing{N: 1}) },
			check: func(t *testing.T, c *fiber.Ctx) {
				if got := GetLocal[thing](c, "k"); got != nil {
					t.Errorf("got %v", got)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runHandler(t, "/", func(c *fiber.Ctx) error {
				tt.setup(c)
				tt.check(t, c)
				return nil
			})
		})
	}
}

func TestQueryArray(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   []string
	}{
		{"none", "/", nil},
		{"one", "/?a=1", []string{"1"}},
		{"many", "/?a=1&a=2&b=3&a=4", []string{"1", "2", "4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runHandler(t, tt.target, func(c *fiber.Ctx) error {
				got := QueryArray(c, "a")
				if len(got) != len(tt.want) {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("got %v, want %v", got, tt.want)
					}
				}
				return nil
			})
		})
	}
}

func TestGetDateInLocal(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
		want   string
	}{
		{"default UTC", "", "UTC"},
		{"valid zone", "America/New_York", "America/New_York"},
		{"invalid zone falls back", "Not/AZone", "UTC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				got := GetDateInLocal(c, nil)
				if got.Location().String() != tt.want {
					t.Errorf("location = %s, want %s", got.Location(), tt.want)
				}
				return nil
			})
			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookie != "" {
				req.Header.Set("Cookie", "tz="+tt.cookie)
			}
			if _, err := app.Test(req); err != nil {
				t.Fatal(err)
			}
		})
	}
}
