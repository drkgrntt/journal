package seeders

import "go-starter/internal/models"

func seedUsers() {
	admin = &models.User{
		FirstName: "Admin",
		LastName:  "Test",
		Email:     "test@example.com",
		Password:  "test",
	}
	db.Create(&admin)
}
