package api

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neicnordic/sda-download/api/middleware"
	"github.com/neicnordic/sda-download/api/sda"
	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"
)

// healthResponse
func healthResponse(c *gin.Context) {
	// ok response to health
	c.Writer.WriteHeader(http.StatusOK)
}

// Setup configures the web server and registers the routes
func Setup() *http.Server {
	// Set up routing
	log.Info("(2/5) Registering endpoint handlers")

	router := gin.Default()

	router.GET("/metadata/datasets", middleware.TokenMiddleware(), sda.Datasets)
	router.GET("/metadata/datasets/*dataset", middleware.TokenMiddleware(), sda.Files)
	router.GET("/files/:fileid", middleware.TokenMiddleware(), sda.Download)
	router.GET("/s3/*dataset", middleware.TokenMiddleware(), sda.S3Download)
	router.HEAD("/s3/*dataset", middleware.TokenMiddleware(), sda.S3Download)
	router.GET("/health", healthResponse)
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	// explicitly return 405 on POST requests
	router.POST("/metadata/datasets", func(c *gin.Context) { c.String(http.StatusMethodNotAllowed, "Method Not Allowed") })

	// Configure TLS settings
	log.Info("(3/5) Configuring TLS")
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}

	// Configure web server
	log.Info("(4/5) Configuring server")
	srv := &http.Server{
		Addr:              config.Config.App.Host + ":" + fmt.Sprint(config.Config.App.Port),
		Handler:           router,
		TLSConfig:         cfg,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      20 * time.Second,
	}

	return srv
}
