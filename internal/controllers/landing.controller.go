package controllers

import (
	"go-starter/cmd/web/landings"
	"go-starter/internal/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&LandingController{})
}

type LandingController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *LandingController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("/")
	c.api = app.Group("api")
}

func (c *LandingController) RegisterViewRoutes() {
	c.views.Get("/about", utils.RenderPage(landings.AboutPage))
	c.views.Get("/privacy", utils.RenderPage(landings.PrivacyPage))
}

func (c *LandingController) RegisterApiRoutes() {
}
