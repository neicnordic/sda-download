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
	Issuer   string `json:"issue"`
	Userinfo string `json:"userinfo_endpoint"`
	JWK      string `json:"jwks_uri"`
}

// GetOIDCDetails requests OIDC configuration information
func GetOIDCDetails(url string) (OIDCDetails, error) {
	log.Debugf("requesting OIDC config from %s", url)
	// Prepare response body struct
	var u OIDCDetails
	// Do request
	body, err := request.Do(url, nil)
	if err != nil {
		log.Errorf("request failed, %s", err)
		return u, err
	}
	// Parse response
	err = json.Unmarshal(body, &u)
	if err != nil {
		log.Errorf("failed to parse JSON response, %s", err)
		return u, err
	}
	log.Debugf("received OIDC config %s from %s", u, url)
	return u, nil
}

// Visas is used to draw the response bytes to a struct
type Visas struct {
	Visa []string `json:"ga4gh_passport_v1"`
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
	if valid != true {
		log.Errorf("cannot get key from set , %s", err)
	}

	log.Info(key)

	verifiedToken, err := jwt.Parse([]byte(token), jwt.WithKeySet(keyset), jwt.UseDefaultKey(true))
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
func GetToken(header string) (string, int, error) {
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

// Visa is used to draw the dataset name out of the visa
type Visa struct {
	Type    string `json:"type"`
	Dataset string `json:"value"`
}

// GetVisas requests the list of visas from userinfo endpoint
func GetVisas(o OIDCDetails, token string) ([]byte, error) {
	log.Debugf("requesting visas from %s", o.Userinfo)
	// Set headers
	headers := map[string]string{}
	headers["Authorization"] = "Bearer " + token
	// Do request
	body, err := request.Do(o.Userinfo, headers)
	if err != nil {
		log.Errorf("request failed, %s", err)
		return []byte{}, err
	}
	log.Debug("visas received")
	return body, nil
}

// GetPermissions parses visas and finds matching dataset names from the database, returning a list of matches
func GetPermissions(visas []byte) ([]string, error) {
	log.Debug("parsing permissions from visas")
	var datasets []string

	// Parse visas bytes to struct
	var visaArray Visas
	err := json.Unmarshal(visas, &visaArray)
	if err != nil {
		log.Errorf("failed to parse JSON response, %s", err)
		return datasets, err
	}
	log.Debugf("number of visas to check: %d", len(visaArray.Visa))

	// Iterate visas
	for _, v := range visaArray.Visa {

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
		verifiedVisa, err := VerifyJWT(o, v)
		if err != nil {
			log.Errorf("failed to validate visa, %s", err)
			continue
		}
		// Validate visa claims, exp, iat, nbf
		if err := jwt.Validate(verifiedVisa); err != nil {
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
		exists, err := database.DB.CheckDataset(datasetFull)
		if err != nil {
			log.Debugf("visa contained dataset %s which doesn't exist in this instance, skip", datasetName)
			continue
		}
		if exists {
			log.Debugf("checking dataset list for duplicates of %s", datasetName)
			// check that dataset name doesn't already exist in return list,
			// we can get duplicates when using multiple AAIs
			duplicate := false
			for i := range datasets {
				if datasets[i] == datasetName {
					duplicate = true
					log.Debugf("found a duplicate: dataset %s is already found, skip", datasetName)
					continue
				}
			}
			if !duplicate {
				log.Debugf("no duplicates of dataset: %s, add dataset to list of permissions", datasetName)
				datasets = append(datasets, datasetName)
			}
		}
	}

	log.Debugf("matched datasets, %s", datasets)
	return datasets, nil
}
