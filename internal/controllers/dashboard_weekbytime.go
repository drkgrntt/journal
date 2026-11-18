package controllers

import (
	"time"

	"journal/internal/models"
	"journal/internal/patterns"
	"journal/internal/utils"
	"journal/internal/web/dashboard"

	"github.com/gofiber/fiber/v2"
)

// Week by time of day -- average rating per weekday x time-of-day block.
// Only date and rating are loaded; entry text is never read.
func (c *DashboardController) getWeekByTime(ctx *fiber.Ctx) error {
	loc, err := time.LoadLocation(ctx.Cookies("tz", "UTC"))
	if err != nil {
		loc = time.UTC
	}
	days := ctx.QueryInt("days", 90)
	granularity := ctx.Query("granularity", "period")
	if granularity != "3hour" {
		granularity = "period"
	}
	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -days)

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal
	c.db.Select("date", "rating_id").
		Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", start.UTC()).
		Preload("Rating").
		Find(&journals)

	entries := make([]patterns.WeekByTimeEntry, 0, len(journals))
	for _, journal := range journals {
		if journal.Date == nil || journal.Rating == nil {
			continue
		}
		value := journal.Rating.Value
		entries = append(entries, patterns.WeekByTimeEntry{Date: *journal.Date, RatingValue: &value})
	}

	data := patterns.ComputeWeekByTime(entries, loc, granularity)
	ctx.Locals(utils.LocalWeekByTimeData, &data)

	return utils.RenderComponent(dashboard.WeekByTimeContent(ctx), ctx)
}
