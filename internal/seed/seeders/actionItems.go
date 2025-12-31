package seeders

import (
	"journal/internal/models"
	"math/rand"
	"time"
)

var mockActionItems = []string{
	"Take a 10-minute walk",
	"Write down three things I’m grateful for",
	"Reach out to someone I trust",
	"Spend 15 minutes planning tomorrow",
	"Do one small task I’ve been avoiding",
	"Go to bed 30 minutes earlier",
	"Limit social media tonight",
	"Stretch or move for a few minutes",
	"Reflect on what went well today",
	"Step outside for fresh air",
}

func maybeCompleted(createdAt time.Time) *time.Time {
	// ~95% chance completed
	if rand.Float64() < 0.95 {
		completed := createdAt.Add(time.Duration(rand.Intn(12)+1) * time.Hour)
		return &completed
	}
	return nil
}

func seedActionItems() error {
	var user *models.User
	err := db.First(&user, "email = ?", "test@example.com").Error
	if err != nil {
		return err
	}

	var journals []*models.Journal
	if err := db.Find(&journals, "creator_id = ?", user.ID).Error; err != nil {
		return err
	}

	actionItems := []*models.ActionItem{}

	for _, journal := range journals {
		// Not every journal needs action items
		count := rand.Intn(4) // 0–3

		for i := 0; i < count; i++ {
			text := mockActionItems[rand.Intn(len(mockActionItems))]

			base := &models.Base{
				CreatorID:     journal.CreatorID,
				LastUpdaterID: journal.CreatorID,
				CreatedAt:     *journal.Date,
				UpdatedAt:     *journal.Date,
			}

			actionItem := &models.ActionItem{
				Text:      text,
				JournalID: journal.ID,
				Base:      base,
			}

			actionItem.CompletedAt = maybeCompleted(base.CreatedAt)

			actionItems = append(actionItems, actionItem)
		}
	}

	if len(actionItems) == 0 {
		return nil
	}

	return db.
		Create(&actionItems).
		Error
}
