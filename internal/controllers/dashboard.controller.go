package controllers

import (
	"journal/internal/logger"
	"journal/internal/middleware"
	"journal/internal/models"
	"journal/internal/patterns"
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

// ratingValueRange returns the min/max Value across the real (non-placeholder) Rating rows,
// used to clamp a computed average rating into a valid bucket. A rating with Value == 0 is a
// placeholder/unset marker, not a real tier, and is excluded from the range.
func ratingValueRange(ratings []*models.Rating) (min int, max int) {
	first := true
	for _, rating := range ratings {
		if rating.Value == 0 {
			continue
		}
		if first {
			min = rating.Value
			max = rating.Value
			first = false
			continue
		}
		if rating.Value < min {
			min = rating.Value
		}
		if rating.Value > max {
			max = rating.Value
		}
	}
	return min, max
}

// bucketDaysByRating averages each day's ratings and tallies how many days round into each
// rating tier, clamping the rounded average into [min, max] so it can never land outside the
// real rating range. Days with no ratings (an empty slice -- e.g. a day with a completed action
// item or a thankful but no rated journal entry) are skipped entirely rather than producing a
// 0/0 NaN average that would silently fail to match any bucket.
func bucketDaysByRating(daysToValues map[string][]int, min, max int) map[int]int {
	counts := make(map[int]int)
	for _, values := range daysToValues {
		if len(values) == 0 {
			continue
		}
		total := 0
		for _, value := range values {
			total += value
		}
		average := float64(total) / float64(len(values))
		bucket := int(math.Round(average))
		if bucket < min {
			bucket = min
		} else if bucket > max {
			bucket = max
		}
		counts[bucket]++
	}
	return counts
}

func (c *DashboardController) setJournals(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal

	c.db.
		Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", time.Now().AddDate(0, -1, -1)).
		Preload("Rating").
		Preload("JournalType").
		Order("created_at DESC").
		Find(&journals)

	ctx.Locals(utils.LocalJournals, &journals)

	return ctx.Next()
}

func (c *DashboardController) setOutstandingActionItems(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)

	var actionItems []*models.ActionItem
	c.db.
		// Created by me
		Where("creator_id = ?", currentUser.ID).
		// Incomplete
		Where("completed_at IS NULL").
		Order("created_at DESC").
		Find(&actionItems)

	ctx.Locals(utils.LocalOutstandingActionItems, &actionItems)
	return ctx.Next()
}

// dashboardPreferences is stored in the current user's Metadata jsonb column
// (see models.EncodeMetadata/CastMetadata) -- HiddenPatterns holds the keys
// (dashboard.PatternKeys) of Patterns-tab charts the user has opted out of.
type dashboardPreferences struct {
	HiddenPatterns []string `json:"hiddenPatterns"`
}

func hiddenPatternSet(user *models.User) map[string]bool {
	hidden := map[string]bool{}
	err, prefs := models.CastMetadata[dashboardPreferences](user.Metadata)
	if err != nil {
		return hidden
	}
	for _, key := range prefs.HiddenPatterns {
		hidden[key] = true
	}
	return hidden
}

func (c *DashboardController) setHiddenPatterns(ctx *fiber.Ctx) error {
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	hidden := hiddenPatternSet(currentUser)
	ctx.Locals(utils.LocalHiddenPatterns, &hidden)
	return ctx.Next()
}

// setPatternHidden opts the current user in or out of a single Patterns-tab
// chart, keyed by dashboard.PatternKeys, and persists it to User.Metadata.
func (c *DashboardController) setPatternHidden(ctx *fiber.Ctx, hidden bool) error {
	key := ctx.Params("key")
	if !dashboard.IsValidPatternKey(key) {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": "Unknown pattern"})
	}

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	set := hiddenPatternSet(currentUser)
	if hidden {
		set[key] = true
	} else {
		delete(set, key)
	}

	updated := make([]string, 0, len(set))
	for k := range set {
		updated = append(updated, k)
	}
	sort.Strings(updated)

	metadata, err := models.EncodeMetadata(dashboardPreferences{HiddenPatterns: updated})
	if err != nil {
		logger.Error(err.Error())
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error saving preference"})
	}
	currentUser.Metadata = metadata

	if err := c.db.Save(currentUser).Error; err != nil {
		logger.Error(err.Error())
		utils.SetRequestError(ctx, err)
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "Error saving preference"})
	}

	ctx.Set("HX-Redirect", "/dashboard?tab=patterns")
	return ctx.SendStatus(http.StatusOK)
}

