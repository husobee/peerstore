package protocol

import (
	"context"
	"encoding/gob"
	"encoding/hex"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// Server - base server type, contains a listener to listen for sockets
type Server struct {
	listener     net.Listener
	ctx          context.Context
	connChan     chan net.Conn
	handlerMap   map[RequestMethod]Handler
	handlerMapMu *sync.RWMutex
}

// NewServer - create a new server
func NewServer(address, dataPath string, bufferSize, numWorkers uint) (*Server, error) {
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
		connChan:     make(chan net.Conn, bufferSize),
		handlerMap:   make(map[RequestMethod]Handler),
		handlerMapMu: new(sync.RWMutex),
	}, nil
}

// startWorkers - we will start the number of numWorkers for the server to
// process requests
func (s *Server) startWorkers() ([]chan bool, []chan bool) {
	var (
		qChans = []chan bool{}
		dChans = []chan bool{}
	)
	var i uint = 0
	for ; i < s.ctx.Value("numWorkers").(uint); i++ {
		var (
			quit = make(chan bool)
			done = make(chan bool)
		)
		qChans = append(qChans, quit)
		dChans = append(dChans, done)
		go func(i uint) {
			glog.Infof("Starting worker: %d, waiting for connections", i)
			defer glog.Infof("Ending worker: %d", i)
			for {
				select {
				case conn := <-s.connChan:
					// perform handling
					glog.Infof("Worker: %d, accepting connection", i)
					s.handleConnection(conn)
				case <-quit:
					// quit processing connections
					glog.Infof("Worker: %d, quitting.", i)
					done <- true
					return
				}
			}
		}(i)
	}
	return qChans, dChans
}

// Serve - process to serve requests, for each request that we accept
// as a connection, we will fork the handling of that connection.
func (s *Server) Serve(q chan bool, done chan bool) {
	workerQChans, workerDChans := s.startWorkers()
	// start goroutine to accept connections
	for {
		// watch for our quit signal
		select {
		case <-q:
			glog.Info("recieved quit signal, shutting down workers")
			// if we are given a quit signal, signal workers to quit
			// and then return from serving connections
			for _, qChan := range workerQChans {
				qChan <- true
			}
			for _, dChan := range workerDChans {
				<-dChan
			}
			glog.Info("signaling done.")
			done <- true
			return
		default:
			// accept a connection
			s.listener.(*net.TCPListener).SetDeadline(
				time.Now().Add(2 * time.Second))
			conn, err := s.listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok {
					if opErr.Timeout() {
						continue
					}
				}
				glog.Infof("ERR in listener accept: %v", err)
				panic("failed to accept socket")
			}
			// pass connection to a worker through channel
			s.connChan <- conn
		}
	}
}

// handleConnection - this function will "handle" the accepted connection
// by decoding the request, processing, and returning a response to the request
// for the lifetime of the connection
func (s *Server) handleConnection(conn net.Conn) {
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
Outer:
	for {
		var request = new(Request)
		err := decoder.Decode(request)
		if err != nil {
			glog.Infof("ERR: %v\n", err)
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
			glog.Infof("ERR: %v\n", err)
			// write the validation error out.
			encoder.Encode(Response{
				Status: Error,
			})
			continue Outer
		}
		// at this point we have a request struct,
		// we will now figure out what type of message it is and perform
		// the method specified
		glog.Infof("Request: %14s - header_key: %s\n",
			RequestMethodToString[request.Method],
			hex.EncodeToString(request.Header.Key[:]))

		// lookup the handler to call
		s.handlerMapMu.RLock()
		handler, ok := s.handlerMap[request.Method]
		s.handlerMapMu.RUnlock()
		if ok {
			encoder.Encode(handler(s.ctx, request))
			continue Outer
		}
		// no handler to call
		glog.Infof("Request is an Unknown Request")
		encoder.Encode(Response{
			Status: Error,
		})
	}
}

// Handle - add handlers to the server
func (s *Server) Handle(method RequestMethod, fn Handler) {
	s.handlerMapMu.Lock()
	defer s.handlerMapMu.Unlock()
	s.handlerMap[method] = fn
}
