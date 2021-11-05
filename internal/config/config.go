package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/elixir-oslo/crypt4gh/keys"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config is a global configuration value store
var Config ConfigMap

// ConfigMap stores all different configs
type ConfigMap struct {
	App     AppConfig
	Session SessionConfig
	DB      DatabaseConfig
	OIDC    OIDCConfig
}

type AppConfig struct {
	// Hostname for this web app
	// Optional. Default value localhost
	Host string

	// Port number for this web app
	// Optional. Default value 8080
	Port int

	// Logging level
	// Optional. Default value debug
	// Possible values error, fatal, info, panic, warn, trace, debug
	LogLevel string

	// TLS server certificate for HTTPS
	// Optional. Defaults to empty
	TLSCert string

	// TLS server certificate key for HTTPS
	// Optional. Defaults to empty
	TLSKey string

	// Stores the Crypt4GH private key if the two configs above are set
	// Unconfigurable. Depends on Crypt4GHKeyFile and Crypt4GHPassFile
	Crypt4GHKey *[32]byte

	// Path to POSIX Archive, prepended to database file name
	// Optional.
	ArchivePath string
}

type SessionConfig struct {
	// Session key expiration time.
	// Optional. Default value 28800 seconds for 8 hours.
	// -1 for disabling sessions and requiring visa-checks on every request.
	Expiration time.Duration
}

type OIDCConfig struct {
	// OIDC OP configuration URL /.well-known/openid-configuration
	// Mandatory.
	ConfigurationURL string
}

type DatabaseConfig struct {
	// Database hostname
	// Optional. Default value localhost
	Host string

	// Database port number
	// Optional. Default value 5432
	Port int

	// Database username
	// Optional. Default value lega_out
	User string

	// Database password for username
	// Optional. Default value lega_out
	Password string

	// Database name
	// Optional. Default value lega
	Database string

	// TLS CA cert for database connection
	// Optional.
	CACert string

	// CA cert for database connection
	// Optional.
	SslMode string

	// TLS Certificate for database connection
	// Optional.
	ClientCert string

	// TLS Key for database connection
	// Optional.
	ClientKey string
}

// NewConfig populates ConfigMap with data
func NewConfig() (*ConfigMap, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigType("yaml")
	if viper.IsSet("configPath") {
		configPath := viper.GetString("configPath")
		splitPath := strings.Split(strings.TrimLeft(configPath, "/"), "/")
		viper.AddConfigPath(path.Join(splitPath...))
	}

	if viper.IsSet("configFile") {
		viper.SetConfigFile(viper.GetString("configFile"))
	}

	// defaults
	viper.SetDefault("app.host", "localhost")
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.LogLevel", "info")
	viper.SetDefault("app.archivePath", "/")
	viper.SetDefault("session.expiration", 28800*time.Second)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Infoln("No config file found, using ENVs only")
		} else {
			return nil, err
		}
	}
	requiredConfVars := []string{
		"db.host", "db.user", "db.password", "db.database", "c4gh.filepath", "c4gh.passphrase", "oidc.ConfigurationURL",
	}

	for _, s := range requiredConfVars {
		if !viper.IsSet(s) || viper.GetString(s) == "" {
			return nil, fmt.Errorf("%s not set", s)
		}
	}

	if viper.IsSet("app.LogLevel") {
		stringLevel := viper.GetString("app.LogLevel")
		intLevel, err := log.ParseLevel(stringLevel)
		if err != nil {
			log.Printf("Log level '%s' not supported, setting to 'trace'", stringLevel)
			intLevel = log.TraceLevel
		}
		log.SetLevel(intLevel)
		log.Printf("Setting log level to '%s'", stringLevel)
	}

	c := &ConfigMap{}
	c.sessionConfig()
	c.OIDC.ConfigurationURL = viper.GetString("oidc.ConfigurationURL")
	err := c.appConfig()
	if err != nil {
		return nil, err
	}

	err = c.configDatabase()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// appConfig sets required settings
func (c *ConfigMap) appConfig() error {
	c.App.Host = viper.GetString("app.host")
	c.App.Port = viper.GetInt("app.port")
	c.App.TLSCert = viper.GetString("app.tlscert")
	c.App.TLSKey = viper.GetString("app.tlskey")
	c.App.ArchivePath = viper.GetString("app.archivePath")

	var err error
	c.App.Crypt4GHKey, err = GetC4GHKey()
	if err != nil {
		return err
	}
	return nil
}

func (c *ConfigMap) sessionConfig() {
	c.Session.Expiration = time.Duration(viper.GetInt("session.expiration")) * time.Second
}

// configDatabase provides configuration for the database
func (c *ConfigMap) configDatabase() error {
	db := DatabaseConfig{}

	// defaults
	viper.SetDefault("db.port", 5432)
	viper.SetDefault("db.sslmode", "verify-full")

	// All these are required
	db.Port = viper.GetInt("db.port")
	db.Host = viper.GetString("db.host")
	db.User = viper.GetString("db.user")
	db.Password = viper.GetString("db.password")
	db.Database = viper.GetString("db.database")

	// Optional settings
	if viper.IsSet("db.port") {
		db.Port = viper.GetInt("db.port")
	}
	if viper.IsSet("db.sslmode") {
		db.SslMode = viper.GetString("db.sslmode")
	}
	if db.SslMode == "verify-full" {
		// Since verify-full is specified, these are required.
		if !(viper.IsSet("db.clientCert") && viper.IsSet("db.clientKey")) {
			return errors.New("when db.sslMode is set to verify-full both db.clientCert and db.clientKey are needed")
		}
	}
	if viper.IsSet("db.clientKey") {
		db.ClientKey = viper.GetString("db.clientKey")
	}
	if viper.IsSet("db.clientCert") {
		db.ClientCert = viper.GetString("db.clientCert")
	}
	if viper.IsSet("db.cacert") {
		db.CACert = viper.GetString("db.cacert")
	}
	c.DB = db
	return nil
}

// GetC4GHKey reads and decrypts and returns the c4gh key
func GetC4GHKey() (*[32]byte, error) {
	log.Info("reading crypt4gh private key")
	keyPath := viper.GetString("c4gh.filepath")
	passphrase := viper.GetString("c4gh.passphrase")

	// Make sure the key path and passphrase is valid
	keyFile, err := os.Open(keyPath)
	if err != nil {
		return nil, err
	}

	key, err := keys.ReadPrivateKey(keyFile, []byte(passphrase))
	if err != nil {
		return nil, err
	}

	keyFile.Close()
	log.Info("crypt4gh private key loaded")
	return &key, nil
}
