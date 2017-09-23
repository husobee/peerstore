package chord

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
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
	ln := &LocalNode{
		&models.Node{
			Addr: addr,
			ID: models.Identifier(
				sha1.Sum([]byte(addr)),
			),
		},
		models.FingerTable{
			// construction, first node in finger table (successor) is self
			models.Node{
				Addr: addr,
				ID: models.Identifier(
					sha1.Sum([]byte(addr)),
				),
			},
		},
		models.Node{}, new(sync.RWMutex), new(sync.RWMutex),
	}
	err := ln.Initialize(peer)
	return ln, err
}

// Stabilize - stabilize the chord ring, makes sure we are actually predecessor
func (ln *LocalNode) Stabilize() error {
	// call successor's predecessor function to see we we are still the predecessor

	ln.fingerTableMutex.RLock()
	currentSuccessor := ln.fingerTable[0]
	ln.fingerTableMutex.RUnlock()

	if currentSuccessor.Addr == "" {
		return errors.New("uninitialized successor addr")
	}
	if bytes.Compare(currentSuccessor.ID[:], ln.ID[:]) == 0 {
		return errors.New("uninitialized successor id")
	}

	successorRN, err := NewRemoteNode(currentSuccessor.Addr)
	if err != nil {
		log.Printf("error creating new remote node for successor: %v\n", err)
		errors.Wrap(err, "error creating new remote node for successor: ")
	}

	currentSuccessorPredecessor, err := successorRN.GetPredecessor()
	log.Printf("stabilize for id=%s, successor id=%s thinks id=%s is predecessor\n",
		hex.EncodeToString(ln.ID[:]),
		hex.EncodeToString(currentSuccessor.ID[:]),
		hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
	)

	if err != nil {
		log.Printf("error setting new predecessor on remote node: %v\n", err)
		errors.Wrap(err, "error setting new predecessor on remote node:")
	}

	// check if the successor's predecessor is ourself
	if bytes.Compare(currentSuccessorPredecessor.ID[:], ln.ID[:]) == 0 {
		// no update! We are still the predecessor
		log.Printf("self is still predecessor of successor - %s == %s\n",
			hex.EncodeToString(ln.ID[:]),
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)
		return nil
	} else if bytes.Compare(currentSuccessorPredecessor.ID[:], ln.ID[:]) == 1 {
		// successor's predecessor is bigger than us, we will set
		// currentSuccessorPredecessor to our successor, and it's predecessor to us
		log.Printf("self is no longer predecessor of successor - %s < %s\n",
			hex.EncodeToString(ln.ID[:]),
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)

		ln.SetSuccessor(currentSuccessorPredecessor)
		log.Printf("self is setting successor to id=%s\n",
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)

		newSuccessorRN, err := NewRemoteNode(currentSuccessorPredecessor.Addr)
		if err != nil {
			log.Printf("error creating new remote node for new successor: %v\n", err)
			return errors.Wrap(err, "error creating new remote node for new  successor: ")
		}

		// set the new successor's predecessor to ourselves
		if err := newSuccessorRN.SetPredecessor(models.Node{
			Addr: ln.Addr, ID: ln.ID,
		}); err != nil {
			log.Printf("error setting new successor's predecessor to self", err)
			return errors.Wrap(err, "error setting new successor's predecessor to self: ")
		}
		log.Printf("self is setting predecessor of the new successor id=%s to self\n",
			hex.EncodeToString(ln.ID[:]),
		)
	} else {
		successorRN, err = NewRemoteNode(currentSuccessor.Addr)
		if err != nil {
			log.Printf("error creating new remote node for successor: %v\n", err)
			errors.Wrap(err, "error creating new remote node for successor: ")
		}
		if err := successorRN.SetPredecessor(models.Node{
			Addr: ln.Addr, ID: ln.ID,
		}); err != nil {
			log.Printf("error resetting successor's predecessor to self", err)
			return errors.Wrap(err, "error setting new successor's predecessor to self: ")
		}

		log.Printf("stabilize for id=%s, corrected successor id=%s thinks id=%s is predecessor now\n",
			hex.EncodeToString(ln.ID[:]),
			hex.EncodeToString(currentSuccessor.ID[:]),
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)
	}
	// if not, set this node's successor to the predecessor, and update the new
	// successor with us as it's predecessor

	// TODO: migrate values we are now in charge of!
	return nil
}

func (ln *LocalNode) SetSuccessor(node models.Node) error {
	ln.fingerTableMutex.Lock()
	defer ln.fingerTableMutex.Unlock()
	if ln.fingerTable == nil {
		ln.fingerTable = models.FingerTable{
			node,
		}
	}
	if len(ln.fingerTable) > 0 {
		ln.fingerTable[0] = node
	}
	return nil
}

