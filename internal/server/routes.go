package server

import (
	"journal/internal/controllers"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/web"
	"net/http"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Use("/assets", filesystem.New(filesystem.Config{
		// Every request that reaches here carries the current build's
		// ?v=AssetVersion (base.templ's tags, and each CSS/JS file's own
		// @import/import specifiers - see web.newVersionedAssetFS), so a
		// year-long cache is safe: a changed file is a genuinely new URL,
		// never the same cached one.
		Root:       web.NewVersionedAssetFS(http.FS(web.Files)),
		PathPrefix: "assets",
		Browse:     false,
		MaxAge:     60 * 60 * 24 * 365,
	}))

	s.App.Use(func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Recovered panic:", "error", r)
				debug.PrintStack()
				c.Status(http.StatusInternalServerError).SendString("Internal Server Error")
			}
		}()
		return c.Next()
	})

	s.App.Use(
		middleware.FilterBots,
		middleware.DeserializeToken,
		middleware.LogAccess,
	)

	for _, controller := range controllers.GetControllers() {
		controller.Init(s.db.DB, s.App)
		controller.RegisterApiRoutes()
		controller.RegisterViewRoutes()
	}
	api := s.App.Group("/api")
	api.Get("/health", s.healthHandler)

	s.App.Use(func(ctx *fiber.Ctx) error {
		return ctx.Redirect("/dashboard")
	})
}

func (s *FiberServer) healthHandler(c *fiber.Ctx) error {
	return c.JSON(s.db.Health())
}
