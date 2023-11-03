package grpc

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/neicnordic/sda-download/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	re "github.com/neicnordic/sda-download/internal/reencrypt"
)

// GetNewHeader
func GetNewHeader(oldHeader []byte, publicKey string) ([]byte, error) {

	// Set up a connection to the server.
	log.Debug("Connect to grpc Server to get new header")
	var opts []grpc.DialOption
	if config.Config.Grpc.CACert != "" && config.Config.Grpc.ServerNameOverride != "" {
		creds, err := credentials.NewClientTLSFromFile(config.Config.Grpc.CACert, config.Config.Grpc.ServerNameOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	log.Debugf("grpc connection to: %s:%d", config.Config.Grpc.Host, config.Config.Grpc.Port)
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", config.Config.Grpc.Host, config.Config.Grpc.Port), opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := re.NewReencryptClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	r, err := c.ReencryptHeader(ctx, &re.ReencryptRequest{Oldheader: oldHeader, Publickey: publicKey})
	if err != nil {
		log.Errorf("could not get the : %v", err)

		return nil, err
	}
	log.Debugf("Reencrypted Header: %s", string(r.GetHeader()))

	return r.GetHeader(), nil
}
