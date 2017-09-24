package chord

import (
	"crypto/sha1"
	"encoding/hex"
	"sync"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
	"github.com/pkg/errors"
)

// LocalNode - Implementation of ChordNode which holds the
// datastructure representing a local chord node.
type LocalNode struct {
	*models.Node
	// first record in the finger table is the successor node
	fingerTable *models.FingerTable

	predecessor      models.Node
	predecessorMutex *sync.RWMutex
}

// NewLocalNode - Creation of the new local node
func NewLocalNode(addr string, peer models.Node) (*LocalNode, error) {
	// make a new finger table for this node

	var fingerTable = models.NewFingerTable()
	fingerTable.SetIth(1, models.Interval{}, models.Node{
		Addr: addr,
		ID: models.Identifier(
			sha1.Sum([]byte(addr)),
		),
	})

	// create a new local node
	ln := &LocalNode{
		&models.Node{
			Addr: addr,
			ID: models.Identifier(
				sha1.Sum([]byte(addr)),
			),
		},
		fingerTable,
		models.Node{}, new(sync.RWMutex),
	}

	// run the initialization process
	err := ln.Initialize(peer)
	return ln, err
}

// ToNode - Convert LocalNode to just a plain Node
func (ln LocalNode) ToNode() models.Node {
	n := models.Node{
		Addr: ln.Addr,
		ID:   ln.ID,
	}
	return n
}

