package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/husobee/peerstore/protocol"
)

func main() {
	fmt.Println("server")
	listener, _ := net.Listen("tcp", ":3000")

	for {
		conn, _ := listener.Accept()
		// get requests
		decoder := gob.NewDecoder(conn)
		var request protocol.Request
		err := decoder.Decode(&request)
		if err != nil {
			log.Printf("ERR: %v\n", err)
			if err == io.EOF {
				continue
			}
		} else {
			log.Printf("Got Request: %v\n", request)
			encoder := gob.NewEncoder(conn)
			encoder.Encode(protocol.Response{
				Data: []byte("Well back at you!"),
			})
		}
	}
}
