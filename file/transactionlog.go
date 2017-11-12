package file

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/gob"
	"log"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/crypto"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
	"github.com/pkg/errors"
)

func GetTransactionLog(thisID models.Identifier, peer models.Node, userKey *rsa.PublicKey, selfKey *rsa.PrivateKey) (models.TransactionLog, error) {
	gobKey, _ := crypto.GobEncodePublicKey(userKey)
	id := models.Identifier(sha1.Sum(append(gobKey, []byte("-transaction-log")...)))

	// create a connection to our peer
	t, err := protocol.NewTransport("tcp", peer.Addr, protocol.NodeType, id, peer.PublicKey, selfKey)
	if err != nil {
		glog.Error("ERR: %v", err)
	}
	defer t.Close()

	var buf = new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	// Perform a Successor Request to our peer
	enc.Encode(models.SuccessorRequest{
		models.Identifier(id),
	})
	resp, err := t.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			Type: protocol.NodeType,
			From: thisID,
			Key:  id,
		},
		Method: protocol.GetSuccessorMethod,
		Data:   buf.Bytes(),
	})
	if err != nil {
		glog.Info("Failed to round trip the successor request: %v", err)
		return models.TransactionLog{}, errors.Wrap(err, "failed to get successor: ")
	}

	// populate our peer to get the log
	var node = models.Node{}
	dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
	err = dec.Decode(&node)
	if err != nil {
		glog.Error("Failed to deserialize the node data: %v", err)
		return models.TransactionLog{}, errors.Wrap(err, "failed deserialize successor: ")
	}

	glog.Info("Peer holding TransactionLog: %s", node.ToString())

	// now connect to the node holding the transaction log
	st, err := protocol.NewTransport("tcp", peer.Addr, protocol.NodeType, thisID, node.PublicKey, selfKey)
	if err != nil {
		log.Printf("ERR: %v", err)
	}
	defer st.Close()
	resp, err = st.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			Type: protocol.NodeType,
			From: thisID,
			Key:  id,
		},
		Method: protocol.GetFileMethod,
	})
	if err != nil {
		log.Printf("Failed to round trip the get file request: %v", err)
		return models.TransactionLog{}, errors.Wrap(err, "failed to get file")
	}

	if resp.Status == protocol.Error {
		log.Printf("failed to get resource requested.")
		return models.TransactionLog{}, errors.Wrap(err, "failed to get file, protocol error")
	}

	var transactionLog = models.TransactionLog{}
	dec = gob.NewDecoder(bytes.NewBuffer(resp.Data))
	err = dec.Decode(&transactionLog)
	if err != nil {
		glog.Error("Failed to deserialize the transactionLog data: %v", err)
		return models.TransactionLog{}, errors.Wrap(err, "failed deserialize transaction log: ")
	}

	return transactionLog, nil
}

func PutTransactionLog(thisID models.Identifier, peer models.Node, userKey *rsa.PublicKey, selfKey *rsa.PrivateKey, transactionLog models.TransactionLog) error {
	gobKey, _ := crypto.GobEncodePublicKey(userKey)
	glog.Infof("userKey bytes: %x", userKey)
	glog.Infof("gobKey bytes: %x", gobKey)
	id := models.Identifier(sha1.Sum(append(gobKey, []byte("-transaction-log")...)))

	glog.Infof("Trying to PUT Transaction LOG, ID: %x", id)

	// create a connection to our peer
	t, err := protocol.NewTransport("tcp", peer.Addr, protocol.NodeType, id, peer.PublicKey, selfKey)
	if err != nil {
		glog.Error("ERR: %v", err)
	}

	var buf = new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	// Perform a Successor Request to our peer
	enc.Encode(models.SuccessorRequest{
		models.Identifier(id),
	})
	resp, err := t.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			Type: protocol.NodeType,
			From: thisID,
			Key:  id,
		},
		Method: protocol.GetSuccessorMethod,
		Data:   buf.Bytes(),
	})
	if err != nil {
		glog.Info("Failed to round trip the successor request: %v", err)
		return errors.Wrap(err, "failed to get successor: ")
	}
	// populate our peer to get the log
	var node = models.Node{}
	dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
	err = dec.Decode(&node)
	if err != nil {
		glog.Error("Failed to deserialize the node data: %v", err)
		return errors.Wrap(err, "failed deserialize successor: ")
	}

	glog.Info("Peer holding TransactionLog: %s", node.ToString())

	// encode the transaction log, and put to our node
	var logBuf = bytes.NewBuffer([]byte{})
	enc = gob.NewEncoder(logBuf)
	err = enc.Encode(&transactionLog)
	if err != nil {
		glog.Error("Failed to serialize the transactionLog data: %v", err)
		return errors.Wrap(err, "failed serialize transaction log: ")
	}

	// figure out where to connect to
	st, err := protocol.NewTransport("tcp", node.Addr, protocol.NodeType, id, node.PublicKey, selfKey)
	if err != nil {
		glog.Error("ERR: %v", err)
		return errors.Wrap(err, "failed serialize transaction log: ")
	}

	// send the file over
	glog.Info("starting request: ", protocol.PostFileMethod)
	request := &protocol.Request{
		Header: protocol.Header{
			Key:        id,
			Type:       protocol.NodeType,
			From:       thisID,
			DataLength: uint64(len(logBuf.Bytes())),
			PubKey:     selfKey.Public().(*rsa.PublicKey),
		},
		Method: protocol.PostFileMethod,
		Data:   logBuf.Bytes(),
	}
	glog.Info("!!!!!!!!!!!!!!!!! PUT TRANSACTION LOG !!!!!!!!!!!! Request: %+v\n", request)

	response, err := t.RoundTrip(request)
	if err != nil {
		glog.Error("ERR: %v\n", err)
		return errors.Wrap(err, "failed serialize transaction log: ")
	}
	glog.Info("!!!!!!!!!!!!!!!!! PUT TRANSACTION LOG !!!!!!!!!!!! Response: %+v\n", response)

	st.Close()
	return nil

}