// Stabilize - stabilize the chord ring, makes sure we are actually predecessor
func (ln *LocalNode) Stabilize() error {
	// call successor's predecessor function to see we we are still the predecessor
	finger, err := ln.fingerTable.GetIth(1)
	if err != nil {
		return errors.Wrap(err, "failed to get successor: ")
	}

	var (
		currentSuccessor = finger.Successor
		compare          = currentSuccessor.Compare(ln.ToNode())
	)

	if currentSuccessor.Addr == "" || compare == 0 {
		return errors.New("uninitialized successor addr")
	}

	successorRN, err := NewRemoteNode(currentSuccessor.Addr)
	if err != nil {
		glog.Infof("error creating new remote node for successor: %v\n", err)
		errors.Wrap(err, "error creating new remote node for successor: ")
	}

	currentSuccessorPredecessor, err := successorRN.GetPredecessor()
	glog.Infof("stabilize for id=%s, successor id=%s thinks id=%s is predecessor\n",
		hex.EncodeToString(ln.ID[:]),
		hex.EncodeToString(currentSuccessor.ID[:]),
		hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
	)

	if err != nil {
		glog.Infof("error setting new predecessor on remote node: %v\n", err)
		errors.Wrap(err, "error setting new predecessor on remote node:")
	}

	// check if the successor's predecessor is ourself
	if currentSuccessorPredecessor.Compare(ln.ToNode()) == 0 {
		// no update! We are still the predecessor
		glog.Infof("self is still predecessor of successor - %s == %s\n",
			hex.EncodeToString(ln.ID[:]),
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)
		return nil
	} else if currentSuccessorPredecessor.Compare(ln.ToNode()) == 1 {
		// successor's predecessor is bigger than us, we will set
		// currentSuccessorPredecessor to our successor, and it's predecessor to us
		glog.Infof("self is no longer predecessor of successor - %s < %s\n",
			hex.EncodeToString(ln.ID[:]),
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)

		ln.SetSuccessor(currentSuccessorPredecessor)
		glog.Infof("self is setting successor to id=%s\n",
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)

		newSuccessorRN, err := NewRemoteNode(currentSuccessorPredecessor.Addr)
		if err != nil {
			glog.Infof("error creating new remote node for new successor: %v\n", err)
			return errors.Wrap(err, "error creating new remote node for new  successor: ")
		}

		// set the new successor's predecessor to ourselves
		if err := newSuccessorRN.SetPredecessor(models.Node{
			Addr: ln.Addr, ID: ln.ID,
		}); err != nil {
			glog.Infof("error setting new successor's predecessor to self", err)
			return errors.Wrap(err, "error setting new successor's predecessor to self: ")
		}
		glog.Infof("self is setting predecessor of the new successor id=%s to self\n",
			hex.EncodeToString(ln.ID[:]),
		)
	} else {
		successorRN, err = NewRemoteNode(currentSuccessor.Addr)
		if err != nil {
			glog.Infof("error creating new remote node for successor: %v\n", err)
			errors.Wrap(err, "error creating new remote node for successor: ")
		}
		if err := successorRN.SetPredecessor(models.Node{
			Addr: ln.Addr, ID: ln.ID,
		}); err != nil {
			glog.Infof("error resetting successor's predecessor to self", err)
			return errors.Wrap(err, "error setting new successor's predecessor to self: ")
		}

		glog.Infof("stabilize for id=%s, corrected successor id=%s thinks id=%s is predecessor now\n",
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

// SetSuccessor - Set the successor for this local node, which is the 1st ith
// entry in the finger table
func (ln *LocalNode) SetSuccessor(node models.Node) error {
	return ln.fingerTable.SetIth(1, models.Interval{}, node)
}

// Initialize - initialize the chord node
func (ln *LocalNode) Initialize(peer models.Node) error {
	// create a new remote node and transport
	glog.Infof("this node: %s\n", ln.ToString())

	glog.Infof("initializing chord node against remote: %s\n", peer.ToString())

	rn, err := NewRemoteNode(peer.Addr)
	if err != nil {
		glog.Infof("failed initializing chord node against remote: %v\n", err)
		return errors.New(
			"failed to initialize, peer remote node creation unsuccessful")
	}

	// call successor on remote node with our ID to figure out our successor
	successor, err := rn.Successor(ln.ID)
	glog.Infof("recieved successor from remote: %s\n", successor.ToString())

	if err != nil {
		glog.Infof("failed initializing chord node against remote: %v\n", err)
		return errors.Wrap(err,
			"failed to initialize, could not get successor: ")
	}

	// update the first finger to include successor
	ln.SetSuccessor(successor)
	glog.Infof("finger table updated: %s\n", ln.fingerTable.ToString())

	// set the successor's new predecessor
	successorRN, err := NewRemoteNode(successor.Addr)
	if err != nil {
		glog.Infof("error creating new remote node for successor: %v\n", err)
		errors.Wrap(err, "error creating new remote node for successor: ")
	}

	if err := successorRN.SetPredecessor(models.Node{Addr: ln.Addr, ID: ln.ID}); err != nil {
		glog.Infof("error setting new predecessor on remote node: %v\n", err)
		errors.Wrap(err, "error setting new predecessor on remote node:")
	}

	return nil
}

// Successor - This is what this is all about, given an Key we will return
// the node that is responsible for that Key
func (ln *LocalNode) Successor(id models.Identifier) (models.Node, error) {
	// does the key fall within ln's ID and the first entry of the finger table
	// if the key is greater than ln.ID and less than ln.successor.ID, return
	// ln.successor

	finger, _ := ln.fingerTable.GetIth(1)
	if (ln.CompareID(id) == -1 || ln.CompareID(id) == 0) && finger.Successor.CompareID(id) == 1 {
		return finger.Successor, nil
	}
	// okay, we don't know who the successor is, lets ask the closest node
	// we can who that would be.
	var closest models.Node

	for i := 1; i < models.M; i++ {
		finger, _ := ln.fingerTable.GetIth(i)
		if finger.Successor.CompareID(id) == -1 || ln.CompareID(id) == 0 {
			// this is less than the successor for finger
			closest = finger.Successor
		} else {
			// this is more than successor, so we break here
			// and ask that node to find the successor for us
			break
		}
	}

	if closest.Addr == "" {
		// no closest addr, unsure if this node is the one, return our id
		return models.Node{ID: ln.ID, Addr: ln.Addr}, nil
	}

	rn, err := NewRemoteNode(closest.Addr)
	if err != nil {
		return models.Node{}, errors.Wrap(err, "failure creating new remote node: ")
	}

	node, err := rn.Successor(id)
	glog.Infof("contacting node: %v\n", closest)
	glog.Infof("recieved successor from remote rpc call: %v\n", node)
	if err != nil {
		errors.Wrap(err, "failure getting successor from remote node: ")
	}

	return node, nil
}

// GetPredecessor - Get the predecessor node for this local node
func (ln *LocalNode) GetPredecessor() (models.Node, error) {
	ln.predecessorMutex.RLock()
	defer ln.predecessorMutex.RUnlock()
	glog.Infof("get predescessor currently set to: %s\n", ln.predecessor.ToString())
	return ln.predecessor, nil
}

// SetPredecessor - Set the predecessor node for this local node
func (ln *LocalNode) SetPredecessor(n models.Node) error {
	ln.predecessorMutex.Lock()
	defer ln.predecessorMutex.Unlock()
	ln.predecessor = n
	glog.Infof("predescessor set to: %s\n", ln.predecessor.ToString())
	return nil
}
