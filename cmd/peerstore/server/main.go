package main

import (
	"flag"
	"log"

	"github.com/husobee/peerstore/protocol"
)

var (
	addr     string
	dataPath string
)

func init() {
	flag.StringVar(
		&addr, "addr", ":3000",
		"the address for the server to listen")
	flag.StringVar(
		&dataPath, "dataPath", "~/.peerstore",
		"the data location for the server to store files")
	flag.Parse()
}

func main() {
	log.Println("Starting server.")
	// create a new server
	server, err := protocol.NewServer(addr, dataPath)
	if err != nil {
		log.Panicf("Failed to create new server: %v", err)
	}
	quit := server.Serve()
	// call the quit to clean up at end of function
	defer func() { quit <- struct{}{} }()
}
