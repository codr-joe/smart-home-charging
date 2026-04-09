package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/smart-charging/api/internal/energy"
)

// New creates and configures the Fiber application.
func New(repo *energy.Repository, hub *Hub) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "smart-charging-api"})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
	h := newHandler(repo, hub)
	v1 := app.Group("/api/v1")
	v1.Get("/energy/current", h.getCurrent)
	v1.Get("/energy/history", h.getHistory)
	v1.Get("/stream", h.stream)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	return app
}
