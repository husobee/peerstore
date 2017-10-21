package protocol

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/crypto"
	"github.com/husobee/peerstore/models"
	"github.com/pkg/errors"
)

// Server - base server type, contains a listener to listen for sockets
type Server struct {
	PrivateKey        *rsa.PrivateKey
	listener          net.Listener
	ctx               context.Context
	connChan          chan net.Conn
	handlerMap        map[RequestMethod]Handler
	handlerMapMu      *sync.RWMutex
	trustedNodes      map[models.Identifier]models.Node
	trustedNodesMapMu *sync.RWMutex
}

// NewServer - create a new server
func NewServer(key *rsa.PrivateKey, peer models.Node, address, dataPath string, bufferSize, numWorkers uint) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "failure to create server: ")
	}
	// make the data dir if it doesnt already exist
	if err := os.MkdirAll(dataPath, 0777); err != nil {
		return nil, errors.Wrap(err, "failed to create data dir: ")
	}

	id := models.Identifier(
		sha1.Sum([]byte(address)),
	)

	return &Server{
		PrivateKey: key,
		listener:   listener,
		ctx: context.WithValue(
			context.WithValue(
				context.Background(), models.DataPathContextKey, dataPath),
			models.NumRequestWorkerContextKey, numWorkers),
		connChan:     make(chan net.Conn, bufferSize),
		handlerMap:   make(map[RequestMethod]Handler),
		handlerMapMu: new(sync.RWMutex),
		trustedNodes: map[models.Identifier]models.Node{
			id: models.Node{
				Addr:      address,
				ID:        id,
				PublicKey: key.Public().(*rsa.PublicKey),
			},
			peer.ID: peer,
		},
		trustedNodesMapMu: new(sync.RWMutex),
	}, nil
}

// startWorkers - we will start the number of numWorkers for the server to
// process requests
func (s *Server) startWorkers() ([]chan bool, []chan bool) {
	var (
		qChans = []chan bool{}
		dChans = []chan bool{}
	)
	var i uint
	for ; i < s.ctx.Value(models.NumRequestWorkerContextKey).(uint); i++ {
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
	// perform decryption of message here on the connection,
	// and take the resulting payload and further decode that
	// as the actual request object.

	// The EncryptedMessage type has the session key
	// which is an RSA encrypted session key, so decrypt
	// with the server's private key, then use that decrypted
	// key to decrypt the AES ciphertext, with the IV in the message.
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
Outer:
	for {
		var em = new(EncryptedMessage)
		err := decoder.Decode(em)
		if err != nil {
			glog.Infof("ERR: %v\n", err)
			if err == io.EOF {
				// the connection has hung up.
				return
			}
			// get the public key of the from
			// another decoding error
			encoder.Encode(Response{
				Status: Error,
			})
		}
		glog.Infof("encrypted message is: %v", em)
		// em now has our encrypted message,
		// decrypt session key
		sessionKey, err := crypto.DecryptRSA(s.PrivateKey, em.SessionKey)
		if err != nil {
			glog.Infof("Invalid Session Key - ERR: %v\n", err)
			return
		}

		glog.Infof("session key is: %v from %v", sessionKey, em.SessionKey)

		// now decrypt the actual payload
		payload, err := crypto.Decrypt(sessionKey, em.CipherText, em.IV)
		if err != nil {
			glog.Infof("Invalid Ciphertext - ERR: %v\n", err)
			return
		}

		// now decode the request from the payload bytes
		payloadDecoder := gob.NewDecoder(bytes.NewBuffer(payload))

		var request = new(Request)
		err = payloadDecoder.Decode(request)
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
			// TODO: encrypt this back to caller
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
			hex.EncodeToString(request.Header.From[:]))

		// lookup the handler to call
		s.handlerMapMu.RLock()
		handler, ok := s.handlerMap[request.Method]
		s.handlerMapMu.RUnlock()
		if ok {
			// TODO: below
			// create session key for caller -> get the public key from trusted nodes
			// or if a user, from the DHT..
			// -> if this is a node registration, or node trust call, it will be added
			// in the handler so we should be good no matter the call.
			// encrypt response from handler
			// encode it to encoder as encrypted message

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
