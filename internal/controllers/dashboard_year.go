package controllers

import (
	"journal/internal/models"
	"journal/internal/patterns"
	"journal/internal/utils"
	"journal/internal/web/dashboard"
	"time"

	"github.com/gofiber/fiber/v2"
)

// getYearInPixels renders the Year in Pixels pattern. Only each entry's date
// and rating are loaded (never its text), and only per-day averages go out.
func (c *DashboardController) getYearInPixels(ctx *fiber.Ctx) error {
	loc, err := time.LoadLocation(ctx.Cookies("tz", "UTC"))
	if err != nil {
		loc = time.UTC
	}
	currentYear := time.Now().In(loc).Year()
	year := ctx.QueryInt("year", currentYear)
	if year > currentYear {
		year = currentYear
	}
	if year < 1970 {
		year = 1970
	}

	start := time.Date(year, time.January, 1, 0, 0, 0, 0, loc)
	end := start.AddDate(1, 0, 0)

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal
	c.db.
		Select("date", "rating_id").
		Where("creator_id = ?", currentUser.ID).
		Where("date >= ? AND date < ?", start.UTC(), end.UTC()).
		Preload("Rating").
		Find(&journals)

	entries := make([]patterns.RatedEntry, 0, len(journals))
	for _, j := range journals {
		if j.Date == nil {
			continue
		}
		e := patterns.RatedEntry{Date: *j.Date}
		if j.Rating != nil {
			v := j.Rating.Value
			e.RatingValue = &v
		}
		entries = append(entries, e)
	}

	data := dashboard.YearInPixelsData{
		Year:        year,
		CurrentYear: currentYear,
		Days:        patterns.DailyRatings(entries, loc),
	}
	ctx.Locals(utils.LocalYearInPixels, &data)

	return utils.RenderComponent(dashboard.YearInPixelsContent(ctx), ctx)
}
