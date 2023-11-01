package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/neicnordic/crypt4gh/keys"
	"github.com/neicnordic/crypt4gh/model/headers"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/neicnordic/sda-download/internal/config"
	"github.com/neicnordic/sda-download/internal/database"
	re "github.com/neicnordic/sda-download/internal/reencrypt"
	"google.golang.org/grpc"
)

// server is used to implement reencrypt.ReEncryptServer.
type server struct {
	re.UnimplementedReencryptServer
}

// init is run before main, it sets up configuration and other required things
func init() {
	log.Info("(1/5) Loading configuration")

	// Load configuration
	conf, err := config.NewConfig("reencrypt")
	if err != nil {
		log.Panicf("configuration loading failed, reason: %v", err)
	}
	config.Config = *conf

	// Connect to database
	db, err := database.NewDB(conf.DB)
	if err != nil {
		log.Panicf("database connection failed, reason: %v", err)
	}
	defer db.Close()
	database.DB = db

}

// Reencrypt implements reencrypt.ReEncryptServer
func (s *server) ReencryptHeader(ctx context.Context, in *re.ReencryptRequest) (*re.ReencryptResponse, error) {
	log.Debugf("Received Public key: %v", in.GetPublickey())
	log.Debugf("Received fileid: %v", in.GetFileid())
	// Get file header
	fileDetails, err := database.GetFile(in.GetFileid())
	if err != nil {

		return nil, err
	}

	newReaderPublicKey, err := keys.ReadPublicKey(strings.NewReader("-----BEGIN CRYPT4GH PUBLIC KEY-----\n" + in.GetPublickey() + "\n-----END CRYPT4GH PUBLIC KEY-----\n"))
	if err != nil {
		return nil, err
	}

	newReaderPublicKeyList := [][chacha20poly1305.KeySize]byte{}
	newReaderPublicKeyList = append(newReaderPublicKeyList, newReaderPublicKey)

	log.Debugf("header: %v", fileDetails.Header)
	log.Debugf("crypt4ghkey path: %v", *config.Config.Grpc.Crypt4GHKey)

	newheader, err := headers.ReEncryptHeader(fileDetails.Header, *config.Config.Grpc.Crypt4GHKey, newReaderPublicKeyList)
	if err != nil {
		return nil, err
	}

	return &re.ReencryptResponse{Header: newheader}, nil
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *&config.Config.Grpc.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	re.RegisterReencryptServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
