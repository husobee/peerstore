package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/husobee/peerstore/protocol"
)

var (
	addr             string
	dataPath         string
	serverBuffer     int
	serverNumWorkers int
)

func init() {
	flag.StringVar(
		&addr, "addr", ":3000",
		"the address for the server to listen")
	flag.StringVar(
		&dataPath, "dataPath", "~/.peerstore",
		"the data location for the server to store files")
	flag.IntVar(
		&serverBuffer, "serverBuffer", runtime.NumCPU()*20,
		"the buffer size of the server for processing requests")
	flag.IntVar(
		&serverNumWorkers, "serverNumWorkers", runtime.NumCPU()*2,
		"the number of server threads for connection processing")
	flag.Parse()
}

func main() {
	log.Println("Starting server.")
	// create a new server
	server, err := protocol.NewServer(addr, dataPath, serverBuffer, serverNumWorkers)
	if err != nil {
		log.Panicf("Failed to create new server: %v", err)
	}
	quit := server.Serve()
	// call the quit to clean up at end of function
	defer func() { quit <- true }()
}
