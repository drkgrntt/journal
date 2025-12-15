package controllers

import (
	"fmt"
	"go-starter/cmd/web/journal"
	"go-starter/internal/logger"
	"go-starter/internal/middleware"
	"go-starter/internal/models"
	"go-starter/internal/utils"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&JournalController{})
}

type JournalController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *JournalController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("journal")
	c.api = app.Group("api/journal")
}

func (c *JournalController) getJournal(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	id := ctx.Params("id")
	var journal models.Journal
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		Preload("JournalType").
		Preload("Rating").
		Preload("ActionItems").
		First(&journal).Error
	if err != nil {
		ctx.Set("HX-Redirect", "/journal")
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Journal not found"})
	}
	ctx.Locals("journal", &journal)
	return ctx.Next()
}

func (c *JournalController) getJournals(ctx *fiber.Ctx) error {
	page := ctx.QueryInt("page")
	pageSize := 10

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	journals := []*models.Journal{}
	err := c.db.Where("creator_id = ?", currentUser.ID).
		Preload("JournalType").
		Preload("Rating").
		Order("date desc").
		Order("created_at desc").
		Limit(pageSize + 1).
		Offset(page * pageSize).
		Find(&journals).Error

	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "No journals found"})
	}

	if len(journals) > pageSize {
		journals = journals[:pageSize]
		hasMore := true
		nextPage := page + 1
		ctx.Locals("hasMore", &hasMore)
		ctx.Locals("nextPage", &nextPage)
	}

	ctx.Locals("journals", &journals)
	return ctx.Next()
}

type JournalBody struct {
	Date          int      `form:"date"`
	JournalTypeID int      `form:"journalType"`
	RatingID      int      `form:"rating"`
	Entry         string   `form:"entry"`
	ActionItemIDs []string `form:"actionItemIds"`
}

func (c *JournalController) parseJournalFromBody(ctx *fiber.Ctx, journal *models.Journal) error {
	var body JournalBody
	err := ctx.BodyParser(&body)

	if err != nil {
		return err
	}

	if body.Date != 0 {
		date := time.UnixMilli(int64(body.Date))
		journal.Date = &date
	}
	journal.Entry = body.Entry
	journal.JournalTypeID = body.JournalTypeID
	journal.RatingID = body.RatingID

	journal.Rating = nil
	journal.JournalType = nil

	if len(body.ActionItemIDs) > 0 {
		actionItems := []*models.ActionItem{}
		err := c.db.Where("id in (?)", body.ActionItemIDs).Find(&actionItems).Error
		if err != nil {
			return err
		}
		journal.ActionItems = actionItems
	}

	return nil
}

func (c *JournalController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/", c.getJournals, utils.RenderPage(journal.ListPage))
	c.views.Get("/new", middleware.SetJournalTypes, middleware.SetRatings, utils.RenderPage(journal.NewPage))
	c.views.Get("/list", c.getJournals, utils.RenderPage(journal.ListItems))

	c.views.Get("/:id", c.getJournal, utils.RenderPage(journal.ViewPage))
	c.views.Get("/:id/edit", c.getJournal, middleware.SetJournalTypes, middleware.SetRatings, utils.RenderPage(journal.EditPage))
}

func (c *JournalController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Post("/", c.createJournal)
	c.api.Put("/:id", c.getJournal, c.updateJournal)
	c.api.Delete("/:id", c.getJournal, c.deleteJournal)
}

func (c *JournalController) createJournal(ctx *fiber.Ctx) error {
	var journal models.Journal
	err := c.parseJournalFromBody(ctx, &journal)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	journal.Base = &models.Base{CreatorID: user.ID, LastUpdaterID: user.ID}

	tx := c.db.Begin()
	defer tx.Rollback()

	if tx.Error != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
	}

	err = tx.Create(journal).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
	}

	for _, actionItem := range journal.ActionItems {
		actionItem.JournalID = journal.ID
	}

	if len(journal.ActionItems) > 0 {
		err = tx.Save(&journal.ActionItems).Error
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action items"})
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action items"})
	}

	ctx.Set("HX-Redirect", fmt.Sprintf("/journal/%s", journal.ID.String()))
	return ctx.Status(http.StatusCreated).JSON(journal)
}

func (c *JournalController) updateJournal(ctx *fiber.Ctx) error {
	journal := utils.GetLocal[models.Journal](ctx, "journal")

	err := c.parseJournalFromBody(ctx, journal)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, "currentUser")
	journal.Base.LastUpdaterID = user.ID

	err = c.db.Save(journal).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating journal"})
	}

	ctx.Set("HX-Redirect", fmt.Sprintf("/journal/%s", journal.ID.String()))
	return ctx.Status(http.StatusOK).JSON(journal)
}

func (c *JournalController) deleteJournal(ctx *fiber.Ctx) error {
	journal := utils.GetLocal[models.Journal](ctx, "journal")
	err := c.db.Delete(journal).Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting journal"})
	}

	ctx.Set("HX-Redirect", "/journal")
	return ctx.Status(http.StatusOK).JSON(journal)
}
