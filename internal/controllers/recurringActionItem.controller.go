package controllers

import (
	"errors"
	"fmt"
	gormlogger "gorm.io/gorm/logger"
	"journal/internal/emails"
	"journal/internal/jobs"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/recurringActionItems"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"github.com/go-co-op/gocron/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	registerController(&RecurringActionItemController{})
}

type RecurringActionItemController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *RecurringActionItemController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("recurring-action-items")
	c.api = app.Group("api/recurring-action-items")

	c.registerCrons()
}

func (c *RecurringActionItemController) registerCrons() {
	s, err := gocron.NewScheduler()
	if err != nil {
		logger.Error("Error starting scheduler", "error", err.Error())
		return
	}

	// Create the next time
	_, err = s.NewJob(
		gocron.DurationJob(time.Minute),
		gocron.NewTask(c.createActionItems),
	)
	if err != nil {
		logger.Error("Error starting scheduler", "error", err.Error())
	}

	s.Start()
}

func (c *RecurringActionItemController) createActionItems() {
	now := time.Now()
	var recurringActionItems []*models.RecurringActionItem
	c.db.
		Session(&gorm.Session{
			Logger: gormlogger.Default.LogMode(gormlogger.Error),
			// Skip the decryption step as it can be slow and isn't needed
			SkipHooks: true,
		}).
		Preload(
			"ActionItems",
			func(tx *gorm.DB) *gorm.DB {
				return tx.Order("created_at DESC")
			},
		).
		Where("starts_at <= ?", now).
		Find(&recurringActionItems)

	subscribed := c.subscribedCreators(recurringActionItems)

	var actionItems []*models.ActionItem
	var emailCreators []uuid.UUID
	for _, recurringActionItem := range recurringActionItems {
		// A routine whose owner's plan has lapsed just waits; it picks up
		// again whenever the plan does.
		if !subscribed[*recurringActionItem.CreatorID] {
			continue
		}

		// Exit if it hasn't started yet
		if recurringActionItem.StartsAt.After(now) {
			// logger.Info("Recurring action item has not started yet", "id", recurringActionItem.ID)
			continue
		}

		if len(recurringActionItem.ActionItems) > 0 {
			recentActionItem := recurringActionItem.ActionItems[0]

			// Exit if it has not been completed yet
			if recentActionItem.CompletedAt == nil {
				// logger.Info("Recurring action item has not been completed yet", "id", recurringActionItem.ID)
				continue
			}

			startsAtUnix := recurringActionItem.StartsAt.Unix()
			nowUnix := now.Unix()
			frequency := int64(recurringActionItem.Frequency.Seconds())
			if frequency <= 0 {
				logger.Warn("Recurring action item has a non-positive frequency, skipping", "id", recurringActionItem.ID)
				continue
			}
			periodsSinceStart := (nowUnix - startsAtUnix) / frequency

			previousScheduled := recurringActionItem.StartsAt.Add(time.Duration(periodsSinceStart) * recurringActionItem.Frequency)

			// logger.Info("period info", "started", recurringActionItem.StartsAt, "periods", periodsSinceStart, "previous scheduled", previousScheduled, "created", recentActionItem.CreatedAt, "completed", recentActionItem.CompletedAt)

			// Exit if it has been created in this period
			if recentActionItem.CreatedAt.After(previousScheduled) {
				// logger.Info("Recurring action item has been created in this period", "id", recurringActionItem.ID)
				continue
			}

			// Exit if it has been completed in this period
			if recentActionItem.CompletedAt.After(previousScheduled) {
				// logger.Info("Recurring action item has been completed in this period", "id", recurringActionItem.ID)
				continue
			}
		}

		actionItem := models.ActionItem{
			Text: recurringActionItem.Text,
			// Since we're skipping the encryption step, we can just set this to true
			IsEncrypted:           true,
			RecurringActionItemID: &recurringActionItem.ID,
			Base: &models.Base{
				CreatorID:     recurringActionItem.CreatorID,
				LastUpdaterID: recurringActionItem.LastUpdaterID,
			},
		}
		actionItems = append(actionItems, &actionItem)
		if recurringActionItem.EmailMe {
			emailCreators = append(emailCreators, *recurringActionItem.CreatorID)
		}
	}

	if len(actionItems) == 0 {
		// logger.Info("No recurring action items to create")
		return
	}

	// logger.Info("Creating recurring action items", "count", len(actionItems))
	err := c.db.
		Session(&gorm.Session{
			// Skip the encryption step as it can be slow and isn't needed
			SkipHooks: true,
		}).
		Create(actionItems).Error
	if err != nil {
		logger.Error(err.Error())
		return
	}

	c.emailRoutineOwners(emailCreators)
}

// emailRoutineOwners sends one short email to each owner (once, even if several
// of their routines came up together) who asked to be emailed. Like the daily
// reminder email it only goes to a confirmed address, and it never contains
// the routine's text: jobs are stored unencrypted, and the text is private.
func (c *RecurringActionItemController) emailRoutineOwners(ids []uuid.UUID) {
	if len(ids) == 0 {
		return
	}
	var users []*models.User
	err := c.db.
		Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Error), SkipHooks: true}).
		Select("id", "email", "first_name", "last_name", "email_confirmed_at").
		Where("id IN ?", ids).
		Where("email_confirmed_at IS NOT NULL").
		Find(&users).Error
	if err != nil {
		logger.Error("Error loading routine email recipients", "error", err.Error())
		return
	}
	for _, user := range users {
		data := &jobs.EmailData{
			Name: emails.GENERAL,
			Variables: &emails.GeneralVariables{
				Subject: "A routine is ready",
				Message: fmt.Sprintf("A routine of yours just came up. Whenever you're ready: %s/dashboard", os.Getenv("ROOT_URL")),
			},
			Recipients: []*emails.EmailRecipient{{Name: user.FullName(), Email: user.Email}},
		}
		if _, err := jobs.ScheduleEmailJob(user.ID, data, time.Now()); err != nil {
			logger.Error("Error queueing routine email", "error", err.Error())
		}
	}
}

