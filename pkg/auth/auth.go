package auth

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/neicnordic/sda-download/internal/database"
	"github.com/neicnordic/sda-download/pkg/request"
	log "github.com/sirupsen/logrus"
)

// Details stores an OIDCDetails struct
var Details OIDCDetails

// OIDCDetails is used to draw the response bytes to a struct
type OIDCDetails struct {
	Issuer   string `json:"issue"`
	Userinfo string `json:"userinfo_endpoint"`
	JWK      string `json:"jwks_uri"`
}

// GetOIDCDetails requests OIDC configuration information
func GetOIDCDetails(url string) OIDCDetails {
	// Prepare response body struct
	var u OIDCDetails
	// Do request
	code, body, err := request.Do(url, nil)
	if code != 200 || err != nil {
		log.Errorf("request failed, %d, %s", code, err)
		return u
	}
	// Parse response
	errj := json.Unmarshal(body, &u)
	if errj != nil {
		log.Errorf("failed to parse JSON response, %s", errj)
	}
	return u
}

// Visas is used to draw the response bytes to a struct
type Visas struct {
	Visa []string `json:"ga4gh_passport_v1"`
}

// VerifyJWT verifies the token signature
func VerifyJWT(o OIDCDetails, token string) (jwt.Token, string) {
	log.Debug("verifying JWT signature")
	// Why do we use context.TODO() ? https://pkg.go.dev/context#TODO
	keyset, err := jwk.Fetch(context.TODO(), o.JWK)
	if err != nil {
		log.Errorf("failed to request JWK set, %s", err)
		return nil, "aai"
	}
	verifiedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(keyset))
	if err != nil {
		log.Errorf("failed to verify token signature, %s", err)
		return nil, "token"
	}
	log.Debug("JWT signature verified")
	return verifiedToken, ""
}

// ValidateJWT validates the token claims
func ValidateJWT(o OIDCDetails, token jwt.Token) bool {
	log.Debug("validating JWT claims")
	// claims validated implicitly: iat, exp, nbf
	// claims validated explicitly: iss
	// if issuer is set, validate issuer (the case for access token)
	if len(o.Issuer) > 0 {
		if err := jwt.Validate(token, jwt.WithIssuer(o.Issuer)); err != nil {
			log.Errorf("failed to validate token claims, %s", err)
			return false
		}
	} else {
		// else, don't validate issuer (the case for visas)
		if err := jwt.Validate(token); err != nil {
			log.Errorf("failed to validate token claims, %s", err)
			return false
		}
	}
	log.Debug("JWT claims validated")
	return true
}

// VerifyOpaque sends the token to AAI introspection for remote validation
func VerifyOpaque(o OIDCDetails, token string) (bool, string) {
	log.Debug("verifying opaque token")
	// Verify opaque token by sending it to token introspection
	headers := map[string]string{}
	headers["Authorization"] = "Bearer " + token
	code, _, err := request.Do(o.Userinfo, headers)
	if code != 200 || err != nil {
		log.Errorf("request failed, %d, %s", code, err)
		return false, "token"
	}
	log.Debug("opaque token verified")
	return true, ""
}

// GetToken parses the token string from header
func GetToken(header string) (string, int) {
	if len(header) == 0 {
		log.Debug("authorization check failed")
		return "access token must be provided", 401
	}

	// Check that Bearer scheme is used
	headerParts := strings.Split(header, " ")
	if headerParts[0] != "Bearer" {
		log.Debug("authorization check failed")
		return "authorization scheme must be bearer", 400
	}

	// Check that header contains a token string
	var token string
	if len(headerParts) == 2 {
		token = headerParts[1]
	} else {
		log.Debug("authorization check failed")
		return "token string is missing from authorization header", 400
	}
	return token, 0
}

type JKU struct {
	URL string `json:"jku"`
}

// Visa is used to draw the dataset name out of the visa
type Visa struct {
	Type    string `json:"type"`
	Dataset string `json:"value"`
}

