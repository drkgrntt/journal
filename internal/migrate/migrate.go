package main

import (
	"journal/internal/database"
	"journal/internal/logger"
)

func main() {
	database.New()
	database.AutoMigrate()
	logger.Info("? AutoMigrated Successfully")
}
