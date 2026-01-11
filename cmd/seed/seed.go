package main

import (
	"journal/cmd/seed/seeders"
	"journal/internal/database"
	"journal/internal/logger"
)

func init() {
	database.New()
}

var (
	Full        bool
	dropTables  bool
	autoMigrage bool
	testUser    bool
)

func main() {
	database.DropTables()
	logger.Info("Dropped Tables Successfully")

	database.AutoMigrate()
	logger.Info("AutoMigrated Successfully")

	seeders.Seed()
	logger.Info("Seeded Successfully")
}
