package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var requiredConfVars = []string{
	"db.host", "db.user", "db.password", "db.database", "c4gh.filepath", "c4gh.passphrase", "oidc.ConfigurationURL",
}

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) SetupTest() {
	viper.Set("db.host", "test")
	viper.Set("db.user", "test")
	viper.Set("db.password", "test")
	viper.Set("db.database", "test")
	viper.Set("c4gh.filepath", "test")
	viper.Set("c4gh.passphrase", "test")
	viper.Set("oidc.ConfigurationURL", "test")
}

func (suite *TestSuite) TearDownTest() {
	viper.Reset()
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (suite *TestSuite) TestConfigFile() {
	viper.Set("configFile", "test")
	config, err := NewConfig()
	assert.Nil(suite.T(), config)
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), "test", viper.ConfigFileUsed())
}

func (suite *TestSuite) TestMissingRequiredConfVar() {
	for _, requiredConfVar := range requiredConfVars {
		requiredConfVarValue := viper.Get(requiredConfVar)
		viper.Set(requiredConfVar, nil)
		expectedError := fmt.Errorf("%s not set", requiredConfVar)
		config, err := NewConfig()
		assert.Nil(suite.T(), config)
		if assert.Error(suite.T(), err) {
			assert.Equal(suite.T(), expectedError, err)
		}
		viper.Set(requiredConfVar, requiredConfVarValue)
	}
}

func (suite *TestSuite) TestAppConfig() {
	// Test fail on key read error
	viper.Set("app.host", "test")
	viper.Set("app.port", 1234)
	viper.Set("app.tlscert", "test")
	viper.Set("app.tlskey", "test")
	viper.Set("app.archivePath", "/test")
	viper.Set("app.logLevel", "debug")

	viper.Set("db.sslmode", "disable")

	_, err := NewConfig()
	assert.Error(suite.T(), err, "Error expected")

	// Test pass on key read
	originalGetC4GHKey := GetC4GHKey
	GetC4GHKey = func() (*[32]byte, error) {
		return nil, nil
	}

	config, err := NewConfig()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test", config.App.Host)
	assert.Equal(suite.T(), 1234, config.App.Port)
	assert.Equal(suite.T(), "test", config.App.TLSCert)
	assert.Equal(suite.T(), "test", config.App.TLSKey)
	assert.Equal(suite.T(), "/test", config.App.ArchivePath)
	assert.Equal(suite.T(), "debug", config.App.LogLevel)
	assert.Nil(suite.T(), config.App.Crypt4GHKey)

	GetC4GHKey = originalGetC4GHKey
}

func (suite *TestSuite) TestSessionConfig() {
	originalGetC4GHKey := GetC4GHKey
	GetC4GHKey = func() (*[32]byte, error) {
		return nil, nil
	}

	viper.Set("session.expiration", 3600)
	viper.Set("session.domain", "test")
	viper.Set("session.secure", false)
	viper.Set("session.httponly", false)

	viper.Set("db.sslmode", "disable")

	config, err := NewConfig()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), time.Duration(3600*time.Second), config.Session.Expiration)
	assert.Equal(suite.T(), "test", config.Session.Domain)
	assert.Equal(suite.T(), false, config.Session.Secure)
	assert.Equal(suite.T(), false, config.Session.HTTPOnly)

	GetC4GHKey = originalGetC4GHKey
}

func (suite *TestSuite) TestDatabaseConfig() {
	originalGetC4GHKey := GetC4GHKey
	GetC4GHKey = func() (*[32]byte, error) {
		return nil, nil
	}

	// Test error on missing SSL vars
	viper.Set("db.sslmode", "verify-full")
	_, err := NewConfig()
	assert.Error(suite.T(), err, "Error expected")

	// Test no error on SSL disabled
	viper.Set("db.sslmode", "disable")
	_, err = NewConfig()
	assert.NoError(suite.T(), err)

	// Test pass on SSL vars set
	viper.Set("db.host", "test")
	viper.Set("db.port", 1234)
	viper.Set("db.user", "test")
	viper.Set("db.password", "test")
	viper.Set("db.database", "test")
	viper.Set("db.cacert", "test")
	viper.Set("db.clientcert", "test")
	viper.Set("db.clientkey", "test")
	viper.Set("db.sslmode", "verify-full")

	config, err := NewConfig()
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test", config.DB.Host)
	assert.Equal(suite.T(), 1234, config.DB.Port)
	assert.Equal(suite.T(), "test", config.DB.User)
	assert.Equal(suite.T(), "test", config.DB.Password)
	assert.Equal(suite.T(), "test", config.DB.Database)
	assert.Equal(suite.T(), "test", config.DB.CACert)
	assert.Equal(suite.T(), "test", config.DB.ClientCert)
	assert.Equal(suite.T(), "test", config.DB.ClientKey)

	GetC4GHKey = originalGetC4GHKey
}
