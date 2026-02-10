package seeders

import (
	"journal/internal/models"

	"github.com/google/uuid"
)

func SeedUsers() error {
	admin = &models.User{
		FirstName: "Admin",
		LastName:  "Test",
		Email:     "test@example.com",
		Password:  "test",
		Base: &models.Base{
			CreatorID:     &uuid.Nil,
			LastUpdaterID: &uuid.Nil,
		},
	}
	err := db.FirstOrCreate(&admin, "email = ?", "test@example.com").Error
	return err
}
