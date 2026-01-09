package seeders

import (
	"journal/internal/models"
	"math/rand"
)

var mockThankfuls = []string{
	"Good coffee this morning",
	"A quiet moment to myself",
	"A text from a friend",
	"Getting outside for a few minutes",
	"Finishing something I started",
	"Feeling rested when I woke up",
	"A helpful conversation",
	"Nice weather today",
	"Having a comfortable place to sit and think",
	"Noticing my breath slow down",
	"A warm meal",
	"Something that made me laugh",
}

func pickThankfuls() []string {
	count := rand.Intn(4) // 0–3
	used := map[int]bool{}
	out := []string{}

	for len(out) < count {
		i := rand.Intn(len(mockThankfuls))
		if used[i] {
			continue
		}
		used[i] = true
		out = append(out, mockThankfuls[i])
	}
	return out
}

func SeedThankfuls() error {
	var user *models.User
	err := db.First(&user, "email = ?", "test@example.com").Error
	if err != nil {
		return err
	}

	var journals []*models.Journal
	if err := db.Find(&journals, "creator_id = ?", user.ID).Error; err != nil {
		return err
	}

	thankfuls := []*models.Thankful{}

	for _, journal := range journals {
		for _, text := range pickThankfuls() {
			base := &models.Base{
				CreatorID:     journal.CreatorID,
				LastUpdaterID: journal.CreatorID,
				CreatedAt:     *journal.Date,
				UpdatedAt:     *journal.Date,
			}

			thankful := &models.Thankful{
				Text:      text,
				JournalID: journal.ID,
				Base:      base,
			}

			thankfuls = append(thankfuls, thankful)
		}
	}

	if len(thankfuls) == 0 {
		return nil
	}

	return db.
		Create(&thankfuls).
		Error
}
