package controllers

import (
	"errors"
	"go-starter/cmd/web/actionItems"
	"go-starter/internal/logger"
	"go-starter/internal/middleware"
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	registerController(&ActionItemController{})
}

type ActionItemController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *ActionItemController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("action-items")
	c.api = app.Group("api/action-items")
}

func (c *ActionItemController) getActionItem(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	id := ctx.Params("id")
	var actionItem models.ActionItem
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		Preload("Journal").
		First(&actionItem).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "ActionItem not found"})
	}
	ctx.Locals("actionItem", &actionItem)

	return ctx.Next()
}

func (c *ActionItemController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/:id/form", c.getActionItem, c.getActionItemForm)
}

func (c *ActionItemController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createActionItem)
	c.api.Put("/:id", c.getActionItem, c.updateActionItem)
	c.api.Delete("/:id", c.getActionItem, c.deleteActionItem)
}

type ActionItemBody struct {
	Text        string `form:"text"`
	CompletedAt int    `form:"completedAt"`
	IsComplete  bool   `form:"isComplete"`
	JournalID   string `form:"journalId"`
}

func (c *ActionItemController) parseActionItemFromBody(ctx *fiber.Ctx, actionItem *models.ActionItem) error {
	var body ActionItemBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return err
	}

	if body.Text == "" {
		return errors.New("text is required")
	}
	actionItem.Text = body.Text

	if body.IsComplete {
		now := time.Now()
		actionItem.CompletedAt = &now
	} else if actionItem.CompletedAt != nil {
		actionItem.CompletedAt = nil
	}

	// if body.CompletedAt != 0 {
	// 	date := time.UnixMilli(int64(body.CompletedAt))
	// 	actionItem.CompletedAt = &date
	// } else if actionItem.CompletedAt != nil {
	// 	actionItem.CompletedAt = nil
	// }

	if body.JournalID != "" {
		journalUuid, err := uuid.Parse(body.JournalID)
		if err != nil {
			return err
		}
		actionItem.JournalID = journalUuid
	}

	return nil
}

func (c *ActionItemController) getActionItemForm(ctx *fiber.Ctx) error {
	actionItem := utils.GetLocal[models.ActionItem](ctx, "actionItem")
	component := actionItems.Form(ctx, actionItem.Journal, actionItem)
	return utils.RenderComponent(component, ctx)
}

func (c *ActionItemController) createActionItem(ctx *fiber.Ctx) error {
	var actionItem models.ActionItem
	err := c.parseActionItemFromBody(ctx, &actionItem)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	actionItem.Base = &models.Base{CreatorID: user.ID, LastUpdaterID: user.ID}

	err = c.db.Create(&actionItem).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action item"})
	}

	components := []templ.Component{
		actionItems.ListItem(ctx, &actionItem),
		actionItems.Form(ctx, actionItem.Journal, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ActionItemController) updateActionItem(ctx *fiber.Ctx) error {
	actionItem := utils.GetLocal[models.ActionItem](ctx, "actionItem")

	err := c.parseActionItemFromBody(ctx, actionItem)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	actionItem.Base.LastUpdaterID = user.ID

	err = c.db.Save(actionItem).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating journal"})
	}

	components := []templ.Component{
		actionItems.ListItem(ctx, actionItem),
		actionItems.Form(ctx, actionItem.Journal, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ActionItemController) deleteActionItem(ctx *fiber.Ctx) error {
	actionItem := utils.GetLocal[models.ActionItem](ctx, "actionItem")

	err := c.db.Delete(actionItem).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting action item"})
	}

	return ctx.Status(http.StatusOK).JSON(actionItem)
}
