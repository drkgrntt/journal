package controllers

import (
	"errors"
	thankfulViews "go-starter/cmd/web/thankful"
	"go-starter/internal/logger"
	"go-starter/internal/middleware"
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	registerController(&ThankfulController{})
}

type ThankfulController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *ThankfulController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("thankfuls")
	c.api = app.Group("api/thankfuls")
}

func (c *ThankfulController) getThankful(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	id := ctx.Params("id")
	var thankful models.Thankful
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		Preload("Journal").
		First(&thankful).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Thankful not found"})
	}
	ctx.Locals("thankful", &thankful)

	return ctx.Next()
}

func (c *ThankfulController) getThankfuls(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	var thankfuls []*models.Thankful
	tx := c.db.
		Preload("Journal.JournalType").
		Where("creator_id = ?", currentUser.ID).
		Order("created_at desc")

	journalTypeParam := ctx.Query("journalType")
	if journalTypeParam != "" {
		tx = tx.Where("journal_id IN (SELECT id FROM journals WHERE journal_type_id IN (SELECT id FROM journal_types WHERE code = ?))", journalTypeParam)
	}

	pageSize := 10
	page := ctx.QueryInt("page")
	tx = tx.Limit(pageSize + 1).Offset(page * pageSize)

	tx.Find(&thankfuls)
	ctx.Locals("thankfuls", &thankfuls)

	if len(thankfuls) > pageSize {
		thankfuls = thankfuls[:pageSize]
		hasMore := true
		nextPage := page + 1
		ctx.Locals("hasMore", &hasMore)
		ctx.Locals("nextPage", &nextPage)
	}

	return ctx.Next()
}

func (c *ThankfulController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	// c.views.Get("/", middleware.SetJournalTypes, c.getThankfuls, utils.RenderPage(thankfuls.ListPage))
	// c.views.Get("/list", c.getThankfuls, utils.RenderPage(thankfuls.ListItems))
	c.views.Get("/:id/form", c.getThankful, c.getThankfulForm)
}

func (c *ThankfulController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createThankful)
	c.api.Put("/:id", c.getThankful, c.updateThankful)
	c.api.Delete("/:id", c.getThankful, c.deleteThankful)
}

type ThankfulBody struct {
	Text      string `form:"text"`
	JournalID string `form:"journalId"`
}

func (c *ThankfulController) parseThankfulFromBody(ctx *fiber.Ctx, thankful *models.Thankful) error {
	var body ThankfulBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return err
	}

	if body.Text == "" {
		return errors.New("text is required")
	}
	thankful.Text = body.Text

	if body.JournalID != "" {
		journalUuid, err := uuid.Parse(body.JournalID)
		if err != nil {
			return err
		}
		thankful.JournalID = journalUuid
	}

	return nil
}

func (c *ThankfulController) getThankfulForm(ctx *fiber.Ctx) error {
	thankful := utils.GetLocal[models.Thankful](ctx, "thankful")
	component := thankfulViews.Form(ctx, thankful.Journal, thankful)
	return utils.RenderComponent(component, ctx)
}

func (c *ThankfulController) createThankful(ctx *fiber.Ctx) error {
	var thankful models.Thankful
	err := c.parseThankfulFromBody(ctx, &thankful)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	thankful.Base = &models.Base{CreatorID: user.ID, LastUpdaterID: user.ID}

	err = c.db.Create(&thankful).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating thankful"})
	}

	if thankful.HasJournal() {
		c.db.Where("id = ?", thankful.JournalID).First(&thankful.Journal)
	}

	components := []templ.Component{
		thankfulViews.ListItem(ctx, &thankful),
		thankfulViews.Form(ctx, thankful.Journal, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ThankfulController) updateThankful(ctx *fiber.Ctx) error {
	thankful := utils.GetLocal[models.Thankful](ctx, "thankful")

	err := c.parseThankfulFromBody(ctx, thankful)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	thankful.Base.LastUpdaterID = user.ID

	err = c.db.Save(thankful).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating journal"})
	}

	if thankful.HasJournal() {
		c.db.Where("id = ?", thankful.JournalID).First(&thankful.Journal)
	}

	components := []templ.Component{
		thankfulViews.ListItem(ctx, thankful),
		thankfulViews.Form(ctx, thankful.Journal, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ThankfulController) deleteThankful(ctx *fiber.Ctx) error {
	thankful := utils.GetLocal[models.Thankful](ctx, "thankful")

	err := c.db.Delete(thankful).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting thankful"})
	}

	return ctx.Status(http.StatusOK).JSON(thankful)
}
