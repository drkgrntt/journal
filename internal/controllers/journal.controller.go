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

func (c *JournalController) setOutstandingActionItems(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	var actionItems []*models.ActionItem
	c.db.
		// Created by me
		Where("creator_id = ?", currentUser.ID).
		// Incomplete
		Where("completed_at IS NULL").
		Order("created_at desc").
		Find(&actionItems)

	ctx.Locals("outstandingActionItems", &actionItems)
	return ctx.Next()
}

func (c *JournalController) getJournal(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	id := ctx.Params("id")
	var journal models.Journal
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", currentUser.ID).
		Preload("JournalType").
		Preload("Rating").
		Preload("ActionItems", c.db.Order("created_at desc")).
		Preload("Thankfuls", c.db.Order("created_at desc")).
		First(&journal).Error

	if err != nil {
		ctx.Set("HX-Redirect", "/journal")
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Journal not found"})
	}

	c.db.
		// None from this journal
		Where("journal_id != ?", journal.ID).
		// Created by me
		Where("creator_id = ?", currentUser.ID).
		// Created before this journal
		Where("created_at < ?", journal.CreatedAt).
		// Incomplete or completed after this journal
		Where("completed_at IS NULL OR completed_at > ?", journal.CreatedAt).
		Order("created_at desc").
		Find(&journal.OutstandingActionItems)

	ctx.Locals("journal", &journal)
	return ctx.Next()
}

func (c *JournalController) getSurroundingJournals(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	journal := utils.GetLocal[models.Journal](ctx, "journal")

	err := c.db.Where("creator_id = ?", currentUser.ID).
		Where("created_at < ?", journal.CreatedAt).
		Order("created_at desc").
		First(&journal.PreviousJournal).Error
	if err != nil {
		journal.PreviousJournal = nil
	}

	err = c.db.Where("creator_id = ?", currentUser.ID).
		Where("created_at > ?", journal.CreatedAt).
		Order("created_at asc").
		First(&journal.NextJournal).Error
	if err != nil {
		journal.NextJournal = nil
	}

	return ctx.Next()
}

func (c *JournalController) getJournals(ctx *fiber.Ctx) error {
	page := ctx.QueryInt("page")
	pageSize := 10

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	journals := []*models.Journal{}
	tx := c.db.Where("creator_id = ?", currentUser.ID).
		Preload("JournalType").
		Preload("Rating").
		Preload("Thankfuls").
		Preload("ActionItems").
		Order("date desc").
		Order("created_at desc")

	date := ctx.Query("date")
	tz := ctx.Query("tz")
	if date != "" && tz != "" {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return err
		}

		tz, err := time.LoadLocation(tz)
		if err != nil {
			return err
		}

		t = t.In(tz)
		tx = tx.Where("date >= ? AND date < ?", t, t.AddDate(0, 0, 1))
	}

	topic := ctx.Query("topic")
	if topic != "" {
		tx = tx.Where("journal_type_id IN (SELECT id FROM journal_types WHERE code = ?)", topic)
	}

	isSortByDate := ctx.Query("sort") == "date"

	if isSortByDate {
		daySubquery := c.db.
			Table("journals").
			Select("date_trunc('day', date)").
			Where("creator_id = ?", currentUser.ID)

		if topic != "" {
			daySubquery = daySubquery.Where("journal_type_id IN (SELECT id FROM journal_types WHERE code = ?)", topic)
		}

		daySubquery.Group("date_trunc('day', date)").
			Order("date_trunc('day', date) DESC").
			Offset(page).
			Limit(1)

		tx.Where(
			"date >= (?) AND date < (?) + INTERVAL '1 day'",
			daySubquery,
			daySubquery,
		)
	} else {
		tx.Limit(pageSize + 1).
			Offset(page * pageSize)
	}

	err := tx.Find(&journals).Error

	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "No journals found"})
	}

	if isSortByDate {
		nextPage := page + 1
		nextDaySubquery := c.db.
			Table("journals").
			Select("1").
			Where("creator_id = ?", currentUser.ID)

		if topic != "" {
			nextDaySubquery = nextDaySubquery.Where("journal_type_id IN (SELECT id FROM journal_types WHERE code = ?)", topic)
		}

		nextDaySubquery.Group("date_trunc('day', date)").
			Order("date_trunc('day', date) DESC").
			Offset(nextPage).
			Limit(1)

		var hasMore bool
		c.db.Raw(
			"SELECT EXISTS (?)",
			nextDaySubquery,
		).Scan(&hasMore)

		ctx.Locals("hasMore", &hasMore)
		ctx.Locals("nextPage", &nextPage)
	} else {
		if len(journals) > pageSize {
			journals = journals[:pageSize]
			hasMore := true
			nextPage := page + 1
			ctx.Locals("hasMore", &hasMore)
			ctx.Locals("nextPage", &nextPage)
		}
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
	ThankfulIDs   []string `form:"thankfulIds"`
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

	if len(body.ThankfulIDs) > 0 {
		thankfuls := []*models.Thankful{}
		err := c.db.Where("id in (?)", body.ThankfulIDs).Find(&thankfuls).Error
		if err != nil {
			return err
		}
		journal.Thankfuls = thankfuls
	}

	return nil
}

func (c *JournalController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)

	c.views.Get("/", middleware.SetJournalTypes, c.getJournals, utils.RenderPage(journal.ListPage))
	c.views.Get("/new", middleware.SetJournalTypes, middleware.SetRatings, c.setOutstandingActionItems, utils.RenderPage(journal.NewPage))
	c.views.Get("/list", c.getJournals, utils.RenderPage(journal.ListItems))

	c.views.Get("/:id", c.getJournal, c.getSurroundingJournals, utils.RenderPage(journal.ViewPage))
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
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
	}

	actionItems := journal.ActionItems
	thankfuls := journal.Thankfuls
	journal.ActionItems = nil
	journal.Thankfuls = nil

	err = tx.Create(&journal).Error
	if err != nil {
		logger.Error(err.Error())
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
	}

	for _, actionItem := range actionItems {
		if actionItem.HasJournal() {
			continue
		}
		actionItem.JournalID = journal.ID
	}

	if len(actionItems) > 0 {
		err = tx.Save(&actionItems).Error
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action items"})
		}
	}

	for _, thankful := range thankfuls {
		if thankful.HasJournal() {
			continue
		}
		thankful.JournalID = journal.ID
	}

	if len(thankfuls) > 0 {
		err = tx.Save(&thankfuls).Error
		if err != nil {
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating thankfuls"})
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action items"})
	}

	addAnotherEntry := ctx.FormValue("addAnotherEntry")
	if addAnotherEntry == "true" {
		ctx.Set("HX-Redirect", "/journal/new")
	} else {
		ctx.Set("HX-Redirect", fmt.Sprintf("/journal/%s", journal.ID.String()))
	}

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

	addAnotherEntry := ctx.FormValue("addAnotherEntry")
	if addAnotherEntry == "true" {
		ctx.Set("HX-Redirect", "/journal/new")
	} else {
		ctx.Set("HX-Redirect", fmt.Sprintf("/journal/%s", journal.ID.String()))
	}

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
