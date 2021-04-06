package main

import (
	"strconv"

	"github.com/neicnordic/sda-download/api"
	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	log "github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	// Connect to database
	db, err := database.NewDB(conf.DB)
	if err != nil {
		log.Errorf("database connection to %s failed, %s", conf.DB.Host, err)
		panic(err)
	}
	defer db.Close()
	database.DB = db
	// app contains the web app and endpoints
	app := api.Setup(conf.OIDC.ConfigurationURL)

	// Start server
	log.Fatal(app.Listen(conf.App.Host + ":" + strconv.Itoa(conf.App.Port)))

}
