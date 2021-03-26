package files

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/elixir-oslo/crypt4gh/keys"
	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/elixir-oslo/crypt4gh/streaming"
	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"
)

// GetC4GHKey reads and decrypts and returns the c4gh key
func GetC4GHKey() (*[32]byte, error) {
	if len(config.Config.App.Crypt4GHKey) > 0 && len(config.Config.App.Crypt4GHPassFile) > 0 {
		log.Info("reading crypt4gh private key")

		// Read private key file
		keyFile, err := os.Open(config.Config.App.Crypt4GHKeyFile)
		if err != nil {
			log.Errorf("failed to open crypt4gh private key file, %s, %s", err, config.Config.App.Crypt4GHKey)
			return nil, err
		}

		// Read password file
		password, err := ioutil.ReadFile(config.Config.App.Crypt4GHPassFile)
		if err != nil {
			log.Errorf("failed to read crypt4gh private key password, %s, %s", err, config.Config.App.Crypt4GHPassFile)
			return nil, err
		}

		// Decrypt private key
		key, err := keys.ReadPrivateKey(keyFile, password)
		if err != nil {
			log.Errorf("failed to decrypt crypt4gh private key, %s", err)
			return nil, err
		}

		keyFile.Close()
		log.Info("crypt4gh private key loaded")
		return &key, nil
	} else {
		log.Error("NOT IMPLEMENTED")
		log.Info("crypt4gh private key not configured, re-encryption microservice required")
		return nil, nil
	}
}

// StreamFile returns a stream of file contents
func StreamFile(header []byte, file *os.File, coordinates *headers.DataEditListHeaderPacket) (*streaming.Crypt4GHReader, error) {
	log.Debugf("preparing file %s for streaming", file.Name())
	// Stitch header and file body together
	hr := bytes.NewReader(header)
	mr := io.MultiReader(hr, file)
	c4ghr, err := streaming.NewCrypt4GHReader(mr, *config.Config.App.Crypt4GHKey, coordinates)
	if err != nil {
		log.Errorf("failed to create Crypt4GH stream reader, %s", err)
	}
	log.Debugf("file stream for %s constructed", file.Name())
	return c4ghr, err
}
