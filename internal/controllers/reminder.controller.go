package controllers

import (
	"errors"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/reminders"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&ReminderController{})
}

type ReminderController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *ReminderController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("reminders")
	c.api = app.Group("api/reminders")
}

func (c *ReminderController) getReminder(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	id := ctx.Params("id")
	var reminder models.Reminder
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		First(&reminder).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Reminder not found"})
	}
	ctx.Locals(utils.LocalReminder, &reminder)

	return ctx.Next()
}

func (c *ReminderController) getReminders(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var allReminders []*models.Reminder
	c.db.
		Where("creator_id = ?", currentUser.ID).
		Order("created_at desc").
		Find(&allReminders)
	ctx.Locals(utils.LocalReminders, &allReminders)

	return ctx.Next()
}

// shuffleReminder swaps the displayed reminder for a different random one.
func (c *ReminderController) shuffleReminder(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var reminder models.Reminder
	err := c.db.
		Where("creator_id = ?", currentUser.ID).
		Where("id != ?", ctx.Query("current", "00000000-0000-0000-0000-000000000000")).
		Order("random()").
		First(&reminder).Error
	if err != nil {
		// Only one reminder (or none): leave the card as it is
		return ctx.SendStatus(http.StatusNoContent)
	}

	return utils.RenderComponent(reminders.Card(ctx, &reminder, true), ctx)
}

func (c *ReminderController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/", c.getReminders, utils.RenderPage(reminders.Page))
	c.views.Get("/shuffle", c.shuffleReminder)
	c.views.Get("/:id/form", c.getReminder, c.getReminderForm)
}

func (c *ReminderController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createReminder)
	c.api.Put("/:id", c.getReminder, c.updateReminder)
	c.api.Delete("/:id", c.getReminder, c.deleteReminder)
}

type ReminderBody struct {
	Text string `form:"text"`
}

func (c *ReminderController) parseReminderFromBody(ctx *fiber.Ctx, reminder *models.Reminder) error {
	var body ReminderBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return err
	}

	if body.Text == "" {
		return errors.New("text is required")
	}
	reminder.Text = body.Text

	return nil
}

func (c *ReminderController) getReminderForm(ctx *fiber.Ctx) error {
	reminder := utils.GetLocal[models.Reminder](ctx, utils.LocalReminder)
	return utils.RenderComponent(reminders.Form(ctx, reminder), ctx)
}

func (c *ReminderController) createReminder(ctx *fiber.Ctx) error {
	var reminder models.Reminder
	err := c.parseReminderFromBody(ctx, &reminder)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	reminder.Base = &models.Base{CreatorID: &user.ID, LastUpdaterID: &user.ID}

	err = c.db.Create(&reminder).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating reminder"})
	}

	components := []templ.Component{
		reminders.ListItem(ctx, &reminder),
		reminders.Form(ctx, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ReminderController) updateReminder(ctx *fiber.Ctx) error {
	reminder := utils.GetLocal[models.Reminder](ctx, utils.LocalReminder)

	err := c.parseReminderFromBody(ctx, reminder)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	reminder.Base.LastUpdaterID = &user.ID

	err = c.db.Save(reminder).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating reminder"})
	}

	components := []templ.Component{
		reminders.ListItem(ctx, reminder),
		reminders.Form(ctx, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ReminderController) deleteReminder(ctx *fiber.Ctx) error {
	reminder := utils.GetLocal[models.Reminder](ctx, utils.LocalReminder)

	err := c.db.Delete(reminder).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting reminder"})
	}

	return ctx.Status(http.StatusOK).JSON(reminder)
}
