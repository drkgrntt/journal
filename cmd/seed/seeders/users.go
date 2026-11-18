package seeders

import (
	"journal/internal/models"
	"time"

	"github.com/google/uuid"
)

func SeedUsers() error {
	// Comped, so every paid feature stays on in dev (a trial would quietly
	// expire after 60 days), with a confirmed address so the reminder email
	// can be switched on.
	confirmedAt := time.Now().UTC()
	admin = &models.User{
		FirstName: "Admin",
		LastName:  "Test",
		Email:     "test@example.com",
		Password:  "test",

		Timezone:           SeedTimezone,
		SubscriptionStatus: models.SubscriptionComped,
		EmailConfirmedAt:   &confirmedAt,
		Base: &models.Base{
			CreatorID:     &uuid.Nil,
			LastUpdaterID: &uuid.Nil,
		},
	}
	err := db.FirstOrCreate(&admin, "email = ?", "test@example.com").Error
	return err
}
