package controllers

import (
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/dashboard"
	"math"
	"net/http"
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

	c.views.Get("/mood-by-topic", c.getMoodByTopic)
	c.views.Get("/time-of-day-patterns", c.getTimeOfDayPatterns)
	c.views.Get("/entry-time-frequency", c.getEntryTimeFrequency)
	c.views.Get("/thankful-frequency", c.getThankfulFrequency)
	c.views.Get("/routine-completion-rate", c.getRoutineCompletionRate)
	c.views.Get("/topic-distribution", c.getTopicDistribution)
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

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	tx := c.db.
		Where("creator_id = ?", currentUser.ID).
		Preload("Rating").
		Preload("JournalType").
		Order("created_at DESC")

	t := time.Now()
	end := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		loc,
	)
	start := end.AddDate(0, -1, -1)
	if month != 0 && year != 0 {
		date := time.Date(
			year,
			time.Month(month),
			1, 0, 0, 0, 0,
			loc,
		)
		start = date.AddDate(0, 0, -1)
		end = date.AddDate(0, 1, 1)
	}
	tx = tx.Where("date BETWEEN ? AND ?", start.UTC(), end.UTC()).
		Find(&journals)

	moodChartTmp := map[string][]int{}
	for _, journal := range journals {
		if journal.Rating == nil {
			continue
		}
		localizedDate := journal.Date.In(loc).Format("01-02-2006")

		if _, ok := moodChartTmp[localizedDate]; !ok {
			moodChartTmp[localizedDate] = []int{}
		}
		moodChartTmp[localizedDate] = append(moodChartTmp[localizedDate], journal.Rating.Value)
	}

	type MoodChartData struct {
		Value    *float64  `json:"value"`
		Date     string    `json:"date"`
		DateTime time.Time `json:"-"`
	}
	moodChartData := []MoodChartData{}
	for date, values := range moodChartTmp {
		total := 0
		for _, value := range values {
			total += value
		}
		dateTime, _ := time.Parse("01-02-2006", date)
		value := float64(total) / float64(len(values))
		moodChartData = append(moodChartData, MoodChartData{
			Value:    &value,
			Date:     date,
			DateTime: dateTime,
		})
	}

	for i := start; i.Before(end); i = i.AddDate(0, 0, 1) {
		localizedDate := i.In(loc).Format("01-02-2006")
		if _, ok := moodChartTmp[localizedDate]; !ok {
			moodChartData = append(moodChartData, MoodChartData{
				Value:    nil,
				Date:     localizedDate,
				DateTime: i,
			})
		}
	}

	sort.Slice(moodChartData, func(i, j int) bool {
		return moodChartData[i].DateTime.Before(moodChartData[j].DateTime)
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
	days := ctx.QueryInt("days", 30)
	t := time.Now()
	date := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		loc,
	).AddDate(0, 0, -days)

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", date.UTC()).
		Preload("Rating").
		Find(&journals)

	moodByDayTmp := map[time.Weekday][]int{}
	for _, journal := range journals {
		if journal.Rating == nil {
			continue
		}
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
		if journal.Rating == nil {
			continue
		}
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
		if rating.Value == 0 {
			continue
		}
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
			if journal.Rating == nil {
				continue
			}
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
		if rating.Value == 0 {
			continue
		}
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

// Mood by topic -- average rating per topic over time, so you can see things like "my professional entries consistently rate lower than family ones"
func (c *DashboardController) getMoodByTopic(ctx *fiber.Ctx) error {
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	days := ctx.QueryInt("days", 30)
	t := time.Now()
	date := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		loc,
	).AddDate(0, 0, -days)

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", date.UTC()).
		Preload("Rating").
		Preload("JournalType").
		Preload("CustomJournalType").
		Find(&journals)

	moodByTopicTmp := map[string][]int{}
	for _, journal := range journals {
		if journal.Rating == nil {
			continue
		}
		topic := journal.JournalType.Name
		if journal.CustomJournalType != nil {
			topic = journal.CustomJournalType.Name
		}
		if _, ok := moodByTopicTmp[topic]; !ok {
			moodByTopicTmp[topic] = []int{}
		}
		moodByTopicTmp[topic] = append(moodByTopicTmp[topic], journal.Rating.Value)
	}

	type MoodByTopicData struct {
		Value float64 `json:"value"`
		Topic string  `json:"topic"`
	}
	moodByTopicData := []MoodByTopicData{}
	for topic, values := range moodByTopicTmp {
		total := 0
		for _, value := range values {
			total += value
		}
		moodByTopicData = append(moodByTopicData, MoodByTopicData{
			Value: float64(total) / float64(len(values)),
			Topic: topic,
		})
	}
	sort.Slice(moodByTopicData, func(i, j int) bool {
		return moodByTopicData[i].Topic < moodByTopicData[j].Topic
	})
	ctx.Locals("moodByTopicData", &moodByTopicData)

	return utils.RenderComponent(dashboard.MoodByTopicContent(ctx), ctx)
}

// Time of day patterns -- do morning entries rate differently than evening ones
func (c *DashboardController) getTimeOfDayPatterns(ctx *fiber.Ctx) error {
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	days := ctx.QueryInt("days", 30)
	t := time.Now()
	date := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		loc,
	).AddDate(0, 0, -days)

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", date.UTC()).
		Preload("Rating").
		Find(&journals)

	moodByTodTmp := map[string][]int{}
	for _, journal := range journals {
		if journal.Rating == nil {
			continue
		}
		tod := journal.Date.In(loc).Format("15:00")
		if _, ok := moodByTodTmp[tod]; !ok {
			moodByTodTmp[tod] = []int{}
		}
		moodByTodTmp[tod] = append(moodByTodTmp[tod], journal.Rating.Value)
	}

	type MoodByTodData struct {
		Value float64 `json:"value"`
		Tod   string  `json:"tod"`
	}
	moodByTodData := []MoodByTodData{}
	for tod, values := range moodByTodTmp {
		total := 0
		for _, value := range values {
			total += value
		}
		moodByTodData = append(moodByTodData, MoodByTodData{
			Value: float64(total) / float64(len(values)),
			Tod:   tod,
		})
	}
	sort.Slice(moodByTodData, func(i, j int) bool {
		return moodByTodData[i].Tod < moodByTodData[j].Tod
	})
	ctx.Locals("moodByTodData", &moodByTodData)

	return utils.RenderComponent(dashboard.MoodByTodContent(ctx), ctx)
}

