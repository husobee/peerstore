package protocol

import (
	"context"
	"encoding/gob"
	"io"
	"log"
	"net"
	"os"

	"github.com/pkg/errors"
)

// Server - base server type, contains a listener to listen for sockets
type Server struct {
	listener net.Listener
	ctx      context.Context
	connChan chan net.Conn
}

// NewServer - create a new server
func NewServer(address, dataPath string, bufferSize, numWorkers int) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "failure to create server: ")
	}
	// make the data dir if it doesnt already exist
	if err := os.MkdirAll(dataPath, 0777); err != nil {
		return nil, errors.Wrap(err, "failed to create data dir: ")
	}

	return &Server{
		listener: listener,
		ctx: context.WithValue(
			context.WithValue(context.Background(), "dataPath", dataPath),
			"numWorkers", numWorkers),
		connChan: make(chan net.Conn, bufferSize),
	}, nil
}

// startWorkers - we will start the number of numWorkers for the server to
// process requests
func (s *Server) startWorkers() []chan bool {
	qChans := []chan bool{}
	for i := 0; i < s.ctx.Value("numWorkers").(int); i++ {
		q := make(chan bool)
		qChans = append(qChans, q)
		go func() {
			for {
				select {
				case conn := <-s.connChan:
					// perform handling
					s.handleConnection(conn)
				case <-q:
					// quit processing connections
					return
				}
			}
		}()
	}
	return qChans
}

// Serve - process to serve requests, for each request that we accept
// as a connection, we will fork the handling of that connection.
func (s *Server) Serve() chan bool {
	// create a quit channel, so we can signal server to stop
	q := make(chan bool)
	workerQChans := s.startWorkers()
	// start goroutine to accept connections
	go func() {
		for {
			select {
			case <-q:
				// if we are given a quit signal, signal workers to quit
				// and then return from serving connections
				for _, qChan := range workerQChans {
					qChan <- true
				}
				return
			default:
				// accept a connection
				conn, err := s.listener.Accept()
				if err != nil {
					log.Printf("ERR in listener accept: %v", err)
					panic("failed to accept socket")
				}
				// pass connection to a worker through channel
				s.connChan <- conn
			}
		}
	}()
	// return the quit channel to caller
	return q
}

// handleConnection - this function will "handle" the accepted connection
// by decoding the request, processing, and returning a response to the request
func (s *Server) handleConnection(conn net.Conn) {
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
Outer:
	for {
		var request = new(Request)
		err := decoder.Decode(request)
		if err != nil {
			log.Printf("ERR: %v\n", err)
			if err == io.EOF {
				// the connection has hung up.
				return
			}
			// another decoding error
			encoder.Encode(Response{
				Status: Error,
			})
		}

		if request.Validate(); err != nil {
			log.Printf("ERR: %v\n", err)
			// write the validation error out.
			encoder.Encode(Response{
				Status: Error,
			})
			continue Outer
		}
		// at this point we have a request struct,
		// we will now figure out what type of message it is and perform
		// the method specified
		log.Printf("Got Request: %+v\n", request)

		// lookup the handler to call
		if handler, ok := MethodHandlerMap[request.Method]; ok {
			encoder.Encode(handler(s.ctx, request))
			continue Outer
		}
		// no handler to call
		log.Printf("Request is an Unknown Request")
		encoder.Encode(Response{
			Status: Error,
		})
	}
}
