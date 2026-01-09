package controllers

import (
	"journal/cmd/web/dashboard"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"math"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func init() {
	registerController(&DashboardController{})
}

type DashboardController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *DashboardController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("dashboard")
	c.api = app.Group("api/dashboard")
}

func (c *DashboardController) setJournals(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.
		Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", time.Now().AddDate(0, -1, -1)).
		Preload("Rating").
		Preload("JournalType").
		Order("created_at DESC").
		Find(&journals)

	ctx.Locals("journals", &journals)

	return ctx.Next()
}

func (c *DashboardController) setOutstandingActionItems(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")

	var actionItems []*models.ActionItem
	c.db.
		// Created by me
		Where("creator_id = ?", currentUser.ID).
		// Incomplete
		Where("completed_at IS NULL").
		Order("created_at DESC").
		Find(&actionItems)

	ctx.Locals("outstandingActionItems", &actionItems)
	return ctx.Next()
}

func (c *DashboardController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)
	c.views.Get("/",
		middleware.SetRatings,
		middleware.SetJournalTypes,
		c.setJournals,
		c.setOutstandingActionItems,
		utils.RenderPage(dashboard.DashboardPage),
	)
	c.views.Get("/calendar", c.getCalendar)
	c.views.Get("/mood-chart", c.getMoodChart)
	c.views.Get("/mood-by-day", c.getMoodByDay)
	c.views.Get("/mood-vs-action-completion", c.getMoodVsActionCompletion)
	c.views.Get("/mood-vs-thankfulness", c.getMoodVsThankfulness)
}

func (c *DashboardController) RegisterApiRoutes() {
}

func (c *DashboardController) getCalendar(ctx *fiber.Ctx) error {
	month := ctx.QueryInt("month")
	year := ctx.QueryInt("year")
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	date := time.Date(
		year,
		time.Month(month),
		1,
		0,
		0,
		0,
		0,
		loc,
	)
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.
		Where("creator_id = ?", currentUser.ID).
		Where("date BETWEEN ? AND ?", date.AddDate(0, 0, -1).UTC(), date.AddDate(0, 1, 1).UTC()).
		Preload("Rating").
		Preload("JournalType").
		Order("created_at DESC").
		Find(&journals)

	ctx.Locals("journals", &journals)

	return utils.RenderComponent(dashboard.RatingCalendar(ctx), ctx)
}

func (c *DashboardController) getMoodChart(ctx *fiber.Ctx) error {
	month := ctx.QueryInt("month")
	year := ctx.QueryInt("year")
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	date := time.Date(
		year,
		time.Month(month),
		1,
		0,
		0,
		0,
		0,
		loc,
	)
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	tx := c.db.
		Where("creator_id = ?", currentUser.ID).
		Preload("Rating").
		Preload("JournalType").
		Order("created_at DESC")

	if month != 0 && year != 0 {
		tx = tx.Where("date BETWEEN ? AND ?", date.AddDate(0, 0, -1).UTC(), date.AddDate(0, 1, 1).UTC())
	} else {
		tx = tx.Where("date >= ?", time.Now().AddDate(0, -1, -1))
	}
	tx.Find(&journals)

	moodChartTmp := map[string][]int{}
	for _, journal := range journals {
		localizedDate := journal.Date.In(loc).Format("01-02-2006")

		if _, ok := moodChartTmp[localizedDate]; !ok {
			moodChartTmp[localizedDate] = []int{}
		}
		moodChartTmp[localizedDate] = append(moodChartTmp[localizedDate], journal.Rating.Value)
	}

	type MoodChartData struct {
		Value float64 `json:"value"`
		Date  string  `json:"date"`
	}
	moodChartData := []MoodChartData{}
	for date, values := range moodChartTmp {
		total := 0
		for _, value := range values {
			total += value
		}
		moodChartData = append(moodChartData, MoodChartData{
			Value: float64(total) / float64(len(values)),
			Date:  date,
		})
	}

	sort.Slice(moodChartData, func(i, j int) bool {
		return moodChartData[i].Date < moodChartData[j].Date
	})
	ctx.Locals("moodChartData", &moodChartData)

	return utils.RenderComponent(dashboard.MoodChartContent(ctx), ctx)
}

func (c *DashboardController) getMoodByDay(ctx *fiber.Ctx) error {
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Preload("Rating").
		Find(&journals)

	moodByDayTmp := map[time.Weekday][]int{}
	for _, journal := range journals {
		localizedDate := journal.Date.In(loc).Weekday()
		if _, ok := moodByDayTmp[localizedDate]; !ok {
			moodByDayTmp[localizedDate] = []int{}
		}
		moodByDayTmp[localizedDate] = append(moodByDayTmp[localizedDate], journal.Rating.Value)
	}

	type MoodByDayData struct {
		Value   float64      `json:"value"`
		Day     string       `json:"day"`
		Weekday time.Weekday `json:"-"`
	}
	moodByDayData := []MoodByDayData{}
	for day, values := range moodByDayTmp {
		total := 0
		for _, value := range values {
			total += value
		}
		moodByDayData = append(moodByDayData, MoodByDayData{
			Value:   float64(total) / float64(len(values)),
			Day:     day.String(),
			Weekday: day,
		})
	}
	sort.Slice(moodByDayData, func(i, j int) bool {
		return moodByDayData[i].Weekday < moodByDayData[j].Weekday
	})
	ctx.Locals("moodByDayData", &moodByDayData)

	return utils.RenderComponent(dashboard.MoodByDayContent(ctx), ctx)
}

