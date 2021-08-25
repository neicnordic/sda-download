package main

import (
	"strconv"

	"github.com/neicnordic/sda-download/api"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/pkg/request"
	log "github.com/sirupsen/logrus"
)

func main() {
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

	// app contains the web app and endpoints
	app := api.Setup(conf.OIDC.ConfigurationURL, conf.App.ArchivePath)

	// Start server
	log.Fatal(app.Listen(conf.App.Host + ":" + strconv.Itoa(conf.App.Port)))

}
