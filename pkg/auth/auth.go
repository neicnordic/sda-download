package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/lestrrat-go/jwx/jwa"
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
	Userinfo string `json:"userinfo_endpoint"`
	JWK      string `json:"jwks_uri"`
}

// GetOIDCDetails requests OIDC configuration information
func GetOIDCDetails(url string) (OIDCDetails, error) {
	log.Debugf("requesting OIDC config from %s", url)
	// Prepare response body struct
	var u OIDCDetails
	// Do request
	response, err := request.MakeRequest("GET", url, nil, nil)
	if err != nil {
		log.Errorf("request failed, %s", err)
		return u, err
	}
	// Parse response
	err = json.NewDecoder(response.Body).Decode(&u)
	if err != nil {
		log.Errorf("failed to parse JSON response, %s", err)
		return u, err
	}
	log.Debugf("received OIDC config %s from %s", u, url)
	return u, nil
}

// VerifyJWT verifies the token signature
func VerifyJWT(o OIDCDetails, token string) (jwt.Token, error) {
	log.Debug("verifying JWT signature")
	// Why do we use context.TODO() ? https://pkg.go.dev/context#TODO
	keyset, err := jwk.Fetch(context.TODO(), o.JWK)
	if err != nil {
		log.Errorf("failed to request JWK set from %s, %s", o.JWK, err)
		return nil, err
	}
	key, valid := keyset.Get(0)
	if !valid {
		log.Errorf("cannot get key from set , %s", err)
	}

	verifiedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(keyset), jwt.InferAlgorithmFromKey(true))
	if err != nil {

		verifiedToken, err = jwt.Parse([]byte(token), jwt.WithVerify(jwa.RS256, key))
		if err != nil {
			log.Errorf("failed to verify token signature of token %s, %s", token, err)
		}
	}
	log.Debug(verifiedToken)
	log.Debug("JWT signature verified")
	return verifiedToken, nil
}

// GetToken parses the token string from header
var GetToken = func(header string) (string, int, error) {
	log.Debug("parsing access token from header")
	if len(header) == 0 {
		log.Debug("authorization check failed")
		return "", 401, errors.New("access token must be provided")
	}

	// Check that Bearer scheme is used
	headerParts := strings.Split(header, " ")
	if headerParts[0] != "Bearer" {
		log.Debug("authorization check failed")
		return "", 400, errors.New("authorization scheme must be bearer")
	}

	// Check that header contains a token string
	var token string
	if len(headerParts) == 2 {
		token = headerParts[1]
	} else {
		log.Debug("authorization check failed")
		return "", 400, errors.New("token string is missing from authorization header")
	}
	log.Debug("access token found")
	return token, 0, nil
}

type JKU struct {
	URL string `json:"jku"`
}

// Visas is used to draw the response bytes to a struct
type Visas struct {
	Visa []string `json:"ga4gh_passport_v1"`
}

// Visa is used to draw the dataset name out of the visa
type Visa struct {
	Type    string `json:"type"`
	Dataset string `json:"value"`
}

// GetVisas requests the list of visas from userinfo endpoint
var GetVisas = func(o OIDCDetails, token string) (*Visas, error) {
	log.Debugf("requesting visas from %s", o.Userinfo)
	// Set headers
	headers := map[string]string{}
	headers["Authorization"] = "Bearer " + token
	// Do request
	response, err := request.MakeRequest("GET", o.Userinfo, headers, nil)
	if err != nil {
		log.Errorf("request failed, %s", err)
		return nil, err
	}
	// Parse response
	var v Visas
	err = json.NewDecoder(response.Body).Decode(&v)
	if err != nil {
		log.Errorf("failed to parse JSON response, %s", err)
		return nil, err
	}
	log.Debug("visas received")
	return &v, nil
}

