package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/gob"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/husobee/peerstore/crypto"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
	"github.com/pkg/errors"
)

var (
	peerAddr string
	// peerKeyFile - the key file location for a known peer on the network
	peerKeyFile  string
	selfKeyFile  string
	localPath    string
	operation    string
	filename     string
	filedest     string
	pollInterval time.Duration
)

func init() {
	flag.StringVar(
		&peerAddr, "peerAddr", "",
		"the address of a peer")
	flag.StringVar(
		&operation, "operation", "",
		"choice of operation, backup or getfile.  backup will put localPath in peerstore, getfile will download the file and put it in filedest. specify the file to download by name with -filename flag")
	flag.StringVar(
		&localPath, "localPath", "",
		"the location of the dir you wish to sync")
	flag.StringVar(
		&filename, "filename", "",
		"the name of the file you want to get from peerstore")
	flag.StringVar(
		&filedest, "filedest", "",
		"destination of the file with doing getfile operation")
	flag.StringVar(
		&peerKeyFile, "peerKeyFile", "",
		"the key file location of a known peer on the network")
	flag.StringVar(
		&selfKeyFile, "selfKeyFile", "",
		"the key file location of your private/public key pem file")
	flag.DurationVar(&pollInterval, "poll", time.Second, "the polling interval for sync")
	flag.Parse()
}

func validateParams() error {
	if peerAddr == "" {
		return errors.New("peerAddr must be set")
	}
	if operation == "backup" {
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
	} else if operation == "sync" {
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
	} else if operation == "getfile" {
		if filedest == "" {
			return errors.New("filedest must be set")
		}
		if filename == "" {
			return errors.New("filename must be set")
		}

	} else {
		return errors.New("must specify operation flag, either backup or getfile")
	}
	return nil
}

