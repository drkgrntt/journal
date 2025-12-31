package main

import (
	"journal/internal/database"
	"journal/internal/models"
	"journal/internal/seed/seeders"
	"log"

	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func main() {
	db = database.New().DB

	var err error

	err = seeders.SeedUsers()
	if err != nil {
		log.Fatal(err)
	}

	deleteItems()

	err = seeders.SeedJournals()
	if err != nil {
		log.Fatal(err)
	}

	err = seeders.SeedActionItems()
	if err != nil {
		log.Fatal(err)
	}

	err = seeders.SeedThankfuls()
	if err != nil {
		log.Fatal(err)
	}
}

func deleteItems() {
	var err error

	var user models.User
	err = db.
		Preload("Journals").
		Preload("ActionItems").
		Preload("RecurringActionItems").
		Preload("Thankfuls").
		First(&user, "email = ?", "test@example.com").Error
	if err != nil {
		log.Fatal(err)
	}

	if len(user.Journals) > 0 {
		err = db.Delete(&user.Journals).Error
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(user.ActionItems) > 0 {
		err = db.Delete(&user.ActionItems).Error
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(user.Thankfuls) > 0 {
		err = db.Delete(&user.Thankfuls).Error
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(user.RecurringActionItems) > 0 {
		err = db.Delete(&user.RecurringActionItems).Error
		if err != nil {
			log.Fatal(err)
		}
	}
}
