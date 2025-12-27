package seeders

import (
	"journal/internal/database"
	"journal/internal/models"
	"log"

	"gorm.io/gorm"
)

var (
	db    *gorm.DB
	admin *models.User
)

func Seed() {
	db = database.New().DB

	var err error

	err = seedUsers()
	if err != nil {
		log.Fatal(err)
	}

	err = seedRatings()
	if err != nil {
		log.Fatal(err)
	}

	err = seedJournalTypes()
	if err != nil {
		log.Fatal(err)
	}

	err = seedJouranls()
	if err != nil {
		log.Fatal(err)
	}
}