func main() {

	log.Println("starting client")

	if err := validateParams(); err != nil {
		log.Fatalf("could not validate params: %v\n", err)
	}

	var (
		privateKey *rsa.PrivateKey
		err        error
	)

	if _, err := os.Stat(selfKeyFile); err != nil {
		// generate our public key
		privateKey, err = crypto.GenerateKeyPair()
		if err != nil {
			log.Printf("failed to generate keypair: %s", err)
			return
		}
		// create our keypair file:
		keyFile, err := os.Create(fmt.Sprintf("%s", selfKeyFile))
		if err != nil {
			glog.Infof("failed to create keypair file: %s", err)
			return
		}
		crypto.WritePrivateKeyAsPem(keyFile, privateKey)
		crypto.WritePublicKeyAsPem(keyFile, privateKey.Public().(*rsa.PublicKey))
		keyFile.Close()
	} else {
		keyFile, err := os.Open(fmt.Sprintf("%s", selfKeyFile))
		privateKey, err = crypto.ReadKeypairAsPem(keyFile)
		if err != nil {
			log.Printf("failed to read keypair: %s", err)
			return
		}
	}

	kb, _ := crypto.GobEncodePublicKey(privateKey.Public().(*rsa.PublicKey))
	id := models.Identifier(sha1.Sum(kb))

	// read in our peer's public key
	keyFile, err := os.Open(peerKeyFile) // For read access.
	if err != nil {
		glog.Infof("failed to read initial peer key file: %s", err)
		return
	}

	peerKey, err := crypto.ReadPublicKeyAsPem(keyFile)
	if err != nil {
		glog.Infof("failed to read keypair file: %s", err)
		return
	}

	// register the user with the network
	log.Printf("usertype should be : %d", protocol.UserType)
	rt, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, &peerKey, privateKey)
	if err != nil {
		log.Printf("ERR: %v", err)
		return
	}
	log.Println("transport established")

	resp, err := rt.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			From:   id,
			Type:   protocol.UserType,
			PubKey: privateKey.Public().(*rsa.PublicKey),
		},
		Method: protocol.UserRegistrationMethod,
	})
	log.Println("registered user")
	if err != nil {
		log.Printf("Failed to round trip the successor request: %v", err)
		return
	}
	rt.Close()
	log.Printf("response: %+v", resp)

	switch operation {
	case "sync":
		log.Println("starting sync!")

		var (
			quitChan   = make(chan bool)
			signalChan = make(chan os.Signal)
		)
		// TODO: need to kickoff a lookup to the transaction log in the DHT
		// if there is a transaction log, we need to perform a get on all the
		// resources that are listed in the transaction log and update our
		// transaction log

		// TODO: need to kick off an fsnotify to watch for changes to files
		// (except when we make changes from the sync)
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("failed to start fs watcher: %s", err)
			os.Exit(1)
		}
		defer watcher.Close()
		log.Println("sync watcher has been created")

		// set the watcher to watch the localpath
		err = watcher.Add(localPath)
		if err != nil {
			log.Printf("fs watcher failed to add localpath: %s", err)
			os.Exit(1)
		}

		// watch for an interrupt
		signal.Notify(signalChan, os.Interrupt)
		go func() {
			for _ = range signalChan {
				log.Print("Interrupt, Killing workers")
				// signal server to quit processing requests
				quitChan <- true
			}
		}()

		var transactionLog = models.TransactionLog{}

		log.Println("starting signal loop")
		for {
			select {
			case <-quitChan:
				os.Exit(0)
			case <-time.After(pollInterval):
				// get the transaction log, look for differences
				// if differences, get the resources that are different
			case event := <-watcher.Events:
				// we got a filesystem event, pull remote transaction log
				// TODO: get transaction log and associated resources
				// update it accordingly and save
				if event.Op == fsnotify.Write {
					log.Println("file written: ", event.Name)
					// update local transaction log
					resourceID := sha1.Sum([]byte(event.Name))
					timestamp := models.GetClock()

					// write file to server where it belongs (will update remote)
					data, err := ioutil.ReadFile(event.Name) // path is the path to the file.

					// figure out where to connect to
					st, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, &peerKey, privateKey)
					if err != nil {
						log.Printf("ERR: %v", err)
					}

					// serialize our get successor request
					var idBuf = new(bytes.Buffer)
					enc := gob.NewEncoder(idBuf)
					enc.Encode(models.SuccessorRequest{
						models.Identifier(resourceID),
					})
					resp, err := st.RoundTrip(&protocol.Request{
						Header: protocol.Header{
							From:   id,
							Type:   protocol.UserType,
							PubKey: privateKey.Public().(*rsa.PublicKey),
						},
						Method: protocol.GetSuccessorMethod,
						Data:   idBuf.Bytes(),
					})
					if err != nil {
						log.Printf("Failed to round trip the successor request: %v", err)
					}
					st.Close()

					// connect to that host for this file
					// pull node out of response, and connect to that host
					var node = models.Node{}
					dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
					err = dec.Decode(&node)
					if err != nil {
						log.Printf("Failed to deserialize the node data: %v", err)
					}

					// figure out where to connect to
					t, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, node.PublicKey, privateKey)
					if err != nil {
						log.Printf("ERR: %v", err)
					}

					// send the file over
					log.Println("starting request: ", protocol.PostFileMethod)
					resp, err = t.RoundTrip(&protocol.Request{
						Header: protocol.Header{
							Key:          resourceID,
							Type:         protocol.UserType,
							From:         id,
							DataLength:   uint64(len(data)),
							PubKey:       privateKey.Public().(*rsa.PublicKey),
							ResourceName: event.Name,
							Log:          true,
							Clock:        timestamp,
						},
						Method: protocol.PostFileMethod,
						Data:   data,
					})
					if err != nil {
						log.Printf("ERR: %v\n", err)
					}
					log.Printf("Response: %+v\n", resp)

					if err != nil {
						fmt.Println("Fail")
					}
					t.Close()

					// after getting a response, use the response header "Clock"
					// as the timestamp in the local transaction log, if err, it
					// will be zero in the local transaction log
					transactionLog = append(transactionLog,
						models.TransactionEntity{
							Operation:    models.UpdateOperation,
							ResourceName: event.Name,
							ResourceID:   resourceID,
							ClientID:     id,
							Timestamp:    models.IncrementClock(resp.Header.Clock),
						})

				}
				if event.Op == fsnotify.Remove {
					log.Println("file removed: ", event.Name)
					// update local transaction log
					// delete file from server (this will update remote trans log)
				}
			case err := <-watcher.Errors:
				// somthing terrible happened with our FS watcher
				log.Printf("fs watcher error: %s", err)
				os.Exit(1)
			}
		}

	case "backup":
		var walkFn = func(path string, fi os.FileInfo, err error) error {
			if !fi.IsDir() {
				log.Printf("file is: %s\n", path)

				// the key for the distributed lookup
				key := sha1.Sum([]byte(path))
				data, err := ioutil.ReadFile(path) // path is the path to the file.

				// figure out where to connect to
				st, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, &peerKey, privateKey)
				if err != nil {
					log.Printf("ERR: %v", err)
				}

				// serialize our get successor request
				var idBuf = new(bytes.Buffer)
				enc := gob.NewEncoder(idBuf)
				enc.Encode(models.SuccessorRequest{
					models.Identifier(key),
				})
				resp, err := st.RoundTrip(&protocol.Request{
					Header: protocol.Header{
						From:   id,
						Type:   protocol.UserType,
						PubKey: privateKey.Public().(*rsa.PublicKey),
					},
					Method: protocol.GetSuccessorMethod,
					Data:   idBuf.Bytes(),
				})
				if err != nil {
					log.Printf("Failed to round trip the successor request: %v", err)
					return errors.Wrap(err, "failed round trip")
				}
				st.Close()

				// connect to that host for this file
				// pull node out of response, and connect to that host
				var node = models.Node{}
				dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
				err = dec.Decode(&node)
				if err != nil {
					log.Printf("Failed to deserialize the node data: %v", err)
					return errors.Wrap(err, "failed to deserialize")
				}

				// figure out where to connect to
				t, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, node.PublicKey, privateKey)
				if err != nil {
					log.Printf("ERR: %v", err)
				}

				// send the file over
				log.Println("starting request: ", protocol.PostFileMethod)
				response, err := t.RoundTrip(&protocol.Request{
					Header: protocol.Header{
						Key:          key,
						Type:         protocol.UserType,
						From:         id,
						DataLength:   uint64(len(data)),
						PubKey:       privateKey.Public().(*rsa.PublicKey),
						ResourceName: path,
						Log:          true,
					},
					Method: protocol.PostFileMethod,
					Data:   data,
				})
				if err != nil {
					log.Printf("ERR: %v\n", err)
				}
				log.Printf("Response: %+v\n", response)

				if err != nil {
					fmt.Println("Fail")
				}
				t.Close()
			}
			return nil
		}

		// Open up directory
		// read each file, and send to peerAddr
		filepath.Walk(localPath, walkFn)

	case "getfile":
		log.Printf("getting file: %s, putting %s", filename, filedest)
		// the key for the distributed lookup
		key := sha1.Sum([]byte(filename))

		// figure out where to connect to
		st, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, &peerKey, privateKey)
		if err != nil {
			log.Printf("ERR: %v", err)
		}

		// serialize our get successor request
		var idBuf = new(bytes.Buffer)
		enc := gob.NewEncoder(idBuf)
		enc.Encode(models.SuccessorRequest{
			models.Identifier(key),
		})
		resp, err := st.RoundTrip(&protocol.Request{
			Header: protocol.Header{
				Type: protocol.UserType,
				From: id,
				Key:  key,
			},
			Method: protocol.GetSuccessorMethod,
			Data:   idBuf.Bytes(),
		})
		if err != nil {
			log.Printf("Failed to round trip the successor request: %v", err)
			return
		}

		log.Printf("found node")

		// connect to that host for this file
		// pull node out of response, and connect to that host
		var node = models.Node{}
		dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
		err = dec.Decode(&node)
		if err != nil {
			log.Printf("Failed to deserialize the node data: %v", err)
			return
		}

		// figure out where to connect to
		t, err := protocol.NewTransport("tcp", peerAddr, protocol.UserType, id, node.PublicKey, privateKey)
		if err != nil {
			log.Printf("ERR: %v", err)
		}

		resp, err = t.RoundTrip(&protocol.Request{
			Header: protocol.Header{
				Type: protocol.UserType,
				From: id,
				Key:  key,
			},
			Method: protocol.GetFileMethod,
		})
		if err != nil {
			log.Printf("Failed to round trip the successor request: %v", err)
			return
		}
		if resp.Status == protocol.Error {
			log.Printf("failed to get resource requested.")
			return
		}

		err = ioutil.WriteFile(filedest, resp.Data, 0644)
		if err != nil {
			log.Println(err)
			return
		}
	}
}
