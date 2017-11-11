package protocol

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/gob"
	"encoding/hex"
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
	id                models.Identifier
	addr              string
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
	trustedNodes := map[models.Identifier]models.Node{
		id: models.Node{
			Addr:      address,
			ID:        id,
			PublicKey: key.Public().(*rsa.PublicKey),
		},
	}
	if peer.Addr != "" {
		trustedNodes[peer.ID] = peer
	}
	ctx := context.WithValue(context.Background(), models.DataPathContextKey, dataPath)
	ctx = context.WithValue(ctx, models.NumRequestWorkerContextKey, numWorkers)
	ctx = context.WithValue(ctx, models.SelfPrivateKeyContextKey, key)
	ctx = context.WithValue(ctx, models.SelfIDContextKey, id)
	ctx = context.WithValue(ctx, models.SelfNodeContextKey, trustedNodes[id])

	return &Server{
		PrivateKey:   key,
		listener:     listener,
		id:           id,
		addr:         address,
		ctx:          ctx,
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

// addTrustedNode - Add a node as a trusted node in the trustedNodes structure
func (s *Server) addTrustedNode(node models.Node) {
	s.trustedNodesMapMu.Lock()
	defer s.trustedNodesMapMu.Unlock()
	glog.Infof("adding a trusted node: %s", node)
	s.trustedNodes[node.ID] = node
}

// getTrustedNode - Get a node from the trustedNodes structure
func (s *Server) getTrustedNode(id models.Identifier) (models.Node, error) {
	s.trustedNodesMapMu.RLock()
	defer s.trustedNodesMapMu.RUnlock()
	glog.Infof("trusted nodes: %+v", s.trustedNodes)
	if node, ok := s.trustedNodes[id]; ok {
		glog.Infof("getting trusted node: %s", node.ToString())
		return node, nil
	}

	return models.Node{}, errors.New("node does not exist in trustedNodes")
}

// getAllTrustedNodes - Get a list of trustedNodes
func (s *Server) getAllTrustedNodes() []models.Node {
	s.trustedNodesMapMu.RLock()
	defer s.trustedNodesMapMu.RUnlock()
	resp := []models.Node{}
	for _, node := range s.trustedNodes {
		resp = append(resp, node)
	}
	return resp
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
		em, request, raw, err := decryptAndDecodeRequest(decoder, s.PrivateKey)

		if err != nil {
			glog.Infof("err: %v\n", err)
			return
		}
		// at this point we have a request struct,
		// we will now figure out what type of message it is and perform
		// the method specified
		glog.Infof("Request: %14s - header_key: %s, %+v\n",
			RequestMethodToString[request.Method],
			hex.EncodeToString(request.Header.From[:]),
			request,
		)
		glog.Infof("EM is %+v", em)

		// lookup the handler to call
		s.handlerMapMu.RLock()
		handler, ok := s.handlerMap[request.Method]
		s.handlerMapMu.RUnlock()
		s.ctx = context.WithValue(s.ctx, models.UserPublicKeyContextKey, em.Header.PubKey)
		s.ctx = context.WithValue(s.ctx, models.ResourceNameContextKey, request.Header.ResourceName)

		if ok {
			// based on the type, we are going to authenticate this request
			glog.Infof("header type is: %d", em.Header.Type)
			switch em.Header.Type {
			case UserType:
				// in the event this is a user type we need to call ourself to
				// figure out which node to talk to in order to get the public
				// key file.  We will masqurade as the "from" for our request
				// and get the key file.  When we have the key file from
				// the dht, we will use that key file to validate the user's
				// signature of the request.  if the signature is invalid,
				// we will respond with an error, as this request is not authorized

				// lookup the user based on the From field in the request header
				if request.Method != UserRegistrationMethod {
					// lookup the public key based on from header in request
					// figure out where to connect to
					t, err := NewTransport("tcp", s.addr, NodeType, s.id, s.PrivateKey.Public().(*rsa.PublicKey), s.PrivateKey)
					if err != nil {
						glog.Infof("ERR: %v", err)
						return
					}
					// serialize our get successor request
					var idBuf = new(bytes.Buffer)
					enc := gob.NewEncoder(idBuf)
					enc.Encode(models.SuccessorRequest{
						models.Identifier(request.Header.Key),
					})

					glog.Infof("about to round trip to find successor to get file node")
					resp, err := t.RoundTrip(&Request{
						Header: Header{
							From: s.id,
							Key:  request.Header.From,
						},
						Method: GetSuccessorMethod,
						Data:   idBuf.Bytes(),
					})
					t.Close()
					if err != nil {
						glog.Infof("Failed to round trip the successor request: %v", err)
						return
					}
					// connect to that host for this file
					// pull node out of response, and connect to that host
					var node = models.Node{}
					dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
					err = dec.Decode(&node)
					if err != nil {
						glog.Infof("Failed to deserialize the node data: %v", err)
						return
					}

					glog.Infof("connecting to node with the public key")
					// OKAY, NOW connect to it, and get the file
					// figure out where to connect to, by asking self
					st, err := NewTransport("tcp", node.Addr, NodeType, s.id, node.PublicKey, s.PrivateKey)
					if err != nil {
						glog.Infof("ERR: %v", err)
						return
					}

					glog.Infof("server id is : %+v", s.id)
					response, err := st.RoundTrip(&Request{
						Header: Header{
							Key:  request.Header.From,
							From: s.id,
						},
						Method: GetFileMethod,
					})
					st.Close()
					if err != nil {
						glog.Infof("ERR: %v\n", err)
						return
					}
					glog.Infof("response from file post: %+v", response)

					// response.data has the pem, need to read that
					pubKey, err := crypto.ReadPublicKeyAsPem(bytes.NewBuffer(response.Data))
					if err != nil {
						glog.Infof("ERR: %v\n", err)
						return
					}
					// validate the signature on the request! almost done!
					if err := crypto.Verify(&pubKey, em.Header.Signature, raw); err != nil {

						glog.Infof("unable to validate signature for user request: %v\n", err)
						return
					}
				}

			case NodeType:
				// if this is a node type request, we need to validate this node
				// is in our trustedNodes map, and use the public key from
				// there to validate the request, if the request signature is not
				// valid we will return an error
				// skip this if this is a node registration request
				if request.Method != NodeRegistrationMethod {
					node, err := s.getTrustedNode(request.Header.From)
					if err != nil {
						glog.Infof("failed to get trusted node: %s", err)
						// if there was an error, respond with error
						encryptAndEncode(encoder, Response{
							Status: Error,
						}, NodeType, em.Header.PubKey, s.id, s.PrivateKey)
						return
					}
					glog.Infof("node from trustedNodes: %s", node.ToString())
					glog.Infof("bytes are: %x", raw)
					glog.Infof("signature from header: %x", em.Header.Signature)

					if err := crypto.Verify(em.Header.PubKey, em.Header.Signature, raw); err != nil {
						glog.Infof("Failed to verify node message: %s", err)
						encryptAndEncode(encoder, Response{
							Status: Error,
						}, NodeType, em.Header.PubKey, s.id, s.PrivateKey)
						return
					}
				}
			default:
				// has to be one of the above two.
				encryptAndEncode(encoder, Response{
					Status: Error,
				}, NodeType, em.Header.PubKey, s.id, s.PrivateKey)
			}

			encryptAndEncode(
				encoder, handler(s.ctx, request), NodeType, em.Header.PubKey, s.id, s.PrivateKey)
			continue Outer
		}
		// no handler to call
		glog.Infof("Request is an Unknown Request")
		encryptAndEncode(encoder, Response{
			Status: Error,
		}, NodeType, em.Header.PubKey, s.id, s.PrivateKey)
	}
}