// Entry frequency heatmap -- GitHub-style contribution graph showing journaling
func (c *DashboardController) getEntryTimeFrequency(ctx *fiber.Ctx) error {
	month := ctx.QueryInt("month")
	year := ctx.QueryInt("year")
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var journals []*models.Journal

	tx := c.db.
		Where("creator_id = ?", currentUser.ID).
		Preload("Rating").
		Preload("JournalType").
		Preload("CustomJournalType").
		Order("created_at DESC")

	t := time.Now()
	end := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		loc,
	)
	start := end.AddDate(0, -1, -1)
	if month != 0 && year != 0 {
		date := time.Date(
			year,
			time.Month(month),
			1, 0, 0, 0, 0,
			loc,
		)
		start = date.AddDate(0, 0, -1)
		end = date.AddDate(0, 1, 1)
	}
	tx = tx.Where("date BETWEEN ? AND ?", start.UTC(), end.UTC()).
		Find(&journals)

	frequencyChartTmp := map[string][]int{}
	for _, journal := range journals {
		if journal.Rating == nil {
			journal.Rating = &models.Rating{Value: 0}
		}
		localizedDate := journal.Date.In(loc).Format("01-02-2006")

		if _, ok := frequencyChartTmp[localizedDate]; !ok {
			frequencyChartTmp[localizedDate] = []int{}
		}
		frequencyChartTmp[localizedDate] = append(frequencyChartTmp[localizedDate], journal.Rating.Value)
	}

	type FrequencyChartData struct {
		Quantity *float64  `json:"quantity"`
		Rating   *float64  `json:"rating"`
		Date     string    `json:"date"`
		DateTime time.Time `json:"-"`
	}
	frequencyChartData := []FrequencyChartData{}
	for date, values := range frequencyChartTmp {
		total := 0
		qtyWithRatings := 0
		for _, rating := range values {
			total += rating
			if rating > 0 {
				qtyWithRatings++
			}
		}
		dateTime, _ := time.Parse("01-02-2006", date)
		rating := float64(total) / float64(qtyWithRatings)
		frequencyChartData = append(frequencyChartData, FrequencyChartData{
			Quantity: utils.Pointer(float64(len(values))),
			Rating:   &rating,
			Date:     date,
			DateTime: dateTime,
		})
	}

	for i := start; i.Before(end); i = i.AddDate(0, 0, 1) {
		localizedDate := i.In(loc).Format("01-02-2006")
		if _, ok := frequencyChartTmp[localizedDate]; !ok {
			frequencyChartData = append(frequencyChartData, FrequencyChartData{
				Rating:   nil,
				Quantity: utils.Pointer(float64(0)),
				Date:     localizedDate,
				DateTime: i,
			})
		}
	}

	sort.Slice(frequencyChartData, func(i, j int) bool {
		return frequencyChartData[i].DateTime.Before(frequencyChartData[j].DateTime)
	})
	ctx.Locals("frequencyChartData", &frequencyChartData)

	return utils.RenderComponent(dashboard.FrequencyChartContent(ctx), ctx)
}

