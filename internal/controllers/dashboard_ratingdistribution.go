package controllers

import (
	"time"

	"journal/internal/models"
	"journal/internal/patterns"
	"journal/internal/utils"
	"journal/internal/web/dashboard"

	"github.com/gofiber/fiber/v2"
)

// Rating distribution -- how many entries picked each rating.
// Only date and rating are loaded; entry text is never read.
func (c *DashboardController) getRatingDistribution(ctx *fiber.Ctx) error {
	loc, err := time.LoadLocation(ctx.Cookies("tz", "UTC"))
	if err != nil {
		loc = time.UTC
	}
	days := ctx.QueryInt("days", 90)
	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -days)

	currentUser := utils.GetLocal[models.User](ctx, utils.LocalCurrentUser)
	var journals []*models.Journal
	c.db.Select("date", "rating_id").
		Where("creator_id = ?", currentUser.ID).
		Where("date >= ?", start.UTC()).
		Preload("Rating").
		Find(&journals)

	values := make([]int, 0, len(journals))
	for _, journal := range journals {
		if journal.Rating == nil {
			continue
		}
		values = append(values, journal.Rating.Value)
	}

	validValues := []int{}
	if ratings := utils.GetLocal[[]*models.Rating](ctx, utils.LocalRatings); ratings != nil {
		for _, rating := range *ratings {
			validValues = append(validValues, rating.Value)
		}
	}

	data := patterns.ComputeRatingDistribution(values, validValues)
	ctx.Locals(utils.LocalRatingDistributionData, &data)

	return utils.RenderComponent(dashboard.RatingDistributionContent(ctx), ctx)
}