// getVisas requests the list of visas from userinfo endpoint
func getVisas(url string, token string) Visas {
	// Set headers
	headers := map[string]string{}
	headers["Authorization"] = "Bearer " + token
	// Prepare visas
	var v Visas
	// Do request
	code, body, err := request.Do(url, headers)
	if code != 200 || err != nil {
		log.Errorf("request failed, %d, %s", code, err)
		return v
	}
	// Parse response
	errj := json.Unmarshal(body, &v)
	if errj != nil {
		log.Errorf("failed to parse JSON response, %s", errj)
	}
	return v
}

// GetPermissions parses visas and finds matching dataset names from the database, returning a list of matches
func GetPermissions(token string) []string {

	var datasets []string

	// Get visas
	visas := getVisas(Details.Userinfo, token)

	// Iterate visas
	for _, v := range visas.Visa {

		log.Debug("checking visa type")
		// Check that visa is of type ControlledAccessGrants
		unknownToken, err := jwt.Parse([]byte(v))
		if err != nil {
			log.Errorf("failed to parse visa, %s", err)
			continue
		}
		unknownTokenVisaClaim := unknownToken.PrivateClaims()["ga4gh_visa_v1"]
		unknownTokenVisa := Visa{}
		unknownTokenVisaClaimJSON, err := json.Marshal(unknownTokenVisaClaim)
		if err != nil {
			log.Errorf("failed to parse unknown visa claim to JSON, %s, %s", err, unknownTokenVisaClaim)
			continue
		}
		err = json.Unmarshal(unknownTokenVisaClaimJSON, &unknownTokenVisa)
		if err != nil {
			log.Errorf("failed to parse unknown visa claim JSON into struct, %s, %s", err, unknownTokenVisaClaimJSON)
			continue
		}
		if unknownTokenVisa.Type != "ControlledAccessGrants" {
			log.Debug("visa is not a ControlledAccessGrants, skip")
			continue
		}
		log.Debug("visa type check passed")

		log.Debug("start visa validation")
		// Extract header from header.payload.signature
		header, err := jws.Parse([]byte(v))
		if err != nil {
			log.Errorf("failed to parse visa header, %s", err)
			continue
		}
		// Parse the jku key from header
		o := OIDCDetails{
			JWK: header.Signatures()[0].ProtectedHeaders().JWKSetURL(),
		}
		// Verify visa signature
		verifiedVisa, errorMessage := VerifyJWT(o, v)
		if len(errorMessage) > 0 {
			log.Errorf("failed to validate visa, %s", errorMessage)
			continue
		}
		// Validate visa claims, e.g. expiration
		valid := ValidateJWT(o, verifiedVisa)
		if !valid {
			log.Error("failed to validate visa")
			continue
		}
		log.Debug("visa validated")
		// Parse the dataset name out of the value field
		visaClaim := verifiedVisa.PrivateClaims()["ga4gh_visa_v1"]
		visa := Visa{}
		visaClaimJSON, err := json.Marshal(visaClaim)
		if err != nil {
			log.Errorf("failed to parse visa claim to JSON, %s, %s", err, visaClaim)
			continue
		}
		err = json.Unmarshal(visaClaimJSON, &visa)
		if err != nil {
			log.Errorf("failed to parse visa claim JSON into struct, %s, %s", err, visaClaimJSON)
			continue
		}
		datasetFull := visa.Dataset
		datasetParts := strings.Split(datasetFull, "://")
		datasetName := datasetParts[len(datasetParts)-1]
		exists, err := database.DB.CheckDataset(datasetName)
		if err != nil {
			continue
		}
		if exists {
			// check that dataset name doesn't already exist in return list,
			// we can get duplicates when using multiple AAIs
			duplicate := false
			for i := range datasets {
				if datasets[i] == datasetName {
					duplicate = true
					continue
				}
			}
			if !duplicate {
				datasets = append(datasets, datasetName)
			}
		}
	}

	log.Debugf("matched datasets, %s", datasets)
	return datasets
}
