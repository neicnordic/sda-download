package files

import (
	"bytes"
	"io"
	"os"

	"github.com/elixir-oslo/crypt4gh/model/headers"
	"github.com/elixir-oslo/crypt4gh/streaming"
	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"
)

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
