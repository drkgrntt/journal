package controllers

import (
	"errors"
	"fmt"
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/journal"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var actionItems []*models.ActionItem
	c.db.
		// Created by me
		Where("creator_id = ?", currentUser.ID).
		// Incomplete
		Where("completed_at IS NULL").
		Order("created_at desc").
		Find(&actionItems)

	ctx.Locals(utils.LocalOutstandingActionItems, &actionItems)
	return ctx.Next()
}

// loadJournal fetches one of the user's journals with the associations the
// edit/update paths need.
func (c *JournalController) loadJournal(userID uuid.UUID, id string) (models.Journal, error) {
	var journal models.Journal
	err := c.db.Where("id = ?", id).
		Where("creator_id = ?", userID).
		Preload("JournalType").
		Preload("CustomJournalType").
		Preload("Rating").
		Preload("ActionItems", c.db.Order("created_at desc")).
		Preload("Thankfuls", c.db.Order("created_at desc")).
		First(&journal).Error
	return journal, err
}

func (c *JournalController) getJournal(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	journal, err := c.loadJournal(currentUser.ID, ctx.Params("id"))

	if err != nil {
		ctx.Set("HX-Redirect", "/journal")
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Journal not found"})
	}

	c.db.
		// None from this journal
		Where("journal_id != ? OR journal_id IS NULL", journal.ID).
		// Created by me
		Where("creator_id = ?", currentUser.ID).
		// Created before this journal
		Where("created_at < ?", journal.CreatedAt.UTC()).
		// Incomplete or completed after this journal
		Where("completed_at IS NULL OR completed_at > ?", journal.CreatedAt.UTC()).
		Order("created_at desc").
		Find(&journal.OutstandingActionItems)

	ctx.Locals(utils.LocalJournal, &journal)
	return ctx.Next()
}

func (c *JournalController) getSurroundingJournals(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	journal := utils.GetLocal[models.Journal](ctx, utils.LocalJournal)

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

// journalFilters applies the list filters shared by every query behind the
// journal list (the page itself, the By Date day picker, and the prev/next
// day lookups), so they can't drift apart. A topic is either a built-in
// type's code or a custom type's SubID; a custom type is matched only among
// the current user's own custom types.
func journalFilters(db *gorm.DB, tx *gorm.DB, userID uuid.UUID, topic string, bookmarkedOnly bool) *gorm.DB {
	tx = tx.Where("creator_id = ?", userID)
	if topic != "" {
		tx = tx.Where(db.Where(
			"journal_type_id IN (SELECT id FROM journal_types WHERE code = ?)",
			topic,
		).Or(
			"custom_journal_type_id IN (SELECT id FROM custom_journal_types WHERE creator_id = ? AND right(id::text, length(?)) = ?)",
			userID, topic, topic,
		))
	}
	if bookmarkedOnly {
		tx = tx.Where("bookmarked_at IS NOT NULL")
	}
	return tx
}

func (c *JournalController) getJournals(ctx *fiber.Ctx) error {
	page := ctx.QueryInt("page")
	pageSize := 10

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	topic := ctx.Query("topic")
	date := ctx.Query("date")
	isSortByBookmark := ctx.Query("sort") == "bookmark"
	isSortByDate := ctx.Query("sort") == "date"

	// tz is interpolated into SQL below, so it must be a name LoadLocation
	// accepts; anything else falls back to UTC.
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		tz = "UTC"
		loc = time.UTC
	}

	var parsedDate time.Time
	if date != "" {
		parsedDate, err = time.Parse("2006-01-02", date)
		if err != nil {
			return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Invalid date"})
		}
	}

	// A specific day shows that whole day; By Date without one pages through
	// the days that have entries.
	pagesByDay := isSortByDate && date == ""
	dayExpr := fmt.Sprintf("date_trunc('day', date AT TIME ZONE '%s')", tz)

	journals := []*models.Journal{}
	tx := journalFilters(c.db, c.db, currentUser.ID, topic, isSortByBookmark).
		Preload("JournalType").
		Preload("CustomJournalType").
		Preload("Rating").
		Preload("Thankfuls").
		Preload("ActionItems").
		Order("date desc").
		Order("created_at desc")

	if date != "" {
		start := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, loc)
		tx = tx.Where("date >= ? AND date < ?", start.UTC(), start.AddDate(0, 0, 1).UTC())
	} else if pagesByDay {
		// dayExpr is a zone-less local timestamp, so compare it against the
		// same expression on each row rather than against the raw column.
		daySubquery := journalFilters(c.db, c.db.Table("journals"), currentUser.ID, topic, isSortByBookmark).
			Select(dayExpr).
			Group(dayExpr).
			Order(dayExpr + " DESC").
			Offset(page).
			Limit(1)
		tx = tx.Where(dayExpr+" = (?)", daySubquery)
	} else {
		tx = tx.Limit(pageSize + 1).Offset(page * pageSize)
	}

	if err := tx.Find(&journals).Error; err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "No journals found"})
	}

	if date != "" || pagesByDay {
		var pageDate time.Time
		if date != "" {
			pageDate = parsedDate
		} else if len(journals) > 0 {
			pageDate = journals[0].Date.In(loc)
		}

		if !pageDate.IsZero() {
			startLocal := time.Date(pageDate.Year(), pageDate.Month(), pageDate.Day(), 0, 0, 0, 0, loc)
			endLocal := startLocal.AddDate(0, 0, 1)

			var prev time.Time
			err = journalFilters(c.db, c.db.Model(&models.Journal{}), currentUser.ID, topic, isSortByBookmark).
				Select("date(date AT TIME ZONE ?)", tz).
				Where("date < ?", startLocal.UTC()).
				Order("date DESC").
				Limit(1).
				Scan(&prev).Error
			if err != nil {
				return err
			}
			if !prev.IsZero() {
				prevDate := prev.Format("2006-01-02")
				ctx.Locals(utils.LocalPrevDate, &prevDate)
			}

			var next time.Time
			err = journalFilters(c.db, c.db.Model(&models.Journal{}), currentUser.ID, topic, isSortByBookmark).
				Select("date(date AT TIME ZONE ?)", tz).
				Where("date >= ?", endLocal.UTC()).
				Order("date ASC").
				Limit(1).
				Scan(&next).Error
			if err != nil {
				return err
			}
			if !next.IsZero() {
				nextDate := next.Format("2006-01-02")
				ctx.Locals(utils.LocalNextDate, &nextDate)
			}
		}
	}

	if pagesByDay {
		nextPage := page + 1

		nextDaySubquery := journalFilters(c.db, c.db.Table("journals"), currentUser.ID, topic, isSortByBookmark).
			Select("1").
			Group(dayExpr).
			Order(dayExpr + " DESC").
			Offset(nextPage).
			Limit(1)

		var hasMore bool
		c.db.Raw("SELECT EXISTS (?)", nextDaySubquery).Scan(&hasMore)

		ctx.Locals(utils.LocalHasMore, &hasMore)
		ctx.Locals(utils.LocalNextPage, &nextPage)
	} else if date == "" && len(journals) > pageSize {
		journals = journals[:pageSize]
		hasMore := true
		nextPage := page + 1
		ctx.Locals(utils.LocalHasMore, &hasMore)
		ctx.Locals(utils.LocalNextPage, &nextPage)
	}

	ctx.Locals(utils.LocalJournals, &journals)
	return ctx.Next()
}

