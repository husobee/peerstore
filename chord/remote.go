package chord

import (
	"bytes"
	"crypto/sha1"
	"encoding/gob"

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
func NewRemoteNode(addr string) (*RemoteNode, error) {
	return &RemoteNode{
		&models.Node{
			Addr: addr,
			ID: models.Identifier(
				sha1.Sum([]byte(addr)),
			),
		},
		nil,
	}, nil
}

// GetPredecessor - Get the predecessor of a remote node
func (rn *RemoteNode) GetPredecessor() (models.Node, error) {
	// if connection is nil, create a new connection to the remote node
	if rn.transport == nil {
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr); err != nil {
			// we had an error setting up our connection
			return models.Node{}, errors.Wrap(err, "failed creating transport: ")
		}
	}
	// send request to the remote
	resp, err := rn.transport.RoundTrip(&protocol.Request{
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
func (rn *RemoteNode) Successor(id models.Identifier) (models.Node, error) {
	// if connection is nil, create a new connection to the remote node
	if rn.transport == nil {
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr); err != nil {
			// we had an error setting up our connection
			return models.Node{}, errors.Wrap(err, "failed creating transport: ")
		}
	}

	var reqBuffer = new(bytes.Buffer)

	enc := gob.NewEncoder(reqBuffer)
	if err := enc.Encode(SuccessorRequest{id}); err != nil {
		return models.Node{}, errors.Wrap(err, "failed to encode request: ")
	}

	// send request to the remote
	resp, err := rn.transport.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			Key: id,
		},
		Method: protocol.GetSuccessorMethod,
		Data:   reqBuffer.Bytes(),
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

// SetPredecessor - set the predecessor on a remote node to node
func (rn *RemoteNode) SetPredecessor(node models.Node) error {
	// if connection is nil, create a new connection to the remote node
	if rn.transport == nil {
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr); err != nil {
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
			Key: node.ID,
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
