package main

import (
	"github.com/neicnordic/sda-download/api"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/internal/files"
	"github.com/neicnordic/sda-download/internal/logging"
	log "github.com/sirupsen/logrus"
)

// init is run before main, it loads the configuration variables
func init() {
	c := &config.Config
	config.LoadConfig(c)
	logging.LoggingSetup(c.App.LogLevel)
	var err error
	config.Config.App.Crypt4GHKey, err = files.GetC4GHKey()
	if err != nil {
		log.Errorf("failed to load Crypt4GH private key, re-encryption microservice required, %s", err)
	}
}

func main() {
	// Connect to database
	db, err := database.NewDB(config.Config.DB)
	if err != nil {
		log.Errorf("database connection to %s failed, %s", config.Config.DB.Host, err)
		panic(err)
	}
	defer db.Close()
	database.DB = db

	// app contains the web app and endpoints
	app := api.Setup()

	// Start server
	log.Fatal(app.Listen(config.Config.App.Host + ":" + config.Config.App.Port))

}
