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

func init() {
	db = database.New().DB
}

func Seed() {

	var err error

	err = SeedUsers()
	if err != nil {
		log.Fatal(err)
	}

	err = SeedRatings()
	if err != nil {
		log.Fatal(err)
	}

	err = SeedJournalTypes()
	if err != nil {
		log.Fatal(err)
	}

	err = SeedJournals()
	if err != nil {
		log.Fatal(err)
	}

	err = SeedActionItems()
	if err != nil {
		log.Fatal(err)
	}

	err = SeedThankfuls()
	if err != nil {
		log.Fatal(err)
	}
}
