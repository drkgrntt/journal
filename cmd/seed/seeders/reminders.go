package seeders

import (
	"journal/internal/models"
)

var mockReminders = []string{
	"You don't have to solve everything today.",
	"Be patient with yourself. You're doing better than you think.",
	"Rest is part of the work.",
}

func SeedReminders() error {
	reminders := []*models.Reminder{}
	for _, text := range mockReminders {
		reminders = append(reminders, &models.Reminder{
			Base: &models.Base{CreatorID: &admin.ID, LastUpdaterID: &admin.ID},
			Text: text,
		})
	}
	return db.Create(&reminders).Error
}
