package controllers

import (
	"go-starter/cmd/web/dashboard"
	"go-starter/internal/middleware"
	"go-starter/internal/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&DashboardController{})
}

type DashboardController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *DashboardController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("dashboard")
	c.api = app.Group("api/dashboard")
}

func (c *DashboardController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)
	c.views.Get("/", utils.RenderPage(dashboard.DashboardPage))
}

func (c *DashboardController) RegisterApiRoutes() {
}
