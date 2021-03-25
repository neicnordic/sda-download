package middleware

import (
	"strings"

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
		if errorCode > 0 {
			return fiber.NewError(errorCode, token)
		}

		// Check if token is JWT or opaque
		if strings.Count(token, ".") == 2 {
			log.Debug("token is JWT")
			// verify token signature
			verifiedToken, reason := auth.VerifyJWT(auth.Details, token)
			if verifiedToken == nil {
				if reason == "aai" {
					return fiber.NewError(500, "AAI request failed")
				} else {
					return fiber.NewError(401, "bad token")
				}
			}
			// verify token claims, e.g. expiration and issuer
			valid := auth.ValidateJWT(auth.Details, verifiedToken)
			if !valid {
				return fiber.NewError(500, "bad token")
			}
		} else {
			log.Debug("token is opaque")
			valid, reason := auth.VerifyOpaque(auth.Details, token)
			if !valid {
				if reason == "aai" {
					return fiber.NewError(500, "AAI request failed")
				} else {
					return fiber.NewError(401, "bad token")
				}
			}
		}
		log.Debug("authorization check passed")
		return c.Next()
	}
}
