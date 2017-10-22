package main

import (
	"crypto/rsa"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/chord"
	"github.com/husobee/peerstore/crypto"
	"github.com/husobee/peerstore/file"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
	"github.com/pkg/errors"
)

var (
	// command line flag definitions

	// addr - the address for the server to listen on
	addr string
	// initialPeerAddr - the address for a known peer on the network
	initialPeerAddr string
	// initialPeerKeyFile - the key file location for a known peer on the network
	initialPeerKeyFile string
	// dataPath - the path where the data should be stored
	dataPath string
	// requestQueueBuffer - the number of requests to buffer in the server
	requestQueueBuffer uint
	// requestNumWorkers - the number of request processing workers
	requestNumWorkers uint
)

func init() {
	// initialize the flag package with variables, and then parse the flags
	flag.StringVar(
		&addr, "addr", ":3000",
		"the address for the server to listen")
	flag.StringVar(
		&initialPeerAddr, "initialPeerAddr", "",
		"the address of a known peer on the network")
	flag.StringVar(
		&initialPeerKeyFile, "initialPeerKeyFile", "",
		"the key file location of a known peer on the network")
	flag.StringVar(
		&dataPath, "dataPath", "./.peerstore",
		"the data location for the server to store files")
	flag.UintVar(
		&requestQueueBuffer, "requestQueueBuffer", uint(runtime.NumCPU()*20),
		"the buffer size of the server for processing requests")
	flag.UintVar(
		&requestNumWorkers, "requestNumWorkers", uint(runtime.NumCPU()*2),
		"the number of server threads for connection processing")
	flag.Parse()
}

func validateParams() error {
	if addr == "" {
		return errors.New("addr must be set")
	}
	if initialPeerAddr == "" {
		return errors.New("intialPeerAddr must be set")
	}
	if dataPath == "" {
		return errors.New("dataPath must be set")
	}
	info, err := os.Stat(dataPath)
	if err != nil {
		return errors.Wrap(err, "error attempting to validate dataPath: ")
	}
	if !info.IsDir() {
		return errors.New("dataPath must be a valid directory")
	}

	return nil
}