// GetPermissions parses visas and finds matching dataset names from the database, returning a list of matches
var GetPermissions = func(visas Visas) []string {
	log.Debug("parsing permissions from visas")
	var datasets []string

	log.Debugf("number of visas to check: %d", len(visas.Visa))

	// Iterate visas
	for _, v := range visas.Visa {

		// Check that visa is of type ControlledAccessGrants
		if checkVisaType(v, "ControlledAccessGrants") {
			// Check that visa is valid and return visa token
			verifiedVisa, valid := validateVisa(v)
			if valid {
				// Parse the dataset name out of the value field
				datasets = getDatasets(verifiedVisa, datasets)
			}
		}

	}

	log.Debugf("matched datasets: %s", datasets)
	return datasets
}

func checkVisaType(visa string, visaType string) bool {

	log.Debug("checking visa type")

	unknownToken, err := jwt.Parse([]byte(visa))
	if err != nil {
		log.Errorf("failed to parse visa, %s", err)
		return false
	}
	unknownTokenVisaClaim := unknownToken.PrivateClaims()["ga4gh_visa_v1"]
	unknownTokenVisa := Visa{}
	unknownTokenVisaClaimJSON, err := json.Marshal(unknownTokenVisaClaim)
	if err != nil {
		log.Errorf("failed to parse unknown visa claim: %s to JSON, with error: %s", unknownTokenVisaClaim, err)
		return false
	}
	err = json.Unmarshal(unknownTokenVisaClaimJSON, &unknownTokenVisa)
	if err != nil {
		log.Errorf("failed to parse unknown visa claim: %s to JSON, with error: %s", unknownTokenVisaClaim, err)
		return false
	}
	if unknownTokenVisa.Type != visaType {
		log.Debugf("visa is not of type: %s, skip", visaType)
		return false
	}
	log.Debug("visa type check passed")

	return true

}

func validateVisa(visa string) (jwt.Token, bool) {
	log.Debug("start visa validation")
	// Extract header from header.payload.signature
	header, err := jws.Parse([]byte(visa))
	if err != nil {
		log.Errorf("failed to parse visa header, %s", err)
		return nil, false
	}
	// Parse the jku key from header
	o := OIDCDetails{
		JWK: header.Signatures()[0].ProtectedHeaders().JWKSetURL(),
	}
	// Verify visa signature
	verifiedVisa, err := VerifyJWT(o, visa)
	if err != nil {
		log.Errorf("failed to validate visa, %s", err)
		return nil, false
	}

	if !ValidateTrustedIss(verifiedVisa.Issuer(), o.JWK) {
		log.Infof("combination of iss: %s and jku: %s is not trusted", verifiedVisa.Issuer(), o.JWK)
		return nil, false
	}

	// Validate visa claims, exp, iat, nbf
	if err := jwt.Validate(verifiedVisa); err != nil {
		log.Error("failed to validate visa")
		return nil, false
	}
	log.Debug("visa validated")

	return verifiedVisa, true
}

func getDatasets(parsedVisa jwt.Token, datasets []string) []string {
	visaClaim := parsedVisa.PrivateClaims()["ga4gh_visa_v1"]
	visa := Visa{}
	visaClaimJSON, err := json.Marshal(visaClaim)
	if err != nil {
		log.Errorf("failed to parse visa claim to JSON, %s, %s", err, visaClaim)
		return datasets
	}
	err = json.Unmarshal(visaClaimJSON, &visa)
	if err != nil {
		log.Errorf("failed to parse visa claim JSON into struct, %s, %s", err, visaClaimJSON)
		return datasets
	}
	exists, err := database.CheckDataset(visa.Dataset)
	if err != nil {
		log.Debugf("visa contained dataset %s which doesn't exist in this instance, skip", visa.Dataset)
		return datasets
	}
	if exists {
		log.Debugf("checking dataset list for duplicates of %s", visa.Dataset)
		// check that dataset name doesn't already exist in return list,
		// we can get duplicates when using multiple AAIs
		for i := range datasets {
			if datasets[i] == visa.Dataset {
				log.Debugf("found a duplicate: dataset %s is already found, skip", visa.Dataset)
				return datasets
			}
		}
		log.Debugf("no duplicates of dataset: %s, add dataset to list of permissions", visa.Dataset)
		datasets = append(datasets, visa.Dataset)
	}

	return datasets
}
