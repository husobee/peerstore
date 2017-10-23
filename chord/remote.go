package chord

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/gob"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
	"github.com/pkg/errors"
)

// RemoteNode - Implementation of ChordNode which holds the
// datastructure representing a remote chord node.
// remote nodes will be queried for finger table, predecessor and
// successsors via PS protocol
type RemoteNode struct {
	*models.Node
	transport *protocol.Transport
}

// NewRemoteNode - create a new remote node, which implements ChordNode, wherein
// we are able to perform queries on this node
func NewRemoteNode(addr string, key *rsa.PublicKey) (*RemoteNode, error) {
	return &RemoteNode{
		&models.Node{
			Addr:      addr,
			PublicKey: key,
			ID: models.Identifier(
				sha1.Sum([]byte(addr)),
			),
		},
		nil,
	}, nil
}

// GetPredecessor - Get the predecessor of a remote node
func (rn *RemoteNode) GetPredecessor(key *rsa.PrivateKey) (models.Node, error) {
	// if connection is nil, create a new connection to the remote node

	if rn.PublicKey == nil {
		glog.Infof("HERE THE RN HAS NO PUBLIC KEY")
	}

	if rn.transport == nil {
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr, protocol.NodeType, rn.ID, rn.PublicKey, key); err != nil {
			// we had an error setting up our connection
			return models.Node{}, errors.Wrap(err, "failed creating transport: ")
		}
	}
	// send request to the remote
	resp, err := rn.transport.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			From:     rn.ID,
			FromAddr: rn.Addr,
			Type:     protocol.NodeType,
			PubKey:   rn.PublicKey,
		},
		Method: protocol.GetPredecessorMethod,
	})
	rn.transport.Close()

	if err != nil {
		return models.Node{}, errors.Wrap(err, "failed round trip: ")
	}

	// decode the response body into a node object
	var (
		buf  = bytes.NewBuffer(resp.Data)
		node = models.Node{}
	)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&node); err != nil {
		errors.Wrap(err, "failure decoding successor response from body")
	}

	return node, nil

}

// Successor - Call successor on
func (rn *RemoteNode) Successor(id models.Identifier, key *rsa.PrivateKey) (models.Node, error) {
	// if connection is nil, create a new connection to the remote node
	if rn.transport == nil {
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr, protocol.NodeType, rn.ID, rn.PublicKey, key); err != nil {
			// we had an error setting up our connection
			return models.Node{}, errors.Wrap(err, "failed creating transport: ")
		}
	}

	var reqBuffer = new(bytes.Buffer)

	enc := gob.NewEncoder(reqBuffer)
	if err := enc.Encode(models.SuccessorRequest{id}); err != nil {
		return models.Node{}, errors.Wrap(err, "failed to encode request: ")
	}

	glog.Infof("rn.PublicKey is %v", rn.PublicKey)
	// send request to the remote
	resp, err := rn.transport.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			From:     rn.ID,
			FromAddr: rn.Addr,
			Type:     protocol.NodeType,
			PubKey:   rn.PublicKey,
		},
		Method: protocol.GetSuccessorMethod,
		Data:   reqBuffer.Bytes(),
	})
	rn.transport.Close()

	glog.Info("DO I GET HERE??????????????????????")

	if err != nil {
		return models.Node{}, errors.Wrap(err, "failed round trip: ")
	}

	// decode the response body into a node object
	var (
		buf  = bytes.NewBuffer(resp.Data)
		node = models.Node{}
	)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&node); err != nil {
		errors.Wrap(err, "failure decoding successor response from body")
	}

	return node, nil
}

// SetPredecessor - set the predecessor on a remote node to node
func (rn *RemoteNode) SetPredecessor(node models.Node, key *rsa.PrivateKey) error {
	// if connection is nil, create a new connection to the remote node
	if rn.transport == nil {
		glog.Infof("setting up transport for set pred call: %v", node)
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr, protocol.NodeType, rn.ID, rn.PublicKey, key); err != nil {
			// we had an error setting up our connection
			return errors.Wrap(err, "failed creating transport: ")
		}
	}

	var reqBuffer = new(bytes.Buffer)

	enc := gob.NewEncoder(reqBuffer)
	if err := enc.Encode(node); err != nil {
		return errors.Wrap(err, "failed to encode request: ")
	}

	// send request to the remote
	_, err := rn.transport.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			From:     rn.ID,
			FromAddr: rn.Addr,
			Type:     protocol.NodeType,
			PubKey:   rn.PublicKey,
		},
		Method: protocol.SetPredecessorMethod,
		Data:   reqBuffer.Bytes(),
	})

	rn.transport.Close()

	if err != nil {
		return errors.Wrap(err, "failed round trip: ")
	}

	return nil
}
