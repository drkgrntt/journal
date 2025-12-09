package server

import (
	"github.com/gofiber/fiber/v2"

	"go-starter/internal/database"
)

type FiberServer struct {
	*fiber.App

	db *database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "go-starter",
			AppName:      "go-starter",
		}),

		db: database.New(),
	}

	return server
}
