package seeders

import (
	"journal/internal/models"
	"math/rand"
	"time"
)

type mockRoutine struct {
	text      string
	frequency time.Duration
	daysAgo   int
}

var mockRoutines = []mockRoutine{
	{"Stretch for a few minutes", 24 * time.Hour, 60},
	{"Reach out to someone I trust", 7 * 24 * time.Hour, 84},
	{"Plan the week ahead", 7 * 24 * time.Hour, 84},
}

// SeedRoutines creates recurring action items for the test user along with the
// action items they would have generated so far: most completed within their
// period, a few left undone, and the current period's still waiting.
func SeedRoutines() error {
	var user *models.User
	if err := db.First(&user, "email = ?", "test@example.com").Error; err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, mock := range mockRoutines {
		startsAt := now.AddDate(0, 0, -mock.daysAgo).Truncate(time.Hour)
		routine := &models.RecurringActionItem{
			Base: &models.Base{
				CreatorID:     &user.ID,
				LastUpdaterID: &user.ID,
				CreatedAt:     startsAt,
				UpdatedAt:     startsAt,
			},
			Text:      mock.text,
			Frequency: mock.frequency,
			StartsAt:  &startsAt,
		}
		if err := db.Create(routine).Error; err != nil {
			return err
		}

		items := []*models.ActionItem{}
		for created := startsAt; created.Before(now); created = created.Add(mock.frequency) {
			item := &models.ActionItem{
				Base: &models.Base{
					CreatorID:     &user.ID,
					LastUpdaterID: &user.ID,
					CreatedAt:     created,
					UpdatedAt:     created,
				},
				Text:                  mock.text,
				RecurringActionItemID: &routine.ID,
			}
			current := created.Add(mock.frequency).After(now)
			if !current && rand.Float64() < 0.8 {
				completed := created.Add(time.Duration(rand.Int63n(int64(mock.frequency) * 3 / 4)))
				item.CompletedAt = &completed
			}
			items = append(items, item)
		}
		if len(items) > 0 {
			if err := db.Create(&items).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