// Handle - add handlers to the server
func (s *Server) Handle(method RequestMethod, fn Handler) {
	s.handlerMapMu.Lock()
	defer s.handlerMapMu.Unlock()
	s.handlerMap[method] = fn
}

func encryptAndEncode(enc encoder, payload interface{}, t CallerType, peerKey *rsa.PublicKey, from models.Identifier, selfKey *rsa.PrivateKey) error {
	// create a buffer for the request to be serialized to
	buf := bytes.NewBuffer([]byte{})

	// serialize the request to the buffer
	requestEncoder := gob.NewEncoder(buf)
	if err := requestEncoder.Encode(payload); err != nil {
		glog.Infof("failed to encode request: %s", err)
		return errors.Wrap(err, "failure encoding request: ")
	}

	// sign the request bytes
	signature, err := crypto.Sign(selfKey, buf.Bytes())

	glog.Infof("bytes are: %x", buf.Bytes())
	glog.Infof("computed signature: %x", signature)

	// generate the session key
	plaintextKey, ciphertextKey, err := crypto.GenerateSessionKey(peerKey)
	if err != nil {
		glog.Infof("failed to generate session key: %s", err)
		return errors.Wrap(err, "failure generating session: ")
	}
	// encrypt with AES
	ciphertext, iv, err := crypto.Encrypt(plaintextKey, buf.Bytes())
	if err != nil {
		glog.Infof("failed to generate ciphertext: %s", err)
		return errors.Wrap(err, "failure generating ciphertext: ")
	}

	respEM := &EncryptedMessage{
		Header: Header{
			Type:      t,
			PubKey:    selfKey.Public().(*rsa.PublicKey),
			From:      from,
			Signature: signature,
		},
		SessionKey: ciphertextKey,
		IV:         iv,
		CipherText: ciphertext,
	}

	// serialize request
	if err := enc.Encode(respEM); err != nil {
		return errors.Wrap(err, "failure encoding request: ")
	}
	return nil
}

