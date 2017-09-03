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
	response, err := t.RoundTrip(&protocol.Request{
		Header: protocol.Header{},
		Method: protocol.GetFileMethod,
		Data:   []byte("hi there"),
	})
	if err != nil {
		log.Printf("ERR: %v\n", err)
	}
	log.Printf("Response: %v\n", response)
}
