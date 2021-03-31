package request

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// Do does a GET request as configured by arguments
func Do(url string, headers map[string]string) ([]byte, error) {
	log.Debugf("setting up client for http request to %s", url)
	// Set up client
	agent := fiber.AcquireAgent()
	request := agent.Request()
	request.Header.SetMethod(fiber.MethodGet)
	// Place headers if any exist
	for k, v := range headers {
		agent.Request().Header.Set(k, v)
	}
	// Execute request
	request.SetRequestURI(url)
	if err := agent.Parse(); err != nil {
		log.Errorf("request failed, %s, %s", url, err)
		return nil, err
	}
	// Read response
	code, body, err := agent.Bytes()
	if code != 200 || err != nil {
		log.Errorf("something went wrong, %s, %d, %s, %s", url, code, body, err)
		return nil, errors.New(strconv.Itoa(code))
	}
	log.Debugf("http request was successful, status:%d, body_len:%d, err:%s", code, len(body), err)
	return body, nil
}
