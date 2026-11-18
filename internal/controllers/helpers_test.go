package controllers

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// newTestApp builds a Fiber app with one controller wired to db (usually a
// testutil.Tx transaction) and its API routes registered.
func newTestApp(t testing.TB, db *gorm.DB, c Controller) *fiber.App {
	t.Helper()
	app := fiber.New()
	c.Init(db, app)
	c.RegisterApiRoutes()
	return app
}
