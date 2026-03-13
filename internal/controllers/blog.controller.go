package controllers

import (
	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"
)

func init() {
	registerController(&BlogController{})
}

type BlogController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *BlogController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("blog")
	c.api = app.Group("blog/api")
}

func (c *BlogController) RegisterApiRoutes() {
}

func (c *BlogController) RegisterViewRoutes() {
	// c.views.Get("/", utils.RenderPage(landings.LandingPage))
}
