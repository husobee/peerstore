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

// Serve - process to serve requests, for each request that we accept
// as a connection, we will fork the handling of that connection.
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
	encoder := gob.NewEncoder(conn)
	for {
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
		var response = new(Response)

		switch request.Method {
		case GetFileMethod:
			log.Printf("Request is a GetFileMethod Request")
			response.Status = Success
		case PostFileMethod:
			log.Printf("Request is a PostFileMethod Request")
			response.Status = Success
		case DeleteFileMethod:
			log.Printf("Request is a DeleteFileMethod Request")
			response.Status = Success
		default:
			log.Printf("Request is an Unknown Request")
			response.Status = Error
		}

		// now we will send back a response
		encoder.Encode(*response)
	}
}