func (c *DashboardController) getMoodVsActionCompletion(ctx *fiber.Ctx) error {
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Preload("Rating").
		Find(&journals)

	var actionItems []*models.ActionItem
	c.db.Where("creator_id = ?", currentUser.ID).
		Where("completed_at IS NOT NULL").
		Find(&actionItems)

	var ratings []*models.Rating
	c.db.Order("value DESC").
		Find(&ratings)

	daysWithCompletedActionItems := make(map[string][]int)
	for _, actionItem := range actionItems {
		localizedDate := actionItem.CompletedAt.In(loc).Format("01-02-2006")
		if _, ok := daysWithCompletedActionItems[localizedDate]; !ok {
			daysWithCompletedActionItems[localizedDate] = []int{}
		}
	}

	daysWithoutCompletedActionItems := make(map[string][]int)
	for _, journal := range journals {
		localizedDate := journal.Date.In(loc).Format("01-02-2006")
		_, ok := daysWithCompletedActionItems[localizedDate]
		if ok {
			daysWithCompletedActionItems[localizedDate] = append(daysWithCompletedActionItems[localizedDate], journal.Rating.Value)
		} else {
			if _, ok := daysWithoutCompletedActionItems[localizedDate]; !ok {
				daysWithoutCompletedActionItems[localizedDate] = []int{}
			}
			daysWithoutCompletedActionItems[localizedDate] = append(daysWithoutCompletedActionItems[localizedDate], journal.Rating.Value)
		}
	}

	type MoodVsActionCompletionData struct {
		RatingValue        int    `json:"-"`
		Rating             string `json:"rating"`
		WithCompletions    int    `json:"withCompletions"`
		WithoutCompletions int    `json:"withoutCompletions"`
	}
	moodVsActionCompletionData := []MoodVsActionCompletionData{}

	for _, rating := range ratings {
		var withCompletions int
		var withoutCompletions int

		for _, values := range daysWithCompletedActionItems {
			var total int
			for _, value := range values {
				total += value
			}
			average := float64(total) / float64(len(values))
			if math.Round(average) == float64(rating.Value) {
				withCompletions++
			}

		}
		for _, values := range daysWithoutCompletedActionItems {
			var total int
			for _, value := range values {
				total += value
			}
			average := float64(total) / float64(len(values))
			if math.Round(average) == float64(rating.Value) {
				withoutCompletions++
			}
		}

		moodVsActionCompletionData = append(moodVsActionCompletionData, MoodVsActionCompletionData{
			RatingValue:        rating.Value,
			Rating:             rating.Name,
			WithCompletions:    withCompletions,
			WithoutCompletions: withoutCompletions,
		})
	}
	ctx.Locals("moodVsActionCompletionData", &moodVsActionCompletionData)

	return utils.RenderComponent(dashboard.MoodVsActionCompletionContent(ctx), ctx)
}

func (c *DashboardController) getMoodVsThankfulness(ctx *fiber.Ctx) error {
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Preload("Rating").
		Preload("Thankfuls").
		Find(&journals)

	ctx.Locals("journals", &journals)

	var ratings []*models.Rating
	c.db.Order("value DESC").
		Find(&ratings)

	journalsByDay := make(map[string][]*models.Journal)
	for _, journal := range journals {
		localizedDate := journal.Date.In(loc).Format("01-02-2006")
		if _, ok := journalsByDay[localizedDate]; !ok {
			journalsByDay[localizedDate] = []*models.Journal{}
		}
		journalsByDay[localizedDate] = append(journalsByDay[localizedDate], journal)
	}

	daysWithThankfuls := make(map[string][]int)
	daysWithoutThankfuls := make(map[string][]int)
	for date, js := range journalsByDay {
		jRatings := []int{}
		withThankfuls := false
		for _, journal := range js {
			if len(journal.Thankfuls) > 0 {
				withThankfuls = true
			}
			jRatings = append(jRatings, journal.Rating.Value)
		}
		if withThankfuls {
			daysWithThankfuls[date] = jRatings
		} else {
			daysWithoutThankfuls[date] = jRatings
		}
	}

	type MoodVsThankfulData struct {
		RatingValue      int    `json:"-"`
		Rating           string `json:"rating"`
		WithThankfuls    int    `json:"withThankfuls"`
		WithoutThankfuls int    `json:"withoutThankfuls"`
	}
	moodVsThankfulData := []MoodVsThankfulData{}

	for _, rating := range ratings {
		var withThankfuls int
		var withoutThankfuls int

		for _, values := range daysWithThankfuls {
			var total int
			for _, value := range values {
				total += value
			}
			average := float64(total) / float64(len(values))
			if math.Round(average) == float64(rating.Value) {
				withThankfuls++
			}
		}
		for _, values := range daysWithoutThankfuls {
			var total int
			for _, value := range values {
				total += value
			}
			average := float64(total) / float64(len(values))
			if math.Round(average) == float64(rating.Value) {
				withoutThankfuls++
			}
		}

		moodVsThankfulData = append(moodVsThankfulData, MoodVsThankfulData{
			RatingValue:      rating.Value,
			Rating:           rating.Name,
			WithThankfuls:    withThankfuls,
			WithoutThankfuls: withoutThankfuls,
		})
	}

	ctx.Locals("moodVsThankfulData", &moodVsThankfulData)

	return utils.RenderComponent(dashboard.MoodVsThankfulnessContent(ctx), ctx)
}
