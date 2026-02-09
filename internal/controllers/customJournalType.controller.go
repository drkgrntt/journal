package controllers

import (
	"errors"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	customJournalTypeViews "journal/internal/web/customJournalTypes"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&CustomJournalTypeController{})
}

type CustomJournalTypeController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *CustomJournalTypeController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("custom-journal-types")
	c.api = app.Group("api/custom-journal-types")
}

func (c *CustomJournalTypeController) getCustomJournalType(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	id := ctx.Params("id")
	var customJournalType models.CustomJournalType
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		First(&customJournalType).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Custom Journal Type not found"})
	}
	ctx.Locals("customJournalType", &customJournalType)

	return ctx.Next()
}

func (c *CustomJournalTypeController) getCustomJournalTypes(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	var customJournalTypes []*models.CustomJournalType
	tx := c.db.
		Where("creator_id = ?", currentUser.ID).
		Order("created_at desc")
	tx.Find(&customJournalTypes)
	ctx.Locals("customJournalTypes", &customJournalTypes)

	return ctx.Next()
}

func (c *CustomJournalTypeController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/:id/form", c.getCustomJournalType, c.getCustomJournalTypeForm)
}

func (c *CustomJournalTypeController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createCustomJournalType)
	c.api.Put("/:id", c.getCustomJournalType, c.updateCustomJournalType)
	c.api.Delete("/:id", c.getCustomJournalType, c.deleteCustomJournalType)
}

type CustomJournalTypeBody struct {
	Name string `form:"name"`
}

func (c *CustomJournalTypeController) parseCustomJournalTypeFromBody(ctx *fiber.Ctx, customJournalType *models.CustomJournalType) error {
	var body CustomJournalTypeBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return err
	}

	if body.Name == "" {
		return errors.New("name is required")
	}
	customJournalType.Name = body.Name

	return nil
}

func (c *CustomJournalTypeController) getCustomJournalTypeForm(ctx *fiber.Ctx) error {
	customJournalType := utils.GetLocal[models.CustomJournalType](ctx, "customJournalType")
	component := customJournalTypeViews.Form(ctx, customJournalType)
	return utils.RenderComponent(component, ctx)
}

func (c *CustomJournalTypeController) createCustomJournalType(ctx *fiber.Ctx) error {
	var customJournalType models.CustomJournalType
	err := c.parseCustomJournalTypeFromBody(ctx, &customJournalType)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	customJournalType.Base = &models.Base{CreatorID: &user.ID, LastUpdaterID: &user.ID}

	err = c.db.Create(&customJournalType).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating custom journal type"})
	}

	components := []templ.Component{
		customJournalTypeViews.ListItem(ctx, &customJournalType),
		customJournalTypeViews.Form(ctx, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *CustomJournalTypeController) updateCustomJournalType(ctx *fiber.Ctx) error {
	customJournalType := utils.GetLocal[models.CustomJournalType](ctx, "customJournalType")

	err := c.parseCustomJournalTypeFromBody(ctx, customJournalType)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	customJournalType.Base.LastUpdaterID = &user.ID

	err = c.db.Save(customJournalType).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating journal"})
	}

	components := []templ.Component{
		customJournalTypeViews.ListItem(ctx, customJournalType),
		customJournalTypeViews.Form(ctx, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *CustomJournalTypeController) deleteCustomJournalType(ctx *fiber.Ctx) error {
	customJournalType := utils.GetLocal[models.CustomJournalType](ctx, "customJournalType")

	err := c.db.Delete(customJournalType).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting custom journal type"})
	}

	return ctx.Status(http.StatusOK).JSON(customJournalType)
}