// Thankful frequency over time -- simple count of how often you include thankfuls, since the behavior itself is the signal
func (c *DashboardController) getThankfulFrequency(ctx *fiber.Ctx) error {
	month := ctx.QueryInt("month")
	year := ctx.QueryInt("year")
	tz := ctx.Cookies("tz", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	currentUser := utils.GetLocal[models.User](ctx, "currentUser")
	var thankfuls []*models.Thankful

	tx := c.db.
		Where("creator_id = ?", currentUser.ID).
		Order("created_at DESC")

	t := time.Now()
	end := time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		0, 0, 0, 0,
		loc,
	)
	start := end.AddDate(0, -1, -1)
	if month != 0 && year != 0 {
		date := time.Date(
			year,
			time.Month(month),
			1, 0, 0, 0, 0,
			loc,
		)
		start = date.AddDate(0, 0, -1)
		end = date.AddDate(0, 1, 1)
	}
	tx = tx.Where("created_at BETWEEN ? AND ?", start.UTC(), end.UTC()).
		Find(&thankfuls)

	thankfulFrequencyChartTmp := map[string]int{}
	for _, thankful := range thankfuls {
		localizedDate := thankful.CreatedAt.In(loc).Format("01-02-2006")
		if _, ok := thankfulFrequencyChartTmp[localizedDate]; !ok {
			thankfulFrequencyChartTmp[localizedDate] = 0
		}
		thankfulFrequencyChartTmp[localizedDate] = thankfulFrequencyChartTmp[localizedDate] + 1
	}

	type ThankfulFrequencyChartData struct {
		Quantity *int      `json:"quantity"`
		Date     string    `json:"date"`
		DateTime time.Time `json:"-"`
	}
	thankfulFrequencyChartData := []ThankfulFrequencyChartData{}
	for date, quantity := range thankfulFrequencyChartTmp {
		dateTime, _ := time.Parse("01-02-2006", date)
		thankfulFrequencyChartData = append(thankfulFrequencyChartData, ThankfulFrequencyChartData{
			Quantity: utils.Pointer(quantity),
			Date:     date,
			DateTime: dateTime,
		})
	}

	for i := start; i.Before(end); i = i.AddDate(0, 0, 1) {
		localizedDate := i.In(loc).Format("01-02-2006")
		if _, ok := thankfulFrequencyChartTmp[localizedDate]; !ok {
			thankfulFrequencyChartData = append(thankfulFrequencyChartData, ThankfulFrequencyChartData{
				Quantity: utils.Pointer(0),
				Date:     localizedDate,
				DateTime: i,
			})
		}
	}

	sort.Slice(thankfulFrequencyChartData, func(i, j int) bool {
		return thankfulFrequencyChartData[i].DateTime.Before(thankfulFrequencyChartData[j].DateTime)
	})
	ctx.Locals("thankfulFrequencyChartData", &thankfulFrequencyChartData)

	return utils.RenderComponent(dashboard.ThankfulFrequencyChartContent(ctx), ctx)
}

// Routine completion rate -- which routines are getting done and which aren't, presented neutrally
func (c *DashboardController) getRoutineCompletionRate(ctx *fiber.Ctx) error {
	return ctx.SendStatus(http.StatusNotImplemented)
}

// Topic distribution over time -- are you writing more about health lately, more about work? Something like a stacked area or donut chart over a selectable time range
func (c *DashboardController) getTopicDistribution(ctx *fiber.Ctx) error {
	return ctx.SendStatus(http.StatusNotImplemented)
}
