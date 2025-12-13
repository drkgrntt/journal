package seeders

import "go-starter/internal/models"

func seedUsers() error {
	admin = &models.User{
		FirstName: "Admin",
		LastName:  "Test",
		Email:     "test@example.com",
		Password:  "test",
	}
	err := db.Create(&admin).Error
	return err
}
