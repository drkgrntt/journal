package controllers

import (
	"errors"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/actionItems"
	"net/http"
	"strings"
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
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	id := ctx.Params("id")
	var actionItem models.ActionItem
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		Preload("Journal").
		First(&actionItem).Error

	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "ActionItem not found"})
	}
	ctx.Locals(utils.LocalActionItem, &actionItem)

	return ctx.Next()
}

func (c *ActionItemController) getRecurringActionItems(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var recurringActionItems []*models.RecurringActionItem
	tx := c.db.
		Where("creator_id = ?", currentUser.ID).
		Order("created_at desc")

	tx.Find(&recurringActionItems)
	ctx.Locals(utils.LocalRecurringActionItems, &recurringActionItems)

	return ctx.Next()
}

func (c *ActionItemController) getActionItems(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var actionItems []*models.ActionItem
	tx := c.db.
		Preload("Journal.JournalType").
		Where("creator_id = ?", currentUser.ID).
		Order("created_at desc")

	completedParam := ctx.Query("completed")
	if completedParam != "" {
		isCompleted := completedParam == "true"
		if isCompleted {
			tx = tx.Where("completed_at IS NOT NULL")
		} else {
			tx = tx.Where("completed_at IS NULL")
		}
	}

	journalTypeParam := ctx.Query("journalType")
	if journalTypeParam != "" {
		// A built-in type's code or one of the user's custom types' SubID,
		// matched the same way as the journal list's topic filter.
		tx = tx.Where(
			"journal_id IN (?)",
			journalFilters(c.db, c.db.Table("journals").Select("id"), currentUser.ID, journalTypeParam, false),
		)
	}

	pageSize := 10
	page := ctx.QueryInt("page")
	tx = tx.Limit(pageSize + 1).Offset(page * pageSize)

	if err := tx.Find(&actionItems).Error; err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Could not load action items"})
	}
	ctx.Locals(utils.LocalActionItems, &actionItems)

	if len(actionItems) > pageSize {
		actionItems = actionItems[:pageSize]
		hasMore := true
		nextPage := page + 1
		ctx.Locals(utils.LocalHasMore, &hasMore)
		ctx.Locals(utils.LocalNextPage, &nextPage)
	}

	return ctx.Next()
}

func (c *ActionItemController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get(
		"/",
		middleware.SetJournalTypes,
		c.getActionItems,
		c.getRecurringActionItems,
		utils.RenderPage(actionItems.Page),
	)
	c.views.Get("/list", c.getActionItems, utils.RenderPage(actionItems.ListItems))
	c.views.Get("/:id/form", c.getActionItem, c.getActionItemForm)
}

func (c *ActionItemController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createActionItem)
	c.api.Put("/:id", c.getActionItem, c.updateActionItem)
	c.api.Delete("/:id", c.getActionItem, c.deleteActionItem)
}

type ActionItemBody struct {
	Text string `form:"text"`
	// CompletedAt is unix milliseconds, set by clients replaying an offline
	// completion so the time it was done survives the delay.
	CompletedAt int64  `form:"completedAt"`
	IsComplete  bool   `form:"isComplete"`
	JournalID   string `form:"journalId"`
}

const (
	maxClientIDLen       = 64
	completedAtMaxAge    = 90 * 24 * time.Hour
	completedAtMaxFuture = 5 * time.Minute
)

// parseClientID reads the optional clientId form field a queued offline create
// carries so a replay can be matched to the row it already made.
func parseClientID(ctx *fiber.Ctx) (string, error) {
	clientID := strings.TrimSpace(ctx.FormValue("clientId"))
	if len(clientID) > maxClientIDLen {
		return "", errors.New("clientId too long")
	}
	return clientID, nil
}