func main() {
	defer glog.Flush()
	// validate our command line parameters
	if err := validateParams(); err != nil {
		glog.Fatalf("failed to validate command line params: %v\n", err)
	}

	var (
		// quit - channel to inform the server to stop listening
		// signal chord to "leave" the network
		quit = make(chan bool)
		// done - channel to inform main the server is shutdown
		// and the chord node has left the network
		done = make(chan bool)
	)

	// handle interupts gracefully
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			glog.Info("Interrupt, Killing workers")
			// signal server to quit processing requests
			quit <- true
			// wait for server to be finished
			<-done
			glog.Info("Done.")
			os.Exit(0)
		}
	}()

	var (
		peerNode models.Node
		key      *rsa.PrivateKey
		err      error
	)

	privateKeyFile, err := os.Open(
		fmt.Sprintf("%s/privatekey.pem", dataPath))

	if err != nil {
		// generate our public key
		key, err = crypto.GenerateKeyPair()
		if err != nil {
			glog.Infof("failed to generate keypair: %s", err)
			return
		}
		// create our keypair file:
		privateKeyFile, err := os.Create(
			fmt.Sprintf("%s/privatekey.pem", dataPath))
		if err != nil {
			glog.Infof("failed to create keypair file: %s", err)
			return
		}
		crypto.WritePrivateKeyAsPem(privateKeyFile, key)
		privateKeyFile.Close()

		publicKeyFile, err := os.Create(
			fmt.Sprintf("%s/publickey.pem", dataPath))
		if err != nil {
			glog.Infof("failed to create keypair file: %s", err)
			return
		}
		crypto.WritePublicKeyAsPem(publicKeyFile, key.Public().(*rsa.PublicKey))
		publicKeyFile.Close()
	} else {
		key, err = crypto.ReadKeypairAsPem(privateKeyFile)
		if err != nil {
			glog.Infof("failed to read keypair: %s", err)
			return
		}
	}

	// if no peer is specified, we are the only one, so dont read a peer
	if initialPeerKeyFile != "" {
		// read in our peer's public key
		keyFile, err := os.Open(initialPeerKeyFile) // For read access.
		if err != nil {
			glog.Infof("failed to read initial peer key file: %s", err)
			return
		}

		peerKey, err := crypto.ReadPublicKeyAsPem(keyFile)
		if err != nil {

		}

		peerNode = models.Node{
			Addr:      initialPeerAddr,
			PublicKey: &peerKey,
			ID:        sha1.Sum([]byte(addr)),
		}
	}

	// create a server to listen on
	server, err := protocol.NewServer(
		key, peerNode, addr, dataPath, requestQueueBuffer, requestNumWorkers)
	if err != nil {
		glog.Fatalf("Failed to create new server: %v", err)
	}

	if initialPeerKeyFile != "" {
		// need to register with our peer first thing
		t, err := protocol.NewTransport("tcp", peerNode.Addr, protocol.NodeType, models.Identifier(sha1.Sum([]byte(addr))), peerNode.PublicKey, key)
		resp, err := t.RoundTrip(&protocol.Request{
			Header: protocol.Header{
				From:     models.Identifier(sha1.Sum([]byte(addr))),
				FromAddr: addr,
				Type:     protocol.NodeType,
				PubKey:   key.Public().(*rsa.PublicKey),
			},
			Method: protocol.NodeRegistrationMethod,
		})
		if err != nil {
			// failed to register with peer node
			glog.Infof("failed to register trust with peer node")
			return
		}
		glog.Infof("Response from registration: %+v", resp)
		// TODO: iterate through all nodes in response, and contact all of
		// them to "NodeTrustMethod" them
	}

	// create our local chord node.
	localNode, err := chord.NewLocalNode(server, addr, peerNode)

	glog.Infof("!!! local node: addr=%s, id=%s\n",
		localNode.Addr,
		hex.EncodeToString(localNode.ID[:]))

	if err != nil {
		// error condition happens when node is unable to connect to
		// the peer specified, we shall log the error, and use an uninitialized
		// peer for now
		glog.Infof("failed to create chord local node: %v\n", err)
	}

	// Start stabilizing!
	go func() {
		for {
			select {
			case <-time.After(10 * time.Second):
				localNode.Stabilize()
				// TODO: use quit chan to stop stabilization
			}
		}
	}()

	glog.Infof("Starting server - %s, %s, %d, %d",
		addr, dataPath, requestQueueBuffer, requestNumWorkers)

	// file handler routes
	server.Handle(protocol.GetFileMethod, file.GetFileHandler)
	server.Handle(protocol.PostFileMethod, file.PostFileHandler)
	server.Handle(protocol.DeleteFileMethod, file.DeleteFileHandler)
	// chord handler routes
	server.Handle(protocol.GetSuccessorMethod, localNode.SuccessorHandler)
	server.Handle(protocol.SetPredecessorMethod, localNode.SetPredecessorHandler)
	server.Handle(protocol.GetPredecessorMethod, localNode.GetPredecessorHandler)
	server.Handle(protocol.GetFingerTableMethod, localNode.FingerTableHandler)
	// registration route
	server.Handle(protocol.UserRegistrationMethod, server.UserRegistrationHandler)
	// node registration route
	server.Handle(protocol.NodeRegistrationMethod, server.NodeRegistrationHandler)
	server.Handle(protocol.NodeTrustMethod, server.NodeTrustHandler)

	go func() {
		for {
			select {
			case <-time.After(30 * time.Second):
				hash := sha1.Sum([]byte("hello"))

				node, err := localNode.Successor(models.Identifier(hash))
				if err != nil {
					glog.Infof("!!!!!!!!!!!!!!!!! error finding node : %s", err)
					continue
				}
				glog.Infof("!!!!!!!!!!!!!!!!! Hash %d goes to node: %s", models.KeyToID(models.Identifier(hash)), node.ToString())
			}
		}
	}()

	// serve requests
	server.Serve(quit, done)
}
