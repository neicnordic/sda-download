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
	auth.Details = auth.GetOIDCDetails(config.Config.OIDC.ConfigurationURL)
	log.Info("retrieving OIDC configuration")
	if (auth.OIDCDetails{}) == auth.Details {
		// Web app requires the OIDC configuration to run
		panic("could not retrieve OIDC configuration")
	}
	log.Info("OIDC configuration retrieved")

	// Middleware check is performed here upon request
	return func(c *fiber.Ctx) error {
		log.Debug("begin authorization check")

		// Check that a token is provided
		token, errorCode := auth.GetToken(c.Get("Authorization"))
		if errorCode != 0 {
			log.Debugf("request rejected, %s", token) // contains error message
			return fiber.NewError(errorCode, token)
		}

		// Verify token by attempting to retrieve visas from AAI
		valid, visas := auth.GetVisas(auth.Details, token)
		if !valid {
			log.Debug("failed to validate token at AAI")
			return fiber.NewError(401, "bad token")
		}
		// Store visas from ga4gh_passport_v1 in the request context for later use
		// this reduces the number of calls to AAI
		c.Locals("visas", visas)

		log.Debug("authorization check passed")
		return c.Next()
	}
}
