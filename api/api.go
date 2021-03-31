package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neicnordic/sda-download/api/middleware"
	"github.com/neicnordic/sda-download/api/sda"
)

// Setup Setup a fiber app with all of its routes
func Setup() *fiber.App {
	// Fiber instance
	app := fiber.New()
	app.Use(middleware.TokenMiddleware())

	// SDA endpoints
	app.Get("/metadata/datasets", func(c *fiber.Ctx) error {
		return sda.Datasets(c)
	})
	app.Get("/metadata/datasets/+/files", func(c *fiber.Ctx) error {
		return sda.Files(c, c.Params(("+")))
	})
	app.Get("/files/:fileId", func(c *fiber.Ctx) error {
		return sda.Download(c, c.Params(("fileId")))
	})

	// S3 endpoints
	// -

	// HTSGET endpoints
	// -

	// DRS endpoints
	// -

	// 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	// Return the configured app
	return app
}