type JournalBody struct {
	Date int `form:"date"`
	// JournalTypeID int      `form:"journalType"`
	JournalTypeID string   `form:"journalType"`
	RatingID      int      `form:"rating"`
	Entry         string   `form:"entry"`
	IsBookmarked  bool     `form:"isBookmarked"`
	ActionItemIDs []string `form:"actionItemIds"`
	ThankfulIDs   []string `form:"thankfulIds"`
	// DraftID is a client-generated id for a brand-new entry, sent with every
	// save of it. The client keeps unsynced drafts on-device and retries them,
	// and a retry after a lost response must not create a second entry - see
	// createJournal.
	DraftID string `form:"draftId"`
}

func (c *JournalController) parseJournalFromBody(ctx *fiber.Ctx, journal *models.Journal) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	if currentUser == nil {
		return errors.New("missing current user")
	}

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
	journal.RatingID = body.RatingID

	journal.CustomJournalTypeID, err = uuid.Parse(body.JournalTypeID)
	if err != nil {
		journal.JournalTypeID, err = strconv.Atoi(body.JournalTypeID)
		if err != nil {
			return err
		}
	} else {
		// Verify the referenced custom journal type belongs to the current
		// user before accepting it, otherwise a user could reference
		// (and thereby associate their journal with) someone else's
		// custom journal type.
		var ownedCustomType models.CustomJournalType
		err = c.db.
			Where("id = ?", journal.CustomJournalTypeID).
			Where("creator_id = ?", currentUser.ID).
			First(&ownedCustomType).Error
		if err != nil {
			return errors.New("custom journal type not found")
		}

		var customType *models.JournalType
		err = c.db.Where("code = ?", "custom").Find(&customType).Error
		if err != nil {
			return err
		}
		journal.JournalTypeID = customType.ID
	}

	journal.Rating = nil
	journal.JournalType = nil
	journal.CustomJournalType = nil

	if body.IsBookmarked {
		now := time.Now()
		journal.BookmarkedAt = &now
	} else if journal.BookmarkedAt != nil {
		journal.BookmarkedAt = nil
	}

	if len(body.ActionItemIDs) > 0 {
		actionItems := []*models.ActionItem{}
		err := c.db.Where("id in (?)", body.ActionItemIDs).
			Where("creator_id = ?", currentUser.ID).
			Find(&actionItems).Error
		if err != nil {
			return err
		}
		journal.ActionItems = actionItems
	}

	if len(body.ThankfulIDs) > 0 {
		thankfuls := []*models.Thankful{}
		err := c.db.Where("id in (?)", body.ThankfulIDs).
			Where("creator_id = ?", currentUser.ID).
			Find(&thankfuls).Error
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
	c.views.Get("/new", middleware.SetJournalTypes, middleware.SetRatings, c.setOutstandingActionItems, middleware.SetDailyReminder, utils.RenderPage(journal.NewPage))
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
	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	// A draft the server has already created an entry for (the first attempt
	// reached us but its response was lost) becomes an update of that entry.
	draftID := strings.TrimSpace(ctx.FormValue("draftId"))
	if draftID != "" {
		var existingID string
		err := c.db.Model(&models.Journal{}).
			Where("creator_id = ?", user.ID).
			Where("metadata->>'draftId' = ?", draftID).
			Limit(1).
			Pluck("id", &existingID).Error
		if err == nil && existingID != "" {
			existing, err := c.loadJournal(user.ID, existingID)
			if err == nil {
				ctx.Locals(utils.LocalJournal, &existing)
				return c.updateJournal(ctx)
			}
		}
	}

	var journal models.Journal
	err := c.parseJournalFromBody(ctx, &journal)
	if err != nil {
		logger.Warn(err.Error())
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	journal.Base = &models.Base{CreatorID: &user.ID, LastUpdaterID: &user.ID}
	if draftID != "" {
		journal.Metadata, err = models.EncodeMetadata(models.JournalMetadata{DraftID: draftID})
		if err != nil {
			logger.Error(err.Error())
			utils.SetRequestError(ctx, err)
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
		}
	}

	tx := c.db.Begin()
	defer tx.Rollback()

	if tx.Error != nil {
		logger.Error(tx.Error.Error())
		utils.SetRequestError(ctx, tx.Error)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
	}

	actionItems := journal.ActionItems
	thankfuls := journal.Thankfuls
	journal.ActionItems = nil
	journal.Thankfuls = nil

	err = tx.Create(&journal).Error
	if err != nil {
		logger.Error(err.Error())
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating journal"})
	}

	for _, actionItem := range actionItems {
		if actionItem.HasJournal() {
			continue
		}
		actionItem.JournalID = &journal.ID
	}

	if len(actionItems) > 0 {
		err = tx.Save(&actionItems).Error
		if err != nil {
			utils.SetRequestError(ctx, err)
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action items"})
		}
	}

	for _, thankful := range thankfuls {
		if thankful.HasJournal() {
			continue
		}
		thankful.JournalID = &journal.ID
	}

	if len(thankfuls) > 0 {
		err = tx.Save(&thankfuls).Error
		if err != nil {
			utils.SetRequestError(ctx, err)
			return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating thankfuls"})
		}
	}

	err = tx.Commit().Error
	if err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error creating action items"})
	}

	isAutosave := ctx.FormValue("autosave") == "true"
	if !isAutosave {
		addAnotherEntry := ctx.FormValue("addAnotherEntry")
		if addAnotherEntry == "true" {
			ctx.Set("HX-Redirect", "/journal/new?addAnother=true")
		} else {
			ctx.Set("HX-Redirect", fmt.Sprintf("/journal/%s", journal.ID.String()))
		}
	}

	return ctx.Status(http.StatusCreated).JSON(journal)
}

func (c *JournalController) updateJournal(ctx *fiber.Ctx) error {
	journal := utils.GetLocal[models.Journal](ctx, utils.LocalJournal)

	err := c.parseJournalFromBody(ctx, journal)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": "Bad request reading body"})
	}

	user := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	journal.Base.LastUpdaterID = &user.ID

	err = c.db.Save(journal).Error
	if err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error updating journal"})
	}

	isAutosave := ctx.FormValue("autosave") == "true"
	if !isAutosave {
		addAnotherEntry := ctx.FormValue("addAnotherEntry")
		if addAnotherEntry == "true" {
			ctx.Set("HX-Redirect", "/journal/new?addAnother=true")
		} else {
			ctx.Set("HX-Redirect", fmt.Sprintf("/journal/%s", journal.ID.String()))
		}
	}

	return ctx.Status(http.StatusOK).JSON(journal)
}

func (c *JournalController) deleteJournal(ctx *fiber.Ctx) error {
	journal := utils.GetLocal[models.Journal](ctx, utils.LocalJournal)
	err := c.db.Delete(journal).Error
	if err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error deleting journal"})
	}

	ctx.Set("HX-Redirect", "/journal")
	return ctx.Status(http.StatusOK).JSON(journal)
}
