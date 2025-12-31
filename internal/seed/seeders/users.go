package seeders

import "journal/internal/models"

func SeedUsers() error {
	admin = &models.User{
		FirstName: "Admin",
		LastName:  "Test",
		Email:     "test@example.com",
		Password:  "test",
	}
	err := db.FirstOrCreate(&admin, "email = ?", "test@example.com").Error
	return err
}
