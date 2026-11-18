package seeders

import (
	"journal/internal/models"
)

func SeedEntryTemplates() error {
	templates := []*models.EntryTemplate{
		{
			Base: &models.Base{CreatorID: &admin.ID, LastUpdaterID: &admin.ID},
			Name: "End of day",
			Body: "What went well today?\n\nWhat was harder than I expected?\n\nWhat can wait until tomorrow?\n",
		},
		{
			Base: &models.Base{CreatorID: &admin.ID, LastUpdaterID: &admin.ID},
			Name: "Brain dump",
			Body: "Everything on my mind right now:\n\n",
		},
	}
	return db.Create(&templates).Error
}
