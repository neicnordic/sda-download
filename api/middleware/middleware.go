package middleware

import (
	"context"
	"net/http"

	"github.com/neicnordic/sda-download/pkg/auth"
	log "github.com/sirupsen/logrus"
)

// TokenMiddleware performs access token verification and validation
// JWTs are verified and validated by the app, opaque tokens are sent to AAI for verification
// Successful auth results in list of authorised datasets
func TokenMiddleware(nextHandler http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that a token is provided
		token, code, err := auth.GetToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}

		// Verify token by attempting to retrieve visas from AAI
		visas, err := auth.GetVisas(auth.Details, token)
		if err != nil {
			log.Debug("failed to validate token at AAI")
			http.Error(w, "bad token", 401)
			return
		}

		// Get permissions
		datasets, err := auth.GetPermissions(*visas)
		if err != nil {
			log.Errorf("failed to parse dataset permission visas, %s", err)
			http.Error(w, "visa parsing failed", 500)
			return
		}
		if len(datasets) == 0 {
			log.Debug("token carries no dataset permissions matching the database")
			http.Error(w, "no datasets found", 404)
			return
		}

		log.Debug("authorization check passed")

		// Store dataset list to request context, for use in the endpoint handlers
		modifiedContext := storeDatasets(r.Context(), datasets)
		modifiedRequest := r.WithContext(modifiedContext)

		// Forward request to the next endpoint handler
		nextHandler.ServeHTTP(w, modifiedRequest)
	})

}

// storeDatasets stores the dataset list to the request context
func storeDatasets(ctx context.Context, datasets []string) context.Context {
	log.Debugf("storing %v datasets to request context", datasets)
	return context.WithValue(ctx, "datasets", datasets)
}

// GetDatasets extracts the dataset list from the request context
func GetDatasets(ctx context.Context) []string {
	datasets := ctx.Value("datasets")
	if datasets == nil {
		log.Debug("request datasets context is empty")
		return []string{}
	}
	log.Debug("returning %v from request context", datasets)
	return datasets.([]string)
}
