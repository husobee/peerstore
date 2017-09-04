package main

import (
	"log"

	"github.com/husobee/peerstore/protocol"
)

func main() {
	log.Println("client")
	t, err := protocol.NewTransport("tcp", "localhost:3000")
	if err != nil {
		log.Printf("ERR: %v", err)
	}

	for _, method := range []protocol.RequestMethod{
		protocol.PostFileMethod,
		protocol.GetFileMethod,
		protocol.DeleteFileMethod,
		42,
	} {
		log.Println("starting request: ", method)
		response, err := t.RoundTrip(&protocol.Request{
			Header: protocol.Header{},
			Method: method,
			Data:   []byte("hi there"),
		})
		if err != nil {
			log.Printf("ERR: %v\n", err)
		}
		log.Printf("Response: %v\n", response)
	}
}
