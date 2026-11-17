package controllers

import (
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/entryTemplates"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&EntryTemplateController{})
}

type EntryTemplateController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *EntryTemplateController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("entry-templates")
	c.api = app.Group("api/entry-templates")
}

func (c *EntryTemplateController) loadTemplates(ctx *fiber.Ctx) ([]*models.EntryTemplate, error) {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var all []*models.EntryTemplate
	err := c.db.
		Where("creator_id = ?", currentUser.ID).
		Order("created_at asc").
		Find(&all).Error
	return all, err
}

func (c *EntryTemplateController) getEntryTemplate(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var entryTemplate models.EntryTemplate
	err := c.db.Where("id = ?", ctx.Params("id")).
		Where("creator_id = ?", currentUser.ID).
		First(&entryTemplate).Error
	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Template not found"})
	}
	ctx.Locals(utils.LocalEntryTemplate, &entryTemplate)

	return ctx.Next()
}

// getEntryTemplates stays open to lapsed users: they can always see (and
// delete) what they wrote.
func (c *EntryTemplateController) getEntryTemplates(ctx *fiber.Ctx) error {
	all, err := c.loadTemplates(ctx)
	if err != nil {
		logger.Error(err.Error())
	}
	ctx.Locals(utils.LocalEntryTemplates, &all)

	return ctx.Next()
}

// pickerList feeds the new-entry picker at runtime. It is never embedded in
// the (service-worker cached) new-entry page, and is never cached itself.
func (c *EntryTemplateController) pickerList(ctx *fiber.Ctx) error {
	all, err := c.loadTemplates(ctx)
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error loading templates"})
	}

	type pickerItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Body string `json:"body"`
	}
	items := make([]pickerItem, 0, len(all))
	for _, t := range all {
		items = append(items, pickerItem{ID: t.ID.String(), Name: t.Name, Body: t.Body})
	}

	ctx.Set("Cache-Control", "no-store")
	return ctx.JSON(items)
}

func (c *EntryTemplateController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/", c.getEntryTemplates, utils.RenderPage(entryTemplates.Page))
	c.views.Get("/:id/form", middleware.RequireSubscription, c.getEntryTemplate, c.getEntryTemplateForm)
}

func (c *EntryTemplateController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Get("/", middleware.RequireSubscription, c.pickerList)
	c.api.Post("/", middleware.RequireSubscription, c.createEntryTemplate)
	c.api.Put("/:id", middleware.RequireSubscription, c.getEntryTemplate, c.updateEntryTemplate)
	c.api.Delete("/:id", c.getEntryTemplate, c.deleteEntryTemplate)
}

type EntryTemplateBody struct {
	Name string `form:"name"`
	Body string `form:"body"`
}

func (c *EntryTemplateController) parseFromBody(ctx *fiber.Ctx, entryTemplate *models.EntryTemplate) error {
	var body EntryTemplateBody
	if err := ctx.BodyParser(&body); err != nil {
		return err
	}

	name, text, err := models.CleanEntryTemplateInput(body.Name, body.Body)
	if err != nil {
		return err
	}
	entryTemplate.Name = name
	entryTemplate.Body = text

	return nil
}

func (c *EntryTemplateController) getEntryTemplateForm(ctx *fiber.Ctx) error {
	entryTemplate := utils.GetLocal[models.EntryTemplate](ctx, utils.LocalEntryTemplate)
	return utils.RenderComponent(entryTemplates.Form(ctx, entryTemplate), ctx)
}

func (c *EntryTemplateController) createEntryTemplate(ctx *fiber.Ctx) error {
	var entryTemplate models.EntryTemplate
	if err := c.parseFromBody(ctx, &entryTemplate); err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	entryTemplate.Base = &models.Base{CreatorID: &user.ID, LastUpdaterID: &user.ID}

	if err := c.db.Create(&entryTemplate).Error; err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating template"})
	}

	components := []templ.Component{
		entryTemplates.ListItem(ctx, &entryTemplate, true),
		entryTemplates.Form(ctx, nil),
	}
	return utils.RenderComponents(components, ctx)
}

func (c *EntryTemplateController) updateEntryTemplate(ctx *fiber.Ctx) error {
	entryTemplate := utils.GetLocal[models.EntryTemplate](ctx, utils.LocalEntryTemplate)

	if err := c.parseFromBody(ctx, entryTemplate); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	entryTemplate.Base.LastUpdaterID = &user.ID

	if err := c.db.Save(entryTemplate).Error; err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating template"})
	}

	components := []templ.Component{
		entryTemplates.ListItem(ctx, entryTemplate, true),
		entryTemplates.Form(ctx, nil),
	}
	return utils.RenderComponents(components, ctx)
}

// deleteEntryTemplate stays open to lapsed users.
func (c *EntryTemplateController) deleteEntryTemplate(ctx *fiber.Ctx) error {
	entryTemplate := utils.GetLocal[models.EntryTemplate](ctx, utils.LocalEntryTemplate)

	if err := c.db.Delete(entryTemplate).Error; err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting template"})
	}

	return ctx.SendStatus(http.StatusOK)
}
