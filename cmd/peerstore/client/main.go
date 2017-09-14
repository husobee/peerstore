package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/husobee/peerstore/protocol"
	"github.com/pkg/errors"
)

var (
	peerAddr  string
	localPath string
)

func init() {
	flag.StringVar(
		&peerAddr, "peerAddr", "",
		"the address of a peer")
	flag.StringVar(
		&localPath, "localPath", "",
		"the location of the dir you wish to sync")
	flag.Parse()
}

func validateParams() error {
	if peerAddr == "" {
		return errors.New("peerAddr must be set")
	}
	if localPath == "" {
		return errors.New("localPath must be set")
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return errors.Wrap(err, "error attempting to validate localPath: ")
	}
	if !info.IsDir() {
		return errors.New("localPath must be a valid directory")
	}
	return nil
}

func main() {

	log.Println("starting client")

	if err := validateParams(); err != nil {
		log.Fatalf("could not validate params: %v\n", err)
	}

	t, err := protocol.NewTransport("tcp", peerAddr)
	if err != nil {
		log.Printf("ERR: %v", err)
	}
	var walkFn = func(path string, fi os.FileInfo, err error) error {
		if !fi.IsDir() {
			log.Printf("file is: %s\n", path)

			// the key for the distributed lookup TODO: fix this
			key := sha1.Sum([]byte(path))
			bytes, err := ioutil.ReadFile(path) // path is the path to the file.

			log.Println("starting request: ", protocol.PostFileMethod)
			response, err := t.RoundTrip(&protocol.Request{
				Header: protocol.Header{
					Key:        key,
					DataLength: uint64(len(bytes)),
				},
				Method: protocol.PostFileMethod,
				Data:   bytes,
			})
			if err != nil {
				log.Printf("ERR: %v\n", err)
			}
			log.Printf("Response: %+v\n", response)

			if err != nil {
				fmt.Println("Fail")
			}
		}
		return nil
	}

	// Open up directory
	// read each file, and send to peerAddr
	filepath.Walk(localPath, walkFn)

	// try to read a file back from the server as a new file
	// validate hashes match

	/*
		for _, method := range []protocol.RequestMethod{
			protocol.PostFileMethod,
			protocol.GetFileMethod,
			protocol.DeleteFileMethod,
			42,
		} {
			log.Println("starting request: ", method)
			response, err := t.RoundTrip(&protocol.Request{
				Header: protocol.Header{},
				Method: method,
				Data:   []byte("hi there"),
			})
			if err != nil {
				log.Printf("ERR: %v\n", err)
			}
			log.Printf("Response: %+v\n", response)
		}
	*/
}
