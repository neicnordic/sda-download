package main

import (
	"github.com/neicnordic/sda-download/api"
	"github.com/neicnordic/sda-download/api/sda"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/internal/session"
	"github.com/neicnordic/sda-download/internal/storage"
	"github.com/neicnordic/sda-download/pkg/auth"
	"github.com/neicnordic/sda-download/pkg/request"
	log "github.com/sirupsen/logrus"
)

// init is run before main, it sets up configuration and other required things
func init() {
	log.Info("(1/5) Loading configuration")

	// Load configuration
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatalf("configuration loading failed, reason: %v", err)
		panic(err)
	}
	config.Config = *conf

	// Connect to database
	db, err := database.NewDB(conf.DB)
	if err != nil {
		log.Fatalf("database connection failed, reason: %v", err)
		panic(err)
	}
	defer db.Close()
	database.DB = db

	// Initialise HTTP client for making requests
	client, err := request.InitialiseClient()
	if err != nil {
		log.Fatalf("http client init failed, reason: %v", err)
		panic(err)
	}
	request.Client = client

	// Initialise OIDC configuration
	details, err := auth.GetOIDCDetails(conf.OIDC.ConfigurationURL)
	log.Info("retrieving OIDC configuration")
	if err != nil {
		log.Fatalf("oidc init failed, reason: %v", err)
		panic(err)
	}
	auth.Details = details
	log.Info("OIDC configuration retrieved")

	// Initialise session cache
	sessionCache, err := session.InitialiseSessionCache()
	if err != nil {
		log.Fatalf("session cache init failed, reason: %v", err)
	}
	session.SessionCache = sessionCache

	backend, err := storage.NewBackend(conf.Archive)
	if err != nil {
		log.Fatalf("Error initiating storage backend, reason: %v", err)
	}
	sda.Backend = backend
}

// main starts the web server
func main() {
	srv := api.Setup()

	// Start the server
	log.Info("(5/5) Starting web server")
	if config.Config.App.ServerCert != "" && config.Config.App.ServerKey != "" {
		log.Infof("Web server is ready to receive connections at https://%s:%d", config.Config.App.Host, config.Config.App.Port)
		log.Fatal(srv.ListenAndServeTLS(config.Config.App.ServerCert, config.Config.App.ServerKey))
	} else {
		log.Infof("Web server is ready to receive connections at http://%s:%d", config.Config.App.Host, config.Config.App.Port)
		log.Fatal(srv.ListenAndServe())
	}
}
