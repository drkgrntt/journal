package seeders

import (
	"journal/internal/models"
	"math/rand"
	"time"
)

var mockJournalEntries = []string{
	"Today felt heavier than I expected. Nothing specific went wrong, but I carried a low-level anxiety most of the day. I tried to notice it without judging it, which helped a little.",

	"Woke up feeling rested for once. Coffee tasted better. Took a short walk and noticed how quiet the neighborhood was. Small moments like that make a difference.",

	"Work was frustrating today. Too many meetings, not enough progress. I did manage to write down a few concrete next steps, which made it feel less overwhelming.",

	"I’m grateful for how patient I was with myself today. Normally I’d spiral over mistakes, but I caught myself and slowed down.",

	"Spent time thinking about long-term goals. It’s still unclear what the next year looks like, but I feel more comfortable admitting that uncertainty.",

	"Had a good conversation with someone I trust. It reminded me that I don’t need to solve everything on my own.",

	"Energy was low today. I didn’t push myself, and that felt like the right call. Rest is productive too.",

	"Noticed how often I reach for my phone when I feel bored or uncomfortable. Tried sitting with that feeling instead.",

	"Felt proud of myself for finishing something I’d been putting off. It wasn’t perfect, but it was done.",

	"Today was quiet. Nothing remarkable happened, and that was okay.",
}

func getMockJournalEntry() string {
	return mockJournalEntries[rand.Intn(len(mockJournalEntries))]
}

func clampRating(r int) int {
	if r < 1 {
		return 1
	}
	if r > 5 {
		return 5
	}
	return r
}

func nextMood(prev int) int {
	roll := rand.Float64()

	switch {
	case roll < 0.60:
		// 60%: stay the same
		return prev
	case roll < 0.85:
		// 25%: small change
		if rand.Intn(2) == 0 {
			return clampRating(prev - 1)
		}
		return clampRating(prev + 1)
	default:
		// 15%: larger swing
		if rand.Intn(2) == 0 {
			return clampRating(prev - 2)
		}
		return clampRating(prev + 2)
	}
}

func normalizeMood(mood int) int {
	if mood > 3 && rand.Float64() < 0.2 {
		return mood - 1
	}
	if mood < 3 && rand.Float64() < 0.2 {
		return mood + 1
	}
	return mood
}

func SeedJournals() error {
	var user *models.User
	var journalTypes []*models.JournalType
	var ratings []*models.Rating

	if err := db.First(&user, "email = ?", "test@example.com").Error; err != nil {
		return err
	}
	if err := db.Find(&journalTypes).Error; err != nil {
		return err
	}
	if err := db.Find(&ratings).Error; err != nil {
		return err
	}

	// Map rating value -> Rating model
	ratingByValue := map[int]*models.Rating{}
	for _, r := range ratings {
		ratingByValue[r.Value] = r // assuming Value is 1–5
	}

	journals := []*models.Journal{}

	currentMood := 3 // start “Fine”

	for i := 0; i < 400; i++ {
		entriesOnDay := rand.Intn(4) + 1 // always at least 1

		// Evolve mood once per day
		currentMood = nextMood(currentMood)
		currentMood = normalizeMood(currentMood)

		rating := ratingByValue[currentMood]

		for j := 0; j < entriesOnDay; j++ {
			date := time.Now().AddDate(0, 0, -i)

			base := &models.Base{
				CreatorID:     user.ID,
				LastUpdaterID: user.ID,
				CreatedAt:     date,
				UpdatedAt:     date,
			}

			journalType := journalTypes[rand.Intn(len(journalTypes))]

			journals = append(journals, &models.Journal{
				Base:        base,
				Date:        &date,
				Entry:       getMockJournalEntry(),
				JournalType: journalType,
				Rating:      rating,
			})
		}
	}

	return db.Create(&journals).Error
}
