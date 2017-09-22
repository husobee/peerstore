package chord

import (
	"bytes"
	"crypto/sha1"
	"sync"

	"github.com/husobee/peerstore/models"
	"github.com/pkg/errors"
)

// LocalNode - Implementation of ChordNode which holds the
// datastructure representing a local chord node.
type LocalNode struct {
	*models.Node
	// first record in the finger table is the successor node
	fingerTable models.FingerTable
	predecessor models.Node
	// fingerTableMutex - a locking mechanism around the finger table
	fingerTableMutex *sync.RWMutex
	predecessorMutex *sync.RWMutex
}

func NewLocalNode(addr string, peer models.Node) (*LocalNode, error) {
	// TODO: need to start up server
	// TODO: need to start initialize step with the peer
	return &LocalNode{
		&models.Node{
			Addr: addr,
			ID: models.Identifier(
				sha1.Sum([]byte(addr)),
			),
		},
		models.FingerTable{},
		models.Node{}, new(sync.RWMutex), new(sync.RWMutex),
	}, nil
}

func (ln *LocalNode) Successor(id models.Identifier) (models.Node, error) {
	ln.fingerTableMutex.RLock()
	defer ln.fingerTableMutex.RUnlock()

	// if we dont have a finger table, or less than 1 entry, throw error
	if ln.fingerTable == nil || len(ln.fingerTable) < 1 {
		return models.Node{}, errors.New("empty finger table, not initialized")
	}
	// does the key fall within ln's ID and the first entry of the finger table
	// if the key is greater than ln.ID and less than ln.successor.ID, return
	// ln.successor
	if (bytes.Compare(ln.ID[:], id[:]) == -1 || bytes.Compare(ln.ID[:], id[:]) == 0) && bytes.Compare(ln.fingerTable[0].ID[:], id[:]) == 1 {
		// return ln.successor in the form of a RemoteNode
		return ln.fingerTable[0], nil
	}
	// okay, we don't know who the successor is, lets ask the closest node
	// we can who that would be.
	var closest models.Node
	for _, finger := range ln.fingerTable {
		if bytes.Compare(finger.ID[:], id[:]) == -1 || bytes.Compare(ln.ID[:], id[:]) == 0 {
			// this is less than the successor for finger
			closest = finger
		} else {
			// this is more than successor, so we break here
			// and ask that node to find the successor for us
			break
		}
	}
	rn, err := NewRemoteNode(closest.Addr)
	if err != nil {
		return models.Node{}, errors.Wrap(err, "failure creating new remote node: ")
	}

	node, err := rn.Successor(id)
	if err != nil {
		errors.Wrap(err, "failure getting successor from remote node: ")
	}

	return node, nil
}

func (ln *LocalNode) FingerTable() (models.FingerTable, error) {
	ln.fingerTableMutex.RLock()
	defer ln.fingerTableMutex.RUnlock()
	return ln.fingerTable, nil
}

func (ln *LocalNode) SetPredecessor(n models.Node) error {
	ln.predecessor = n
	return nil
}

func (ln *LocalNode) UpdateFingerTable(index int, n models.Node) error {
	if index > MaxFingerTableSize {
		return errors.New("finger table index is greater than max")
	}
	if ln.fingerTable == nil {
		ln.fingerTable = models.FingerTable{}
	}
	if len(ln.fingerTable) < index+1 {
		// finger table is too small, expand
		for i := len(ln.fingerTable); i < index+1; i++ {
			ln.fingerTable = append(ln.fingerTable, models.Node{})
		}
	}
	ln.fingerTable[index] = n
	return nil
}
