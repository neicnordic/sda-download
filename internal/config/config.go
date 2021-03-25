package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config is a global configuration value store
var Config ConfigMap

// ConfigMap stores all different configs
type ConfigMap struct {
	App  AppConfig
	DB   DatabaseConfig
	OIDC OIDCConfig
}

type AppConfig struct {
	// Hostname for this web app
	// Optional. Default value localhost
	Host string

	// Port number for this web app
	// Optional. Default value 8080
	Port string

	// Logging level
	// Optional. Default value debug
	// Possible values error, fatal, info, panic, warn, trace, debug
	LogLevel string

	// Path to Crypt4GH private key for file decryption
	// Optional.
	// TO DO: If left empty, another re-encryption service needs to be called to serve files
	Crypt4GHKeyFile string

	// Path to Crypt4GH private key password file
	// Optional.
	// Required to open Crypt4GHKey file
	Crypt4GHPassFile string

	// Stores the Crypt4GH private key if the two configs above are set
	// Unconfigurable. Depends on Crypt4GHKeyFile and Crypt4GHPassFile
	Crypt4GHKey *[32]byte
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

// GetEnv returns given os.Getenv value, or a default value if os.Getenv is empty
func GetEnv(key string, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

// LoadConfig populates ConfigMap with data
func LoadConfig(c *ConfigMap) {
	// Load settings from .env
	godotenv.Load(".env")
	// Populate config structs, place defaults if empty in .env
	c.App.Host = GetEnv("APP_HOST", "localhost")
	c.App.Port = GetEnv("APP_PORT", "8080")
	c.App.LogLevel = GetEnv("APP_LOG_LEVEL", "debug")
	c.App.Crypt4GHKeyFile = GetEnv("APP_CRYPT4GH_KEY", "")
	c.App.Crypt4GHPassFile = GetEnv("APP_CRYPT4GH_PASS", "")
	c.OIDC.ConfigurationURL = GetEnv("OIDC_CONFIGURATION_URL", "")
	c.DB.Host = GetEnv("DB_HOST", "localhost")
	c.DB.Port, _ = strconv.Atoi(GetEnv("DB_PORT", "5432"))
	c.DB.User = GetEnv("DB_USER", "lega_out")
	c.DB.Password = GetEnv("DB_PASS", "lega_out")
	c.DB.Database = GetEnv("DB_NAME", "lega")
	c.DB.SslMode = GetEnv("DB_SSL_MODE", "")
	c.DB.CACert = GetEnv("DB_CA_CERT", "")
	c.DB.ClientCert = GetEnv("DB_CLIENT_CERT", "")
	c.DB.ClientKey = GetEnv("DB_CLIENT_KEY", "")
}
