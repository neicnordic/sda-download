package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/neicnordic/crypt4gh/keys"
	"github.com/neicnordic/sda-download/internal/storage"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const POSIX = "posix"
const S3 = "s3"

// Config is a global configuration value store
var Config ConfigMap

// ConfigMap stores all different configs
type ConfigMap struct {
	App     AppConfig
	Session SessionConfig
	DB      DatabaseConfig
	OIDC    OIDCConfig
	Archive storage.Conf
}

type AppConfig struct {
	// Hostname for this web app
	// Optional. Default value localhost
	Host string

	// Port number for this web app
	// Optional. Default value 8080
	Port int

	// TLS server certificate for HTTPS
	// Optional. Defaults to empty
	ServerCert string

	// TLS server certificate key for HTTPS
	// Optional. Defaults to empty
	ServerKey string

	// Stores the Crypt4GH private key if the two configs above are set
	// Unconfigurable. Depends on Crypt4GHKeyFile and Crypt4GHPassFile
	Crypt4GHKey *[32]byte
}

type SessionConfig struct {
	// Session key expiration time in seconds.
	// Optional. Default value -1
	// Negative values disable the session and requires visa auth to be done on every request.
	// Positive values indicate amount of seconds the session stays active, e.g. 3600 for one hour.
	Expiration time.Duration

	// Cookie domain, this should be the hostname of the server.
	// Optional. Default value empty.
	Domain string

	// Cookie secure value. If true, cookie can only travel in HTTPS.
	// Optional. Default value true
	Secure bool

	// Cookie HTTPOnly value. If true, cookie can't be read by JavaScript.
	// Optional. Default value true
	HTTPOnly bool

	// Name of session cookie.
	// Optional. Default value sda_session_key
	Name string
}

type TrustedISS struct {
	ISS string `json:"iss"`
	JKU string `json:"jku"`
}

type OIDCConfig struct {
	// OIDC OP configuration URL /.well-known/openid-configuration
	// Mandatory.
	ConfigurationURL string
	Whitelist        *jwk.MapWhitelist
	TrustedList      []TrustedISS
	CACert           string
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

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Infoln("No config file found, using ENVs only")
		} else {
			return nil, err
		}
	}
	requiredConfVars := []string{
		"db.host", "db.user", "db.password", "db.database", "c4gh.filepath", "c4gh.passphrase", "oidc.configuration.url",
	}

	if viper.GetString("archive.type") == S3 {
		requiredConfVars = append(requiredConfVars, []string{"archive.url", "archive.accesskey", "archive.secretkey", "archive.bucket"}...)
	} else if viper.GetString("archive.type") == POSIX {
		requiredConfVars = append(requiredConfVars, []string{"archive.location"}...)
	}

	for _, s := range requiredConfVars {
		if !viper.IsSet(s) || viper.GetString(s) == "" {
			return nil, fmt.Errorf("%s not set", s)
		}
	}

	if viper.IsSet("log.format") {
		if viper.GetString("log.format") == "json" {
			log.SetFormatter(&log.JSONFormatter{})
			log.Info("The logs format is set to JSON")
		}
	}

	if viper.IsSet("log.level") {
		stringLevel := viper.GetString("log.level")
		intLevel, err := log.ParseLevel(stringLevel)
		if err != nil {
			log.Printf("Log level '%s' not supported, setting to 'trace'", stringLevel)
			intLevel = log.TraceLevel
		}
		log.SetLevel(intLevel)
		log.Printf("Setting log level to '%s'", stringLevel)
	}

	c := &ConfigMap{}
	c.applyDefaults()
	c.sessionConfig()
	c.configArchive()
	err := c.configureOIDC()
	if err != nil {
		return nil, err
	}
	err = c.appConfig()
	if err != nil {
		return nil, err
	}

	err = c.configDatabase()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// applyDefaults set default values for web server and session
// default to host 0.0.0.0 as it will the main way we deploy this application
func (c *ConfigMap) applyDefaults() {
	viper.SetDefault("app.host", "0.0.0.0")
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("session.expiration", -1)
	viper.SetDefault("session.secure", true)
	viper.SetDefault("session.httponly", true)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("session.name", "sda_session_key")
}

