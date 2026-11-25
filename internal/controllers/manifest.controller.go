package controllers

import (
	"journal/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&ManifestController{})
}

// ManifestController serves the web app manifest with a theme color that
// matches the palette in the query string (?palette=lavender). The palette is
// not secret, and carrying it in the URL means the install and update checks
// don't depend on cookies. Unknown values fall back to the default mint.
type ManifestController struct {
	app *fiber.App
}

func (c *ManifestController) Init(db *gorm.DB, app *fiber.App) {
	c.app = app
}

func (c *ManifestController) RegisterViewRoutes() {
	c.app.Get("/manifest.webmanifest", serveManifest)
}

func (c *ManifestController) RegisterApiRoutes() {}

func serveManifest(ctx *fiber.Ctx) error {
	ctx.Set(fiber.HeaderContentType, "application/manifest+json; charset=utf-8")
	ctx.Set(fiber.HeaderCacheControl, "no-cache")
	return ctx.JSON(fiber.Map{
		"name":             "Contour Journal",
		"short_name":       "Contour",
		"description":      "A journal that helps you notice patterns",
		"id":               "/dashboard",
		"start_url":        "/dashboard",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#fafafa",
		"theme_color":      models.PaletteThemeColor(ctx.Query("palette")),
		"icons": []fiber.Map{
			{"src": "/assets/images/icon-192.png", "sizes": "192x192", "type": "image/png"},
			{"src": "/assets/images/icon-512.png", "sizes": "512x512", "type": "image/png"},
		},
	})
}
