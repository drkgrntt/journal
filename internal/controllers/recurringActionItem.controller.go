package controllers

import (
	"errors"
	gormlogger "gorm.io/gorm/logger"
	"journal/cmd/web/recurringActionItems"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/go-co-op/gocron/v2"
	"github.com/gofiber/fiber/v2"
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

	// s.Start()
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
				return tx.Order("created_at DESC").Limit(1)
			},
		).
		Where("starts_at <= ?", now).
		Find(&recurringActionItems)

	var actionItems []*models.ActionItem
	for _, recurringActionItem := range recurringActionItems {
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
			RecurringActionItemID: recurringActionItem.ID,
			Base: &models.Base{
				CreatorID:     recurringActionItem.CreatorID,
				LastUpdaterID: recurringActionItem.LastUpdaterID,
			},
		}
		actionItems = append(actionItems, &actionItem)
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
	}
}

func (c *RecurringActionItemController) getRecurringActionItem(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	id := ctx.Params("id")
	var recurringActionItem models.RecurringActionItem
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		First(&recurringActionItem).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "RecurringActionItem not found"})
	}
	ctx.Locals("recurringActionItem", &recurringActionItem)

	return ctx.Next()
}

// func (c *RecurringActionItemController) getActionItems(ctx *fiber.Ctx) error {
// 	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
//
// 	var actionItems []*models.ActionItem
// 	tx := c.db.
// 		Preload("Journal.JournalType").
// 		Where("creator_id = ?", currentUser.ID).
// 		Order("created_at desc")
//
// 	completedParam := ctx.Query("completed")
// 	if completedParam != "" {
// 		isCompleted := completedParam == "true"
// 		if isCompleted {
// 			tx = tx.Where("completed_at IS NOT NULL")
// 		} else {
// 			tx = tx.Where("completed_at IS NULL")
// 		}
// 	}
//
// 	journalTypeParam := ctx.Query("journalType")
// 	if journalTypeParam != "" {
// 		tx = tx.Where("journal_id IN (SELECT id FROM journals WHERE journal_type_id IN (SELECT id FROM journal_types WHERE code = ?))", journalTypeParam)
// 	}
//
// 	pageSize := 10
// 	page := ctx.QueryInt("page")
// 	tx = tx.Limit(pageSize + 1).Offset(page * pageSize)
//
// 	tx.Find(&actionItems)
// 	ctx.Locals("actionItems", &actionItems)
//
// 	if len(actionItems) > pageSize {
// 		actionItems = actionItems[:pageSize]
// 		hasMore := true
// 		nextPage := page + 1
// 		ctx.Locals("hasMore", &hasMore)
// 		ctx.Locals("nextPage", &nextPage)
// 	}
//
// 	return ctx.Next()
// }

func (c *RecurringActionItemController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	// c.views.Get("/list", c.getActionItems, utils.RenderPage(actionItems.ListItems))
	c.views.Get("/:id/form", c.getRecurringActionItem, c.getRecurringActionItemForm)
}

func (c *RecurringActionItemController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createRecurringActionItem)
	c.api.Put("/:id", c.getRecurringActionItem, c.updateRecurringActionItem)
	c.api.Delete("/:id", c.getRecurringActionItem, c.deleteRecurringActionItem)
}

type RecurringActionItemBody struct {
	Text      string `form:"text"`
	StartsAt  int    `form:"startsAt"`
	Frequency int    `form:"frequency"`
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

	if body.StartsAt != 0 {
		date := time.UnixMilli(int64(body.StartsAt))
		recurringActionItem.StartsAt = &date
	} else if recurringActionItem.StartsAt != nil {
		recurringActionItem.StartsAt = nil
	}

	duration := time.Duration(body.Frequency) * time.Millisecond
	recurringActionItem.Frequency = duration

	return nil
}

func (c *RecurringActionItemController) getRecurringActionItemForm(ctx *fiber.Ctx) error {
	recurringActionItem := utils.GetLocal[models.RecurringActionItem](ctx, "recurringActionItem")
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

	user := utils.GetLocal[models.User](ctx, "currentUser")
	recurringActionItem.Base = &models.Base{CreatorID: user.ID, LastUpdaterID: user.ID}

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
	recurringActionItem := utils.GetLocal[models.RecurringActionItem](ctx, "recurringActionItem")

	err := c.parseRecurringActionItemFromBody(ctx, recurringActionItem)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	recurringActionItem.Base.LastUpdaterID = user.ID

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
	recurringActionItem := utils.GetLocal[models.RecurringActionItem](ctx, "recurringActionItem")

	err := c.db.Delete(recurringActionItem).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting recurring action item"})
	}

	return ctx.Status(http.StatusOK).JSON(recurringActionItem)
}
