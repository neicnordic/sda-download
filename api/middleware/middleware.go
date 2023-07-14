package middleware

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// CustomMiddleware is an interface to a middleware
// which is used to get a user's dataset permissions
type CustomMiddleware interface {
	GetDatasets(c *gin.Context) []string
}

var CustomLoadedMiddleware CustomMiddleware
var datasetsKey = "datasets"

// Middleware performs access token verification and validation
// JWTs are verified and validated by the app, opaque tokens are sent to AAI for verification
// Successful auth results in list of authorised datasets
func Middleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		datasets := CustomLoadedMiddleware.GetDatasets(c)

		// Store dataset list to request context, for use in the endpoint handlers
		c = storeDatasets(c, datasets)

		// Forward request to the next endpoint handler
		c.Next()
	}

}

// storeDatasets stores the dataset list to the request context
func storeDatasets(c *gin.Context, datasets []string) *gin.Context {
	log.Debugf("storing %v datasets to request context", datasets)

	c.Set(datasetsKey, datasets)

	return c
}

// GetDatasets extracts the dataset list from the request context
var GetDatasets = func(c *gin.Context) []string {
	datasets := c.GetStringSlice(datasetsKey)

	log.Debugf("returning %v from request context", datasets)

	return datasets
}