// subscribedCreators reports which of the routines' owners currently have the
// paid features on. Only the subscription columns are loaded.
func (c *RecurringActionItemController) subscribedCreators(routines []*models.RecurringActionItem) map[uuid.UUID]bool {
	seen := map[uuid.UUID]bool{}
	var ids []uuid.UUID
	for _, r := range routines {
		if r.CreatorID != nil && !seen[*r.CreatorID] {
			seen[*r.CreatorID] = true
			ids = append(ids, *r.CreatorID)
		}
	}

	result := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return result
	}
	var users []*models.User
	err := c.db.
		Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Error), SkipHooks: true}).
		Select("id", "subscription_status", "trial_ends_at", "current_period_end").
		Where("id IN ?", ids).
		Find(&users).Error
	if err != nil {
		logger.Error("Error loading routine owners", "error", err.Error())
		return result
	}
	for _, u := range users {
		result[u.ID] = u.IsSubscribed()
	}
	return result
}

func (c *RecurringActionItemController) getRecurringActionItem(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	id := ctx.Params("id")
	var recurringActionItem models.RecurringActionItem
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		First(&recurringActionItem).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "RecurringActionItem not found"})
	}
	ctx.Locals(utils.LocalRecurringActionItem, &recurringActionItem)

	return ctx.Next()
}

func (c *RecurringActionItemController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/:id/form", c.getRecurringActionItem, c.getRecurringActionItemForm)
}

func (c *RecurringActionItemController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	// Creating needs an active plan; editing or deleting what already exists
	// never does.
	c.api.Post("/", middleware.RequireSubscription, c.createRecurringActionItem)
	c.api.Put("/:id", c.getRecurringActionItem, c.updateRecurringActionItem)
	c.api.Delete("/:id", c.getRecurringActionItem, c.deleteRecurringActionItem)
}

type RecurringActionItemBody struct {
	Text            string `form:"text"`
	StartsAt        int    `form:"startsAt"`
	Frequency       int    `form:"frequency"`
	HourlyFrequency int    `form:"hourlyFrequency"`
	EmailMe         string `form:"emailMe"`
}

func (c *RecurringActionItemController) parseRecurringActionItemFromBody(ctx *fiber.Ctx, recurringActionItem *models.RecurringActionItem) error {
	var body RecurringActionItemBody
	err := ctx.BodyParser(&body)
	if err != nil {
		return err
	}

	if body.Text == "" {
		return errors.New("text is required")
	}
	recurringActionItem.Text = body.Text
	recurringActionItem.EmailMe = body.EmailMe == "true" || body.EmailMe == "on"

	if body.StartsAt != 0 {
		date := time.UnixMilli(int64(body.StartsAt))
		recurringActionItem.StartsAt = &date
	} else if recurringActionItem.StartsAt != nil {
		recurringActionItem.StartsAt = nil
	}

	var duration time.Duration
	if body.Frequency == 0 {
		if body.HourlyFrequency == 0 {
			return errors.New("frequency is required")
		}
		duration = time.Duration(body.HourlyFrequency) * time.Hour
	} else {
		duration = time.Duration(body.Frequency) * time.Millisecond
	}
	recurringActionItem.Frequency = duration

	return nil
}

func (c *RecurringActionItemController) getRecurringActionItemForm(ctx *fiber.Ctx) error {
	recurringActionItem := utils.GetLocal[models.RecurringActionItem](ctx, utils.LocalRecurringActionItem)
	component := recurringActionItems.Form(ctx, recurringActionItem)
	return utils.RenderComponent(component, ctx)
}

func (c *RecurringActionItemController) createRecurringActionItem(ctx *fiber.Ctx) error {
	var recurringActionItem models.RecurringActionItem
	err := c.parseRecurringActionItemFromBody(ctx, &recurringActionItem)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	recurringActionItem.Base = &models.Base{CreatorID: &user.ID, LastUpdaterID: &user.ID}

	err = c.db.Create(&recurringActionItem).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action item"})
	}

	components := []templ.Component{
		recurringActionItems.ListItem(ctx, &recurringActionItem),
		recurringActionItems.Form(ctx, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *RecurringActionItemController) updateRecurringActionItem(ctx *fiber.Ctx) error {
	recurringActionItem := utils.GetLocal[models.RecurringActionItem](ctx, utils.LocalRecurringActionItem)

	err := c.parseRecurringActionItemFromBody(ctx, recurringActionItem)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	recurringActionItem.Base.LastUpdaterID = &user.ID

	err = c.db.Save(recurringActionItem).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating recurring action item"})
	}

	components := []templ.Component{
		recurringActionItems.ListItem(ctx, recurringActionItem),
		recurringActionItems.Form(ctx, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *RecurringActionItemController) deleteRecurringActionItem(ctx *fiber.Ctx) error {
	recurringActionItem := utils.GetLocal[models.RecurringActionItem](ctx, utils.LocalRecurringActionItem)

	err := c.db.Delete(recurringActionItem).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting recurring action item"})
	}

	return ctx.Status(http.StatusOK).JSON(recurringActionItem)
}