func decryptAndDecodeResponse(dec decoder, selfKey *rsa.PrivateKey) (*EncryptedMessage, *Response, []byte, error) {
	var em = new(EncryptedMessage)
	err := dec.Decode(em)
	if err != nil {
		glog.Infof("ERR: %v\n", err)
		return em, nil, nil, errors.Wrap(err, "failed to decrypt response")
	}

	// validate response
	if err := em.Validate(); err != nil {
		return em, nil, nil, errors.Wrap(err, "failure validating response: ")
	}

	// em now has our encrypted message,
	// decrypt session key
	sessionKey, err := crypto.DecryptRSA(selfKey, em.SessionKey)
	if err != nil {
		glog.Infof("Invalid Session Key - ERR: %v\n", err)
		return em, nil, nil, errors.Wrap(err, "invalid session key")
	}

	glog.Infof("session key is: %v from %v", sessionKey, em.SessionKey)

	// now decrypt the actual payload
	payload, err := crypto.Decrypt(sessionKey, em.CipherText, em.IV)
	if err != nil {
		glog.Infof("Invalid Ciphertext - ERR: %v\n", err)
		return em, nil, nil, errors.Wrap(err, "invalid ciphertext")
	}

	// now decode the request from the payload bytes
	payloadDecoder := gob.NewDecoder(bytes.NewBuffer(payload))

	var response = new(Response)
	err = payloadDecoder.Decode(response)
	if err != nil {
		return em, nil, nil, errors.Wrap(err, "failed to decode response")
	}
	// validate response
	if err := response.Validate(); err != nil {
		return em, nil, nil, errors.Wrap(err, "failure validating response: ")
	}
	return em, response, payload, nil
}

func decryptAndDecodeRequest(dec decoder, selfKey *rsa.PrivateKey) (*EncryptedMessage, *Request, []byte, error) {
	var em = new(EncryptedMessage)
	err := dec.Decode(em)
	if err != nil {
		glog.Infof("ERR: %v\n", err)
		return em, nil, nil, errors.Wrap(err, "failed to decrypt response")
	}

	// validate response
	if err := em.Validate(); err != nil {
		return em, nil, nil, errors.Wrap(err, "failure validating response: ")
	}

	// em now has our encrypted message,
	// decrypt session key
	sessionKey, err := crypto.DecryptRSA(selfKey, em.SessionKey)
	if err != nil {
		glog.Infof("Invalid Session Key - ERR: %v\n", err)
		return em, nil, nil, errors.Wrap(err, "invalid session key")
	}

	glog.Infof("session key is: %v from %v", sessionKey, em.SessionKey)

	// now decrypt the actual payload
	payload, err := crypto.Decrypt(sessionKey, em.CipherText, em.IV)
	if err != nil {
		glog.Infof("Invalid Ciphertext - ERR: %v\n", err)
		return em, nil, nil, errors.Wrap(err, "invalid ciphertext")
	}

	// now decode the request from the payload bytes

	glog.Infof("bytes after decryption are: %x", payload)
	payloadDecoder := gob.NewDecoder(bytes.NewBuffer(payload))

	var request = new(Request)
	err = payloadDecoder.Decode(request)
	if err != nil {
		return em, nil, nil, errors.Wrap(err, "failed to decode response")
	}

	if err := request.Validate(); err != nil {
		return em, nil, nil, errors.Wrap(err, "failed to validate request")
	}
	return em, request, payload, nil
}
