package controllers

import (
	"journal/internal/logger"
	"journal/internal/utils"
	"journal/internal/web"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&ServiceWorkerController{})
}

// ServiceWorkerController serves the service worker from the site root. A
// worker's scope is capped at the directory it's served from, so it can't
// live under /assets/ like the rest of the embedded files and still cover
// /journal/new and /dashboard.
type ServiceWorkerController struct {
	app *fiber.App
}

func (c *ServiceWorkerController) Init(db *gorm.DB, app *fiber.App) {
	c.app = app
}

func (c *ServiceWorkerController) RegisterViewRoutes() {
	c.app.Get("/sw.js", serveServiceWorker)
}

func (c *ServiceWorkerController) RegisterApiRoutes() {}

func serveServiceWorker(ctx *fiber.Ctx) error {
	src, err := web.ServiceWorkerSource()
	if err != nil {
		logger.Error(err.Error())
		utils.SetRequestError(ctx, err)
		return ctx.SendStatus(http.StatusInternalServerError)
	}
	ctx.Set(fiber.HeaderContentType, "application/javascript; charset=utf-8")
	// Browsers re-check the worker script on every navigation regardless, but
	// make sure nothing in between holds a stale copy.
	ctx.Set(fiber.HeaderCacheControl, "no-cache")
	return ctx.Send(src)
}
