package server

import (
	"github.com/gofiber/fiber/v2"

	"journal/internal/database"
)

type FiberServer struct {
	*fiber.App

	db *database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "journal",
			AppName:      "journal",
		}),

		db: database.New(),
	}

	return server
}