func (c *DashboardController) hidePattern(ctx *fiber.Ctx) error {
	return c.setPatternHidden(ctx, true)
}

func (c *DashboardController) showPattern(ctx *fiber.Ctx) error {
	return c.setPatternHidden(ctx, false)
}

func (c *DashboardController) RegisterViewRoutes() {
	c.views.Use(middleware.RequireAuth)
	c.views.Get("/",
		middleware.SetRatings,
		middleware.SetJournalTypes,
		c.setJournals,
		c.setOutstandingActionItems,
		middleware.SetDailyReminder,
		c.setHiddenPatterns,
		utils.RenderPage(dashboard.DashboardPage),
	)
	c.views.Get("/calendar", middleware.SetRatings, c.getCalendar)
	c.views.Get("/year", middleware.SetRatings, c.getYearInPixels)
	c.views.Get("/mood-chart", middleware.SetRatings, c.getMoodChart)
	c.views.Get("/mood-vs-action-completion", c.getMoodVsActionCompletion)
	c.views.Get("/mood-vs-thankfulness", c.getMoodVsThankfulness)

	c.views.Get("/mood-by-topic", middleware.SetRatings, c.getMoodByTopic)
	c.views.Get("/week-by-time", middleware.SetRatings, c.getWeekByTime)
	c.views.Get("/rating-distribution", middleware.SetRatings, c.getRatingDistribution)
	c.views.Get("/routine-completion-rate", c.getRoutineCompletionRate)
	c.views.Get("/topic-distribution", c.getTopicDistribution)
}

