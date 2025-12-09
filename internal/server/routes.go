package server

import (
	"go-starter/cmd/web"
	"go-starter/internal/controllers"
	"go-starter/internal/logger"
	"go-starter/internal/middleware"
	"net/http"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func (s *FiberServer) RegisterFiberRoutes() {
	s.App.Use("/assets", filesystem.New(filesystem.Config{
		Root:       http.FS(web.Files),
		PathPrefix: "assets",
		Browse:     false,
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

	s.App.Use(middleware.DeserializeToken)

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
