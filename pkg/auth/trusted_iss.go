package auth

import (
	"encoding/json"
	"io/ioutil"

	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"
)

// readTrustedIssuers reads information about trusted iss: jku keypair
// the data can be changed in the deployment by configuring OIDC_TRUSTED_ISS env var
func readTrustedIssuers(filePath string) (interface{}, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Errorf("Error when opening file with issuers, reason: %v", err)
		return nil, err
	}

	// Now let's unmarshall the data into `payload`
	var payload interface{}
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Errorf("Error during Unmarshal, reason: %v", err)
		return nil, err
	}

	return payload, nil
}

// checkISS searches a nested structure consisting of map[string]interface{}
// and []interface{} looking for a map with a specific key value pair for iss.
// If found searchNested checks for the corresponding the value associated with jku
// and returns nil, true
// If the key is not found checkISS returns nil, false
func checkISS(obj interface{}, issuerValue string, jkuValue string) (interface{}, bool) {
	switch t := obj.(type) {
	case map[string]interface{}:
		// Return the value if the map contains the key
		if iss, ok := t["iss"]; ok {
			if jku, ok := t["jku"]; ok && iss == issuerValue {
				if jku == jkuValue {
					return nil, ok
				}
			}
		}
		for _, v := range t {
			if result, ok := checkISS(v, issuerValue, jkuValue); ok {
				return result, ok
			}
		}
	case []interface{}:
		for _, v := range t {
			if result, ok := checkISS(v, issuerValue, jkuValue); ok {
				return result, ok
			}
		}
	}
	// key not found
	return nil, false
}

// ValidateTrustedIss opens the file for the iss, jku combination
// and searches for that combination, only if the file is set.
// If the file is not set it passes silently
func ValidateTrustedIss(iss string, jku string) bool {
	log.Debugf("check combination of iss: %s and jku: %s", iss, jku)
	if config.Config.OIDC.TrustedISS != "" {
		conf, err := readTrustedIssuers(config.Config.OIDC.TrustedISS)
		if err != nil {
			return false
		}
		_, valid := checkISS(conf, iss, jku)
		return valid
	}
	return true
}
