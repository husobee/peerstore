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

func NewRemoteNode(addr string) (ChordNode, error) {
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

func (rn *RemoteNode) GetSuccessor(id models.Identifier) (ChordNode, error) {
	// if connection is nil, create a new connection to the remote node
	if rn.transport == nil {
		var err error
		if rn.transport, err = protocol.NewTransport("tcp", rn.Addr); err != nil {
			// we had an error setting up our connection
			return nil, errors.Wrap(err, "failed creating transport: ")
		}
	}
	// send request to the remote
	resp, err := rn.transport.RoundTrip(&protocol.Request{
		Header: protocol.Header{
			Key: id,
		},
		Method: protocol.GetSuccessorMethod,
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed round trip: ")
	}

	// decode the response body into a node object
	var (
		buf  = bytes.NewBuffer(resp.Data)
		node = new(models.Node)
	)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&node); err != nil {
		errors.Wrap(err, "failure decoding successor response from body")
	}

	// create a remote node
	successor, err := NewRemoteNode(node.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating successor remote node: ")
	}

	return successor, nil
}

func (rn *RemoteNode) GetPredecessor(id models.Identifier) (ChordNode, error) {
	return rn, nil
}

func (rn *RemoteNode) GetFingerTable() (models.FingerTable, error) {
	return models.FingerTable{}, nil
}

func (rn *RemoteNode) SetSuccessor(ChordNode) error {
	return nil
}

func (rn *RemoteNode) SetPredecessor(ChordNode) error {
	return nil
}

func (rn *RemoteNode) UpdateFingerTable(index int, n models.Node) error {
	return nil
}
