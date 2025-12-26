package controllers

import (
	"go-starter/cmd/web/dashboard"
	"go-starter/internal/middleware"
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"time"

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

func (c *DashboardController) getJournals(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.
		Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", time.Now().AddDate(0, -1, 0)).
		Preload("Rating").
		Preload("JournalType").
		Order("created_at desc").
		Find(&journals)

	ctx.Locals("journals", &journals)

	return ctx.Next()
}

func (c *DashboardController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)
	c.views.Get("/", middleware.SetRatings, middleware.SetJournalTypes, c.getJournals, utils.RenderPage(dashboard.DashboardPage))
}

func (c *DashboardController) RegisterApiRoutes() {
}
