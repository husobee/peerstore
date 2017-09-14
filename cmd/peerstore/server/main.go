package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
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
	// create a new server
	server, err := protocol.NewServer(addr, dataPath, serverBuffer, serverNumWorkers)
	if err != nil {
		log.Panicf("Failed to create new server: %v", err)
	}

	log.Println("Starting server - ", addr, dataPath, serverBuffer, serverNumWorkers)
	var (
		quit = make(chan bool)
		done = make(chan bool)
	)

	// handle interupts gracefully
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for _ = range signalChan {
			log.Println("Interrupt, Killing workers")
			// signal server to quit processing requests
			quit <- true
			// wait for server to be finished
			<-done
			log.Println("Done.")
			os.Exit(0)
		}
	}()

	// start server
	server.Serve(quit, done)

}
