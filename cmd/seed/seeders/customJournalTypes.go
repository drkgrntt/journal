package seeders

import (
	"journal/internal/models"
)

// SeedCustomJournalTypes gives the test user a couple of their own topics
// (a paid feature), which SeedJournals then files some entries under.
func SeedCustomJournalTypes() error {
	base := func() *models.Base {
		return &models.Base{CreatorID: &admin.ID, LastUpdaterID: &admin.ID}
	}
	types := []*models.CustomJournalType{
		{Name: "Parenting", Base: base()},
		{Name: "Woodworking", Base: base()},
	}
	return db.Create(&types).Error
}
