package auth

import (
	"encoding/json"
	"io/ioutil"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"
)

type TrustedISS struct {
	ISS string `json:"iss"`
	JKU string `json:"jku"`
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

// checkISS searches a nested list of TrustedISS
// looking for a map with a specific key value pair for iss.
// If found checkISS returns a jwk.MapWhitelist to be used to validate the visa
func checkISS(obj []TrustedISS, issuerValue string, jkuValue string) (*jwk.MapWhitelist, bool) {
	keys := make(map[string]bool)
	wl := jwk.NewMapWhitelist()
	found := false

	for _, value := range obj {
		if value.ISS == issuerValue && value.JKU == jkuValue {
			if _, ok := keys[value.JKU]; !ok {
				keys[value.JKU] = true
				wl.Add(value.JKU)
				found = true
			}
		}
	}
	return wl, found
}

// ValidateTrustedIss opens the file for the iss, jku combination
// and searches for that combination, only if the file is set.
// If the file is not set it passes silently
func ValidateTrustedIss(iss string, jku string) (*jwk.MapWhitelist, bool) {
	log.Debugf("check combination of iss: %s and jku: %s", iss, jku)
	if config.Config.OIDC.TrustedISS != "" {
		conf, err := readTrustedIssuers(config.Config.OIDC.TrustedISS)
		if err != nil {
			return nil, false
		}
		validList, found := checkISS(conf, iss, jku)
		if found {
			return validList, true
		} else {
			return nil, false
		}
	}
	return nil, true
}
