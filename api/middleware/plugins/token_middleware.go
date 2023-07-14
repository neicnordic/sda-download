package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/session"
	"github.com/neicnordic/sda-download/pkg/auth"
	log "github.com/sirupsen/logrus"
)

type customMiddleware []string

var CustomMiddleware customMiddleware

// GetDatasets takes a Bearer token and requests a user's
// GA4GH visas from an OIDC provider, and then validates
// the visas and parses them into a list of datasets
func (cm customMiddleware) GetDatasets(c *gin.Context) []string {
	// Check if dataset permissions are cached to session
	sessionCookie, err := c.Cookie(config.Config.Session.Name)
	if err != nil {
		log.Debugf("no session cookie received")
	}
	var datasets []string
	var exists bool
	if sessionCookie != "" {
		log.Debug("session cookie received")
		datasets, exists = session.Get(sessionCookie)
	}

	if !exists {
		log.Debug("no session found, create new session")

		// Check that a token is provided
		token, code, err := auth.GetToken(c.Request.Header)
		if err != nil {
			c.String(code, err.Error())
			c.AbortWithStatus(code)

			return datasets
		}

		// Verify token by attempting to retrieve visas from AAI
		visas, err := auth.GetVisas(auth.Details, token)
		if err != nil {
			log.Debug("failed to validate token at AAI")
			c.String(http.StatusUnauthorized, "get visas failed")
			c.AbortWithStatus(code)

			return datasets
		}

		// Get permissions
		// This used to cause a "404 no datasets found", but now the error has been moved deeper:
		// 200 OK with [] empty dataset list, when listing datasets (use case for sda-filesystem download tool)
		// 404 dataset not found, when listing files from a dataset
		// 401 unauthorised, when downloading a file
		datasets = auth.GetPermissions(*visas)

		// Start a new session and store datasets under the session key
		key := session.NewSessionKey()
		session.Set(key, datasets)
		c.SetCookie(config.Config.Session.Name, // name
			key, // value
			int(config.Config.Session.Expiration)/1e9, // max age
			"/",                            // path
			config.Config.Session.Domain,   // domain
			config.Config.Session.Secure,   // secure
			config.Config.Session.HTTPOnly, // httpOnly
		)
		log.Debug("authorization check passed")
	}

	return datasets
}
