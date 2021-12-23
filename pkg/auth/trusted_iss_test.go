package auth

import (
	"testing"

	"github.com/neicnordic/sda-download/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateTrustedIss(t *testing.T) {

	// this also tests checkIss
	config.Config.OIDC.TrustedISS = "../../dev_utils/iss.json"

	_, ok := ValidateTrustedIss("https://demo.example", "https://mockauth:8000/idp/profile/oidc/keyset")

	assert.True(t, ok, "values might have changed in fixture")

	_, ok = ValidateTrustedIss("https://demo3.example", "https://mockauth:8000/idp/profile/oidc/keyset")

	assert.False(t, ok, "values might have changed in fixture")
}

func TestValidateTrustedIssNoConfig(t *testing.T) {

	// this also tests checkIss
	config.Config.OIDC.TrustedISS = ""

	_, ok := ValidateTrustedIss("https://demo.example", "https://mockauth:8000/idp/profile/oidc/keyset")

	assert.True(t, ok, "this should be true")
}