func (c *DashboardController) RegisterApiRoutes() {
	c.api.Use(middleware.RequireAuth)

	c.api.Put("/patterns/:key/hide", c.hidePattern)
	c.api.Put("/patterns/:key/show", c.showPattern)
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
	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	lens := patterns.NormalizeLens(ctx.Query("lens"))
	windowStart := date.AddDate(0, 0, -1).UTC()
	windowEnd := date.AddDate(0, 1, 1).UTC()

	if lens == patterns.LensRating {
		var journals []*models.Journal

		c.db.
			Where("creator_id = ?", currentUser.ID).
			Where("date BETWEEN ? AND ?", windowStart, windowEnd).
			Preload("Rating").
			Preload("JournalType").
			Order("created_at DESC").
			Find(&journals)

		ctx.Locals(utils.LocalJournals, &journals)
		return utils.RenderComponent(dashboard.RatingCalendar(ctx), ctx)
	}

	// The other lenses only need when something happened, so pluck just the
	// timestamp column; no entry, thankful, or action item text is loaded.
	var times []time.Time
	var tx *gorm.DB
	var column string
	switch lens {
	case patterns.LensWrote:
		column = "date"
		tx = c.db.Model(&models.Journal{})
	case patterns.LensThankful:
		column = "created_at"
		tx = c.db.Model(&models.Thankful{})
	default: // patterns.LensDid
		column = "completed_at"
		tx = c.db.Model(&models.ActionItem{}).Where("completed_at IS NOT NULL")
	}
	if err := tx.
		Where("creator_id = ?", currentUser.ID).
		Where(column+" BETWEEN ? AND ?", windowStart, windowEnd).
		Pluck(column, &times).Error; err != nil {
		utils.SetRequestError(ctx, err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	counts := patterns.CountByLocalDay(times, loc)
	ctx.Locals(utils.LocalCalendarCounts, &counts)

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

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
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
	ctx.Locals(utils.LocalMoodChartData, &moodChartData)

	return utils.RenderComponent(dashboard.MoodChartContent(ctx), ctx)
}

func (c *DashboardController) getMoodVsActionCompletion(ctx *fiber.Ctx) error {
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

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", date.UTC()).
		Preload("Rating").
		Find(&journals)

	var actionItems []*models.ActionItem
	c.db.Where("creator_id = ?", currentUser.ID).
		Where("completed_at IS NOT NULL").
		Where("completed_at >= ?", date.UTC()).
		Find(&actionItems)

	var ratings []*models.Rating
	c.db.Order("value DESC").
		Find(&ratings)

	minRatingValue, maxRatingValue := ratingValueRange(ratings)

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

	// Bucket each day's average rating (clamped into the valid rating range) exactly once,
	// skipping days with no rated journal entries instead of dividing by zero.
	withCompletionsByRating := bucketDaysByRating(daysWithCompletedActionItems, minRatingValue, maxRatingValue)
	withoutCompletionsByRating := bucketDaysByRating(daysWithoutCompletedActionItems, minRatingValue, maxRatingValue)

	type MoodVsActionCompletionData struct {
		RatingValue        int    `json:"-"`
		Rating             string `json:"rating"`
		WithCompletions    int    `json:"withCompletions"`
		WithoutCompletions int    `json:"withoutCompletions"`
		// Percent (0-100) of each group's days that landed at this rating, so
		// groups of different sizes can be compared.
		WithShare    float64 `json:"withShare"`
		WithoutShare float64 `json:"withoutShare"`
	}
	moodVsActionCompletionData := []MoodVsActionCompletionData{}

	ratingValues := []int{}
	for _, rating := range ratings {
		if rating.Value != 0 {
			ratingValues = append(ratingValues, rating.Value)
		}
	}
	withShares := patterns.ShareByBucket(withCompletionsByRating, ratingValues)
	withoutShares := patterns.ShareByBucket(withoutCompletionsByRating, ratingValues)

	for _, rating := range ratings {
		if rating.Value == 0 {
			continue
		}

		moodVsActionCompletionData = append(moodVsActionCompletionData, MoodVsActionCompletionData{
			RatingValue:        rating.Value,
			Rating:             rating.Name,
			WithCompletions:    withCompletionsByRating[rating.Value],
			WithoutCompletions: withoutCompletionsByRating[rating.Value],
			WithShare:          withShares[rating.Value],
			WithoutShare:       withoutShares[rating.Value],
		})
	}
	ctx.Locals(utils.LocalMoodVsActionCompletionData, &moodVsActionCompletionData)

	return utils.RenderComponent(dashboard.MoodVsActionCompletionContent(ctx), ctx)
}

func (c *DashboardController) getMoodVsThankfulness(ctx *fiber.Ctx) error {
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

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", date.UTC()).
		Preload("Rating").
		Preload("Thankfuls").
		Find(&journals)

	ctx.Locals(utils.LocalJournals, &journals)

	var ratings []*models.Rating
	c.db.Order("value DESC").
		Find(&ratings)

	minRatingValue, maxRatingValue := ratingValueRange(ratings)

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

	// Bucket each day's average rating (clamped into the valid rating range) exactly once,
	// skipping days with no rated journal entries instead of dividing by zero.
	withThankfulsByRating := bucketDaysByRating(daysWithThankfuls, minRatingValue, maxRatingValue)
	withoutThankfulsByRating := bucketDaysByRating(daysWithoutThankfuls, minRatingValue, maxRatingValue)

	type MoodVsThankfulData struct {
		RatingValue      int    `json:"-"`
		Rating           string `json:"rating"`
		WithThankfuls    int    `json:"withThankfuls"`
		WithoutThankfuls int    `json:"withoutThankfuls"`
		// Percent (0-100) of each group's days that landed at this rating, so
		// groups of different sizes can be compared.
		WithShare    float64 `json:"withShare"`
		WithoutShare float64 `json:"withoutShare"`
	}
	moodVsThankfulData := []MoodVsThankfulData{}

	ratingValues := []int{}
	for _, rating := range ratings {
		if rating.Value != 0 {
			ratingValues = append(ratingValues, rating.Value)
		}
	}
	withShares := patterns.ShareByBucket(withThankfulsByRating, ratingValues)
	withoutShares := patterns.ShareByBucket(withoutThankfulsByRating, ratingValues)

	for _, rating := range ratings {
		if rating.Value == 0 {
			continue
		}

		moodVsThankfulData = append(moodVsThankfulData, MoodVsThankfulData{
			RatingValue:      rating.Value,
			Rating:           rating.Name,
			WithThankfuls:    withThankfulsByRating[rating.Value],
			WithoutThankfuls: withoutThankfulsByRating[rating.Value],
			WithShare:        withShares[rating.Value],
			WithoutShare:     withoutShares[rating.Value],
		})
	}

	ctx.Locals(utils.LocalMoodVsThankfulData, &moodVsThankfulData)

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

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
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
	ctx.Locals(utils.LocalMoodByTopicData, &moodByTopicData)

	return utils.RenderComponent(dashboard.MoodByTopicContent(ctx), ctx)
}

// Routine completion rate -- which routines are getting done and which aren't, presented neutrally
func (c *DashboardController) getRoutineCompletionRate(ctx *fiber.Ctx) error {
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

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var routines []*models.RecurringActionItem

	c.db.Where("creator_id = ?", currentUser.ID).
		Preload("ActionItems", func(tx *gorm.DB) *gorm.DB {
			return tx.Where("created_at >= ?", date.UTC()).
				Where("completed_at IS NOT NULL")
		}).
		Find(&routines)

	type RoutineCompletionCount struct {
		Routine   string `json:"routine"`
		Completed int    `json:"completed"`
	}
	routineCompletionRates := []RoutineCompletionCount{}
	for _, routine := range routines {
		routineCompletionRates = append(routineCompletionRates, RoutineCompletionCount{
			Routine:   routine.Text,
			Completed: len(routine.ActionItems),
		})
	}
	sort.Slice(routineCompletionRates, func(i, j int) bool {
		return routineCompletionRates[i].Routine < routineCompletionRates[j].Routine
	})
	ctx.Locals(utils.LocalRoutineCompletionRateData, &routineCompletionRates)

	return utils.RenderComponent(dashboard.RoutineCompletionRateContent(ctx), ctx)
}

// Topic distribution -- a snapshot of how your entries split across topics (not a trend over time).
func (c *DashboardController) getTopicDistribution(ctx *fiber.Ctx) error {
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

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal

	c.db.Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", date.UTC()).
		Preload("JournalType").
		Preload("CustomJournalType").
		Find(&journals)

	distByTopicTmp := map[string]int{}
	for _, journal := range journals {
		topic := journal.JournalType.Name
		if journal.CustomJournalType != nil {
			topic = journal.CustomJournalType.Name
		}
		if _, ok := distByTopicTmp[topic]; !ok {
			distByTopicTmp[topic] = 0
		}
		distByTopicTmp[topic] = distByTopicTmp[topic] + 1
	}

	type DistByTopicData struct {
		Quantity int    `json:"quantity"`
		Topic    string `json:"topic"`
	}
	distByTopicData := []DistByTopicData{}
	for topic, quantity := range distByTopicTmp {
		distByTopicData = append(distByTopicData, DistByTopicData{
			Quantity: quantity,
			Topic:    topic,
		})
	}
	sort.Slice(distByTopicData, func(i, j int) bool {
		return distByTopicData[i].Topic < distByTopicData[j].Topic
	})
	ctx.Locals(utils.LocalDistByTopicData, &distByTopicData)

	return utils.RenderComponent(dashboard.DistByTopicContent(ctx), ctx)
}