// resolveCompletedAt picks the completion time for an item being marked
// complete: the client's time if it is plausible, else the existing one (so a
// retried request doesn't move it), else now.
func resolveCompletedAt(supplied int64, existing *time.Time, now time.Time) time.Time {
	if supplied != 0 {
		t := time.UnixMilli(supplied)
		if !t.Before(now.Add(-completedAtMaxAge)) && !t.After(now.Add(completedAtMaxFuture)) {
			if t.After(now) {
				return now
			}
			return t
		}
	}
	if existing != nil && !existing.IsZero() {
		return *existing
	}
	return now
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
		completedAt := resolveCompletedAt(body.CompletedAt, actionItem.CompletedAt, time.Now())
		actionItem.CompletedAt = &completedAt
	} else {
		actionItem.CompletedAt = nil
	}

	if body.JournalID != "" {
		journalUuid, err := uuid.Parse(body.JournalID)
		if err != nil {
			return err
		}

		user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
		var journal models.Journal
		err = c.db.Where("id = ? AND creator_id = ?", journalUuid, user.ID).First(&journal).Error
		if err != nil {
			return errors.New("journal not found")
		}

		actionItem.JournalID = &journalUuid
	}

	return nil
}

func (c *ActionItemController) getActionItemForm(ctx *fiber.Ctx) error {
	actionItem := utils.GetLocal[models.ActionItem](ctx, utils.LocalActionItem)
	component := actionItems.Form(ctx, actionItem.Journal, actionItem)
	return utils.RenderComponent(component, ctx)
}

func (c *ActionItemController) createActionItem(ctx *fiber.Ctx) error {
	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	clientID, err := parseClientID(ctx)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	// A create the server has already seen (the first attempt's response was
	// lost) becomes an update of that item. Scoped to the current user.
	if clientID != "" {
		var existing models.ActionItem
		err := c.db.Preload("Journal").
			Where("creator_id = ?", user.ID).
			Where("metadata->>'clientId' = ?", clientID).
			First(&existing).Error
		if err == nil {
			ctx.Locals(utils.LocalActionItem, &existing)
			return c.updateActionItem(ctx)
		}
	}

	var actionItem models.ActionItem
	err = c.parseActionItemFromBody(ctx, &actionItem)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	actionItem.Base = &models.Base{CreatorID: &user.ID, LastUpdaterID: &user.ID}
	if clientID != "" {
		actionItem.Metadata, err = models.EncodeMetadata(models.ActionItemMetadata{ClientID: clientID})
		if err != nil {
			utils.SetRequestError(ctx, err)
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action item"})
		}
	}

	err = c.db.Create(&actionItem).Error
	if err != nil {
		logger.Error(err.Error())
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action item"})
	}

	if actionItem.HasJournal() {
		c.db.Where("id = ?", actionItem.JournalID).First(&actionItem.Journal)
	}

	ctx.Set("X-Item-Id", actionItem.ID.String())
	components := []templ.Component{
		actionItems.ListItem(ctx, &actionItem),
		actionItems.Form(ctx, actionItem.Journal, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ActionItemController) updateActionItem(ctx *fiber.Ctx) error {
	actionItem := utils.GetLocal[models.ActionItem](ctx, utils.LocalActionItem)

	err := c.parseActionItemFromBody(ctx, actionItem)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	actionItem.Base.LastUpdaterID = &user.ID

	err = c.db.Save(actionItem).Error
	if err != nil {
		logger.Error(err.Error())
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating journal"})
	}

	if actionItem.HasJournal() {
		c.db.Where("id = ?", actionItem.JournalID).First(&actionItem.Journal)
	}

	ctx.Set("X-Item-Id", actionItem.ID.String())
	components := []templ.Component{
		actionItems.ListItem(ctx, actionItem),
		actionItems.Form(ctx, actionItem.Journal, nil),
	}

	return utils.RenderComponents(components, ctx)
}

func (c *ActionItemController) deleteActionItem(ctx *fiber.Ctx) error {
	actionItem := utils.GetLocal[models.ActionItem](ctx, utils.LocalActionItem)

	err := c.db.Delete(actionItem).Error
	if err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting action item"})
	}

	return ctx.Status(http.StatusOK).JSON(actionItem)
}
