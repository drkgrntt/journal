package controllers

import (
	"journal/internal/emails"
	"journal/internal/utils"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"
)

func init() {
	registerController(&DevController{})
}

type DevController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *DevController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("dev")
	c.api = app.Group("api/dev")
}

func (c *DevController) verifyNotProd(ctx *fiber.Ctx) error {
	if os.Getenv("APP_ENV") == "production" {
		return ctx.SendStatus(http.StatusForbidden)
	}
	return ctx.Next()
}

func (c *DevController) RegisterViewRoutes() {
	c.views.Use(c.verifyNotProd)
	c.views.Get("/email", c.renderTestEmail)
}

func (c *DevController) RegisterApiRoutes() {
}

func (c *DevController) renderTestEmail(ctx *fiber.Ctx) error {
	template := ctx.Query("template")
	if template == "" {
		return ctx.SendStatus(http.StatusBadRequest)
	}

	var component templ.Component

	switch template {
	case emails.GENERAL:
		component = emails.General(&emails.GeneralVariables{
			Message: "This is a test email",
		})
	case emails.FORGOT_PASSWORD:
		component = emails.ForgotPassword(&emails.ForgotPasswordVariables{
			ResetToken: "token",
			RootUrl:    "",
		})
	default:
		return ctx.SendStatus(http.StatusBadRequest)
	}

	return utils.RenderPage(func(ctx *fiber.Ctx) templ.Component {
		return component
	})(ctx)
}