// configS3Storage populates and returns a S3Conf from the
// configuration
func configS3Storage(prefix string) storage.S3Conf {
	s3 := storage.S3Conf{}
	// All these are required
	s3.URL = viper.GetString(prefix + ".url")
	s3.AccessKey = viper.GetString(prefix + ".accesskey")
	s3.SecretKey = viper.GetString(prefix + ".secretkey")
	s3.Bucket = viper.GetString(prefix + ".bucket")

	// Defaults (move to viper?)

	s3.Port = 443
	s3.Region = "us-east-1"
	s3.NonExistRetryTime = 2 * time.Minute

	if viper.IsSet(prefix + ".port") {
		s3.Port = viper.GetInt(prefix + ".port")
	}

	if viper.IsSet(prefix + ".region") {
		s3.Region = viper.GetString(prefix + ".region")
	}

	if viper.IsSet(prefix + ".chunksize") {
		s3.Chunksize = viper.GetInt(prefix+".chunksize") * 1024 * 1024
	}

	if viper.IsSet(prefix + ".cacert") {
		s3.Cacert = viper.GetString(prefix + ".cacert")
	}

	return s3
}

func (c *ConfigMap) configureOIDC() error {
	c.OIDC.ConfigurationURL = viper.GetString("oidc.configuration.url")
	c.OIDC.Whitelist = nil
	c.OIDC.TrustedList = nil
	if viper.IsSet("oidc.trusted.iss") {
		obj, err := readTrustedIssuers(viper.GetString("oidc.trusted.iss"))
		if err != nil {
			return err
		}
		c.OIDC.Whitelist = constructWhitelist(obj)
		c.OIDC.TrustedList = obj
	}
	if viper.IsSet("oidc.cacert") {
		c.OIDC.CACert = viper.GetString("oidc.cacert")
	}

	return nil
}

// configArchive provides configuration for the archive storage
// we default to POSIX unless S3 specified
func (c *ConfigMap) configArchive() {
	if viper.GetString("archive.type") == S3 {
		c.Archive.Type = S3
		c.Archive.S3 = configS3Storage("archive")
	} else {
		c.Archive.Type = POSIX
		c.Archive.Posix.Location = viper.GetString("archive.location")
	}
}

// appConfig sets required settings
func (c *ConfigMap) appConfig() error {
	c.App.Host = viper.GetString("app.host")
	c.App.Port = viper.GetInt("app.port")
	c.App.ServerCert = viper.GetString("app.servercert")
	c.App.ServerKey = viper.GetString("app.serverkey")

	if c.App.Port != 443 && c.App.Port != 8080 {
		c.App.Port = viper.GetInt("app.port")
	} else {
		if c.App.ServerCert != "" && c.App.ServerKey != "" {
			c.App.Port = 443
		}

	}

	var err error
	c.App.Crypt4GHKey, err = GetC4GHKey()
	if err != nil {
		return err
	}
	return nil
}

// sessionConfig controls cookie settings and session cache
func (c *ConfigMap) sessionConfig() {
	c.Session.Expiration = time.Duration(viper.GetInt("session.expiration")) * time.Second
	c.Session.Domain = viper.GetString("session.domain")
	c.Session.Secure = viper.GetBool("session.secure")
	c.Session.HTTPOnly = viper.GetBool("session.httponly")
	c.Session.Name = viper.GetString("session.name")
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

// readTrustedIssuers reads information about trusted iss: jku keypair
// the data can be changed in the deployment by configuring OIDC_TRUSTED_ISS env var
func readTrustedIssuers(filePath string) ([]TrustedISS, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Errorf("Error when opening file with issuers, reason: %v", err)
		return nil, err
	}

	// Now let's unmarshall the data into `payload`
	var payload []TrustedISS
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Errorf("Error during Unmarshal, reason: %v", err)
		return nil, err
	}

	return payload, nil
}

func constructWhitelist(obj []TrustedISS) *jwk.MapWhitelist {
	keys := make(map[string]bool)
	wl := jwk.NewMapWhitelist()

	for _, value := range obj {
		if _, ok := keys[value.JKU]; !ok {
			keys[value.JKU] = true
			wl.Add(value.JKU)
		}
	}
	return wl
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
