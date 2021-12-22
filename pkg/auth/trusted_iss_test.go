package auth

import (
	"testing"

	"github.com/neicnordic/sda-download/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateTrustedIss(t *testing.T) {

	// this also tests checkIss
	config.Config.OIDC.TrustedISS = "../../dev_utils/iss.json"

	actual := ValidateTrustedIss("http://demo.example", "http://mockauth:8000/idp/profile/oidc/keyset")

	assert.True(t, actual, "values might have changed in fixture")

	actual = ValidateTrustedIss("http://demo3.example", "http://mockauth:8000/idp/profile/oidc/keyset")

	assert.False(t, actual, "values might have changed in fixture")
}

func TestValidateTrustedIssNoConfig(t *testing.T) {

	// this also tests checkIss
	config.Config.OIDC.TrustedISS = ""

	actual := ValidateTrustedIss("http://demo.example", "http://mockauth:8000/idp/profile/oidc/keyset")

	assert.True(t, actual, "this should be true")
}