// Initialize - initialize the chord node
func (ln *LocalNode) Initialize(peer models.Node) error {
	// create a new remote node and transport
	log.Printf("this node: addr= %s, id=%s\n", ln.Addr,
		hex.EncodeToString(ln.ID[:]))

	log.Printf("initializing chord node against remote: addr=%s, id=%s\n",
		peer.Addr,
		hex.EncodeToString(peer.ID[:]))

	rn, err := NewRemoteNode(peer.Addr)
	if err != nil {
		log.Printf("failed initializing chord node against remote: %v\n", err)
		return errors.New(
			"failed to initialize, peer remote node creation unsuccessful")
	}

	// call successor on remote node with our ID to figure out our successor
	successor, err := rn.Successor(ln.ID)
	log.Printf("recieved successor from remote: addr=%s, id=%s\n",
		successor.Addr,
		hex.EncodeToString(successor.ID[:]))

	if err != nil {
		log.Printf("failed initializing chord node against remote: %v\n", err)
		return errors.Wrap(err,
			"failed to initialize, could not get successor: ")
	}

	// update the first finger to include successor
	ln.SetSuccessor(successor)

	ln.fingerTableMutex.RLock()
	var fmtFingerTable = "["
	for _, v := range ln.fingerTable {
		fmtFingerTable = fmt.Sprintf("%s{addr=%s, id=%s}, ",
			fmtFingerTable, v.Addr,
			hex.EncodeToString(v.ID[:]))
	}
	fmtFingerTable += "]"

	log.Printf("finger table updated: %s\n", fmtFingerTable)

	ln.fingerTableMutex.RUnlock()

	// set the successor's new predecessor
	successorRN, err := NewRemoteNode(successor.Addr)
	if err != nil {
		log.Printf("error creating new remote node for successor: %v\n", err)
		errors.Wrap(err, "error creating new remote node for successor: ")
	}

	if err := successorRN.SetPredecessor(models.Node{Addr: ln.Addr, ID: ln.ID}); err != nil {
		log.Printf("error setting new predecessor on remote node: %v\n", err)
		errors.Wrap(err, "error setting new predecessor on remote node:")
	}

	return nil
}

func (ln *LocalNode) Successor(id models.Identifier) (models.Node, error) {
	ln.fingerTableMutex.RLock()
	// if we dont have a finger table, or less than 1 entry, throw error
	if ln.fingerTable == nil || len(ln.fingerTable) < 1 {
		return models.Node{}, errors.New("empty finger table, not initialized")
	}
	ln.fingerTableMutex.RUnlock()
	// does the key fall within ln's ID and the first entry of the finger table
	// if the key is greater than ln.ID and less than ln.successor.ID, return
	// ln.successor
	if (bytes.Compare(ln.ID[:], id[:]) == -1 || bytes.Compare(ln.ID[:], id[:]) == 0) && bytes.Compare(ln.fingerTable[0].ID[:], id[:]) == 1 {
		// return ln.successor in the form of a RemoteNode
		ln.fingerTableMutex.RLock()
		defer ln.fingerTableMutex.RUnlock()
		return ln.fingerTable[0], nil
	}
	// okay, we don't know who the successor is, lets ask the closest node
	// we can who that would be.
	ln.fingerTableMutex.RLock()
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
	ln.fingerTableMutex.RUnlock()

	if closest.Addr == "" {
		// no closest addr, unsure if this node is the one, return our id
		return models.Node{ID: ln.ID, Addr: ln.Addr}, nil
	}

	rn, err := NewRemoteNode(closest.Addr)
	if err != nil {
		return models.Node{}, errors.Wrap(err, "failure creating new remote node: ")
	}

	node, err := rn.Successor(id)
	log.Printf("contacting node: %v\n", closest)
	log.Printf("recieved successor from remote rpc call: %v\n", node)
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

func (ln *LocalNode) GetPredecessor() (models.Node, error) {
	ln.predecessorMutex.RLock()
	defer ln.predecessorMutex.RUnlock()
	log.Printf("get predescessor currently set to : addr=%s, id=%s\n",
		ln.predecessor.Addr,
		hex.EncodeToString(ln.predecessor.ID[:]),
	)
	return ln.predecessor, nil
}

func (ln *LocalNode) SetPredecessor(n models.Node) error {
	ln.predecessorMutex.Lock()
	defer ln.predecessorMutex.Unlock()
	ln.predecessor = n
	log.Printf("predescessor set to : addr=%s, id=%s\n",
		ln.predecessor.Addr,
		hex.EncodeToString(ln.predecessor.ID[:]),
	)
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
