package main

import (
	"journal/cmd/seed/seeders"
	"journal/internal/database"
	"journal/internal/logger"
	"os"
)

func init() {
	database.New()
}

func main() {
	if os.Getenv("APP_ENV") == "production" {
		logger.Error("Refusing to run seed against a production environment (APP_ENV=production)")
		os.Exit(1)
	}

	database.DropTables()
	logger.Info("Dropped Tables Successfully")

	database.RunMigrations()
	logger.Info("Ran pending SQL migrations successfully")

	seeders.Seed()
	logger.Info("Seeded Successfully")
}
