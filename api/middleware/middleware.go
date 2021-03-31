package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/pkg/auth"
	log "github.com/sirupsen/logrus"
)

// TokenMiddleware performs access token verification and validation
// JWTs are verified and validated by the app, opaque tokens are sent to AAI for verification
func TokenMiddleware() fiber.Handler {
	log.Info("starting token middleware")
	// Middleware can be configured here and is run once on startup

	// Initialise OIDC configuration
	details, err := auth.GetOIDCDetails(config.Config.OIDC.ConfigurationURL)
	log.Info("retrieving OIDC configuration")
	if err != nil {
		// Web app requires the OIDC configuration to run
		panic("could not retrieve OIDC configuration")
	}
	auth.Details = details
	log.Info("OIDC configuration retrieved")

	// Middleware check is performed here upon request
	return func(c *fiber.Ctx) error {
		log.Debug("begin authorization check")

		// Check that a token is provided
		token, code, err := auth.GetToken(c.Get("Authorization"))
		if err != nil {
			log.Debugf("request rejected, %s, %s", token, err.Error())
			return fiber.NewError(code, err.Error())
		}

		// Verify token by attempting to retrieve visas from AAI
		visas, err := auth.GetVisas(auth.Details, token)
		if err != nil {
			log.Debug("failed to validate token at AAI")
			return fiber.NewError(401, "bad token")
		}

		// Get permissions
		datasets, err := auth.GetPermissions(visas)
		if err != nil {
			log.Errorf("failed to parse dataset permission visas, %s", err)
			return fiber.NewError(500, "an error occurred while parsing visas")
		}
		if len(datasets) == 0 {
			log.Debug("token carries no dataset permissions matching the database")
			return fiber.NewError(404, "no datasets found")
		}
		// Store list of datasets to request context for use at endpoints
		c.Locals("datasets", datasets)

		log.Debug("authorization check passed")
		return c.Next()
	}
}
