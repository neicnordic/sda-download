package main

import (
	"context"
	"flag"
	"log"
	"time"

	re "github.com/neicnordic/sda-download/internal/reencrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr      = flag.String("addr", "localhost:5051", "the address to connect to")
	publickey = flag.String("publickey", "NZfoJzFcOli3UWi/7U624h6fv2PufL1i2QPK8JkpmFg=", "Name to greet")
	fileid    = flag.String("fileid", "urn:neic:001-002", "Name to greet")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := re.NewReencryptClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ReencryptHeader(ctx, &re.ReencryptRequest{Fileid: *fileid, Publickey: *publickey})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", string(r.GetHeader()))
}
