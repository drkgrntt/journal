package controllers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&WebhookController{})
}

type WebhookController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *WebhookController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	// c.views = app.Group("webhooks")
	c.api = app.Group("api/webhooks")

}

func (c *WebhookController) RegisterViewRoutes() {
}

func (c *WebhookController) RegisterApiRoutes() {
}
