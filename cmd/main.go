package main

import (
	"github.com/neicnordic/sda-download/api"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/pkg/request"
	log "github.com/sirupsen/logrus"
)

// init is run before main, it sets up configuration and other required things
func init() {
	log.Info("(1/5) Loading configuration")

	// Load configuration
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	config.Config = *conf

	// Connect to database
	db, err := database.NewDB(conf.DB)
	if err != nil {
		log.Errorf("database connection to %s failed, %s", conf.DB.Host, err)
		panic(err)
	}
	defer db.Close()
	database.DB = db

	// Initialise HTTP client for making requests
	client, err := request.InitialiseClient()
	if err != nil {
		panic(err)
	}
	request.Client = client
}

// main starts the web server
func main() {
	srv := api.Setup()

	// Start the server
	log.Info("(5/5) Starting web server")
	if config.Config.App.TLSCert != "" && config.Config.App.TLSKey != "" {
		log.Infof("Web server is ready to receive connections at https://%s:%d", config.Config.App.Host, config.Config.App.Port)
		log.Fatal(srv.ListenAndServeTLS(config.Config.App.TLSCert, config.Config.App.TLSKey))
	} else {
		log.Infof("Web server is ready to receive connections at http://%s:%d", config.Config.App.Host, config.Config.App.Port)
		log.Fatal(srv.ListenAndServe())
	}
}
