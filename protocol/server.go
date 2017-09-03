package protocol

import (
	"encoding/gob"
	"log"
	"net"

	"github.com/pkg/errors"
)

// Server - base server type, contains a listener to listen for sockets
type Server struct {
	listener net.Listener
}

// NewServer - create a new server
func NewServer(proto, address string) (*Server, error) {
	listener, err := net.Listen(proto, address)
	if err != nil {
		return nil, errors.Wrap(err, "failure to create server: ")
	}
	return &Server{
		listener: listener,
	}, nil
}

func (s *Server) Serve() chan struct{} {
	// create a quit channel, so we can signal server to stop
	q := make(chan struct{})
	// start goroutine to accept connections
	go func(quit chan struct{}) {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("ERR: %v", err)
				return
			}
			go s.handleConnection(conn)
		}
	}(q)
	// return the quit channel to caller
	return q
}

// handleConnection - this function will "handle" the accepted connection
// by decoding the request, processing, and returning a response to the request
func (s *Server) handleConnection(conn net.Conn) {
	decoder := gob.NewDecoder(conn)
	var request Request
	err := decoder.Decode(&request)
	if err != nil {
		log.Printf("ERR: %v\n", err)
		return
	}
	// at this point we have a request struct,
	// we will now figure out what type of message it is and perform
	// the method specified
	log.Printf("Got Request: %v\n", request)

	// now we will send back a response
	encoder := gob.NewEncoder(conn)
	encoder.Encode(Response{
		Data: []byte("Well back at you!"),
	})
}
