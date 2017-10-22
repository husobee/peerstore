package chord

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/hex"
	"sync"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
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
	server           *protocol.Server
}

// NewLocalNode - Creation of the new local node
func NewLocalNode(s *protocol.Server, addr string, peer models.Node) (*LocalNode, error) {
	// make a new finger table for this node
	n := models.Node{
		Addr: addr,
		ID: models.Identifier(
			sha1.Sum([]byte(addr)),
		),
		PublicKey: s.PrivateKey.Public().(*rsa.PublicKey),
	}

	var (
		fingerTable = models.NewFingerTable()
		err         error
	)
	// set initial finger table to have self for the whole range
	ln := &LocalNode{
		&n, fingerTable, models.Node{}, new(sync.RWMutex), s,
	}
	fingerTable.SetIth(1, models.NewInterval(n, n), n, ln.ToNode())
	glog.Infof("bootstrapping fingertable: %s", fingerTable.ToString())
	// create a new local node

	if &peer != nil {
		// run the initialization process
		err = ln.Initialize(peer)
	}
	return ln, err
}

// UserRegistrationHandler - this handler handles all user registrations.  A user
// registration consists of the user giving the server it's public key, and the
// server signing that key and placing it within the DHT
func (ln *LocalNode) UserRegistrationHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	// store the user's public key in the DHT
	return protocol.Response{}
}

// ToNode - Convert LocalNode to just a plain Node
func (ln LocalNode) ToNode() models.Node {
	return models.Node{
		Addr:      ln.Addr,
		ID:        ln.ID,
		PublicKey: ln.server.PrivateKey.Public().(*rsa.PublicKey),
	}
}

// Stabilize - stabilize the chord ring, makes sure we are actually predecessor
func (ln *LocalNode) Stabilize() error {
	// call successor's predecessor function to see we we are still the predecessor
	currentSuccessor, err := ln.Successor(ln.ID)
	if err != nil {
		glog.Infof("failed to get successor for stabilize: %s", err)
		return errors.Wrap(err, "failed to get successor: ")
	}

	glog.Infof("current successor: %s, this: %s", currentSuccessor.ToString(), ln.ToString())

	// we need to close the loop if successor of this node is nil
	// if successor is nil, ask every pred in the chain if they have a nil pred
	// when we find the end of the line, set our successor to that node, and that node's pred
	// to us thereby closing the ring
	if ln.Compare(currentSuccessor) == 0 {
		// if no successor, traverse back through chain and make node with no predecessor
		// the successor
		newPredecessor, _ := ln.GetPredecessor()
		if newPredecessor.Addr != "" {
			// if the first entry in finger is ourself, then we dont have a successor
			predecessor := newPredecessor
			glog.Infof("--------------here is predecessor: %s", predecessor)
			for {
				// lookup the public key for the predecessor
				predecessorRN, err := NewRemoteNode(predecessor.Addr, predecessor.PublicKey)
				if err != nil {
					glog.Infof("error creating new remote node for predecessor: %v\n", err)
					break
				}

				nextPredecessor, err := predecessorRN.GetPredecessor(ln.server.PrivateKey)
				if err != nil {
					glog.Infof("error getting new predecessor on remote node: %v\n", err)
					break
				}

				if nextPredecessor.Addr == "" {
					// predecessor was our last one in the chain
					ln.SetSuccessor(predecessor)
					newPredecessorRN, err := NewRemoteNode(predecessor.Addr, predecessor.PublicKey)
					if err != nil {
						glog.Infof("error creating new remote node for predecessor: %v\n", err)
						break
					}
					newPredecessorRN.SetPredecessor(ln.ToNode(), ln.server.PrivateKey)
					break
				}
				predecessor = nextPredecessor
			}
		}
	} else {

		successorRN, err := NewRemoteNode(currentSuccessor.Addr, currentSuccessor.PublicKey)
		if err != nil {
			glog.Infof("error creating new remote node for successor: %v\n", err)
			return errors.Wrap(err, "error creating new remote node for successor: ")
		}

		currentSuccessorPredecessor, err := successorRN.GetPredecessor(ln.server.PrivateKey)
		glog.Infof("stabilize for id=%s, successor id=%s thinks id=%s is predecessor\n",
			hex.EncodeToString(ln.ID[:]),
			hex.EncodeToString(currentSuccessor.ID[:]),
			hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
		)

		if err != nil {
			glog.Infof("error setting new predecessor on remote node: %v\n", err)
			return errors.Wrap(err, "error setting new predecessor on remote node:")
		}

		lnID := models.KeyToID(ln.ID)
		succPredID := models.KeyToID(currentSuccessorPredecessor.ID)
		succID := models.KeyToID(currentSuccessor.ID)

		if succPredID == lnID {
			// no update! We are still the predecessor
			glog.Infof("self is still predecessor of successor - %s == %s\n",
				hex.EncodeToString(ln.ID[:]),
				hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
			)
			return nil
		}

		if succPredID < succID {
			if succPredID < lnID && lnID < succID {
				// change the successor to use ln as predecessor
				successorRN, err = NewRemoteNode(currentSuccessor.Addr, currentSuccessor.PublicKey)
				if err != nil {
					glog.Infof("error creating new remote node for successor: %v\n", err)
					errors.Wrap(err, "error creating new remote node for successor: ")
				}
				glog.Info("!!! succPredID < lnID -> CORRECTING HERE")
				if err := successorRN.SetPredecessor(ln.ToNode(), ln.server.PrivateKey); err != nil {
					glog.Infof("error resetting successor's predecessor to self", err)
					return errors.Wrap(err, "error setting new successor's predecessor to self: ")
				}

				glog.Infof("stabilize for id=%s, corrected successor id=%s thinks id=%s is predecessor now\n",
					hex.EncodeToString(ln.ID[:]),
					hex.EncodeToString(currentSuccessor.ID[:]),
					hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
				)
			} else {
				// change ln successor to new successor pred
				glog.Infof("self is no longer predecessor of successor - %s < %s\n",
					hex.EncodeToString(ln.ID[:]),
					hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
				)

				ln.SetSuccessor(currentSuccessorPredecessor)
				glog.Infof("self is setting successor to id=%s\n",
					hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
				)

				newSuccessorRN, err := NewRemoteNode(currentSuccessorPredecessor.Addr, currentSuccessorPredecessor.PublicKey)
				if err != nil {
					glog.Infof("error creating new remote node for new successor: %v\n", err)
					return errors.Wrap(err, "error creating new remote node for new  successor: ")
				}

				// set the new successor's predecessor to ourselves
				if err := newSuccessorRN.SetPredecessor(ln.ToNode(), ln.server.PrivateKey); err != nil {
					glog.Infof("error setting new successor's predecessor to self", err)
					return errors.Wrap(err, "error setting new successor's predecessor to self: ")
				}
				glog.Infof("self is setting predecessor of the new successor id=%s to self\n",
					hex.EncodeToString(ln.ID[:]),
				)
			}
		} else {
			if succPredID < lnID || lnID < succID {
				// use new succ pred
				// change ln successor to new successor pred
				glog.Infof("self is no longer predecessor of successor - %s < %s\n",
					hex.EncodeToString(ln.ID[:]),
					hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
				)

				glog.Infof("!!! lnID < succPredID && succPredID < succID : %d < %d && %d < %d",
					lnID, succPredID, succPredID, succID,
				)
				ln.SetSuccessor(currentSuccessorPredecessor)
				glog.Infof("self is setting successor to id=%s\n",
					hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
				)

				newSuccessorRN, err := NewRemoteNode(currentSuccessorPredecessor.Addr, currentSuccessorPredecessor.PublicKey)
				if err != nil {
					glog.Infof("error creating new remote node for new successor: %v\n", err)
					return errors.Wrap(err, "error creating new remote node for new  successor: ")
				}

				// set the new successor's predecessor to ourselves
				if err := newSuccessorRN.SetPredecessor(ln.ToNode(), ln.server.PrivateKey); err != nil {
					glog.Infof("error setting new successor's predecessor to self", err)
					return errors.Wrap(err, "error setting new successor's predecessor to self: ")
				}
				glog.Infof("self is setting predecessor of the new successor id=%s to self\n",
					hex.EncodeToString(ln.ID[:]),
				)

			} else {
				// change the successor to use ln as predecessor
				successorRN, err = NewRemoteNode(currentSuccessor.Addr, currentSuccessor.PublicKey)
				if err != nil {
					glog.Infof("error creating new remote node for successor: %v\n", err)
					return errors.Wrap(err, "error creating new remote node for successor: ")
				}
				glog.Info("!!!CORRECTING HERE")
				glog.Infof("!!! lnID < succPredID && succPredID < succID : %d < %d && %d < %d",
					lnID, succPredID, succPredID, succID,
				)
				if err := successorRN.SetPredecessor(ln.ToNode(), ln.server.PrivateKey); err != nil {
					glog.Infof("error resetting successor's predecessor to self", err)
					return errors.Wrap(err, "error setting new successor's predecessor to self: ")
				}

				glog.Infof("stabilize for id=%s, corrected successor id=%s thinks id=%s is predecessor now\n",
					hex.EncodeToString(ln.ID[:]),
					hex.EncodeToString(currentSuccessor.ID[:]),
					hex.EncodeToString(currentSuccessorPredecessor.ID[:]),
				)
			}
		}
	}

	// if not, set this node's successor to the predecessor, and update the new
	// successor with us as it's predecessor

	// TODO: migrate values we are now in charge of!
	return nil
}

// SetSuccessor - Set the successor for this local node, which is the 1st ith
// entry in the finger table
func (ln *LocalNode) SetSuccessor(node models.Node) error {
	return ln.fingerTable.SetIth(1, models.NewInterval(ln.ToNode(), node), node, ln.ToNode())
}

// Initialize - initialize the chord node
func (ln *LocalNode) Initialize(peer models.Node) error {
	// create a new remote node and transport
	glog.Infof("this node: %s\n", ln.ToString())
	glog.Infof("initial finger table: %s\n", ln.fingerTable.ToString())

	glog.Infof("initializing chord node against remote: %s\n", peer.ToString())

	rn, err := NewRemoteNode(peer.Addr, peer.PublicKey)
	if err != nil {
		glog.Infof("failed initializing chord node against remote: %v\n", err)
		return errors.New(
			"failed to initialize, peer remote node creation unsuccessful")
	}

	// call successor on remote node with our ID to figure out our successor
	successor, err := rn.Successor(ln.ID, ln.server.PrivateKey)

	if err != nil {
		glog.Infof("failed initializing chord node against remote: %v\n", err)
		return errors.Wrap(err,
			"failed to initialize, could not get successor: ")
	}

	glog.Infof("recieved successor from remote: %s\n", successor.ToString())

	// update the first finger to include successor
	ln.SetSuccessor(successor)
	glog.Infof("finger table updated: %s\n", ln.fingerTable.ToString())

	// set the successor's new predecessor
	successorRN, err := NewRemoteNode(successor.Addr, successor.PublicKey)
	if err != nil {
		glog.Infof("error creating new remote node for successor: %v\n", err)
		errors.Wrap(err, "error creating new remote node for successor: ")
	}

	glog.Infof("!!! should only initialize once")
	if err := successorRN.SetPredecessor(ln.ToNode(), ln.server.PrivateKey); err != nil {
		glog.Infof("error setting new predecessor on remote node: %v\n", err)
		errors.Wrap(err, "error setting new predecessor on remote node:")
	}

	return nil
}

// ClosestPrecedingNode - Find the node that directly preceeds ID
// closest preceeding node
func (ln *LocalNode) ClosestPrecedingNode(id models.Identifier) (models.Node, error) {
	// convert hash to big int
	lnID := models.KeyToID(ln.ID)
	nID := models.KeyToID(id)

	for i := models.M; i >= 1; i-- {
		ith, err := ln.fingerTable.GetIth(uint64(i))
		ithSuccessorID := models.KeyToID(ith.Successor.ID)

		if err != nil {
			return models.Node{}, errors.Wrap(err, "failed to get closest preceding: ")
		}

		if ith.Successor.Addr == "" {
			continue
		}

		// if ith.Successor is within lnID and nID
		if lnID < nID {
			if lnID < ithSuccessorID && ithSuccessorID < nID {
				glog.Infof("closest preceding node to id=%s is %s", models.KeyToID(id), ith.Successor.ToString())
				return ith.Successor, nil
			}
		} else {
			if lnID < ithSuccessorID || ithSuccessorID < nID {
				glog.Infof("closest preceding node to id=%s is %s", models.KeyToID(id), ith.Successor.ToString())
				return ith.Successor, nil
			}
		}
	}
	glog.Infof("closest preceding node to id=%s is %s", models.KeyToID(id), ln.ToString())
	return ln.ToNode(), nil
}

// Successor - This is what this is all about, given an Key we will return
// the node that is responsible for that Key
func (ln *LocalNode) Successor(id models.Identifier) (models.Node, error) {
	// does the key fall within ln's ID and the first entry of the finger table
	// if the key is greater than ln.ID and less than ln.successor.ID, return
	// ln.successor
	nPrime, err := ln.ClosestPrecedingNode(id)
	if err != nil {
		return nPrime, errors.Wrap(err, "failed to get successor: ")
	}
	glog.Infof("successor called: based on finger table, goto: %s", nPrime.ToString())
	glog.Infof("finger table: %s", ln.fingerTable.ToString())
	// if we are the nPrime, return self
	if bytes.Compare(nPrime.ID[:], ln.ID[:]) == 0 {
		return ln.ToNode(), nil
	}

	// call whoever we think is closest
	rn, err := NewRemoteNode(nPrime.Addr, nPrime.PublicKey)
	if err != nil {
		return models.Node{}, errors.Wrap(err, "failure creating new remote node: ")
	}

	glog.Infof("contacting node: %s\n", nPrime.ToString())
	node, err := rn.Successor(id, ln.server.PrivateKey)
	if err != nil {
		return models.Node{}, errors.Wrap(err, "failure getting successor from remote node: ")
	}
	glog.Infof("recieved successor from remote rpc call: %s\n", node.ToString())

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

	lnID := models.KeyToID(ln.ID)
	nID := models.KeyToID(n.ID)
	pID := models.KeyToID(ln.predecessor.ID)

	if ln.predecessor.Addr == "" {
		ln.predecessor = n
		glog.Infof("predescessor set to: %s\n", ln.predecessor.ToString())
		return nil
	}

	if pID < lnID {
		// easy, no wrapping
		if pID < nID && nID < lnID {
			// yep, closer, change it
			ln.predecessor = n
			glog.Infof("predescessor set to: %s\n", ln.predecessor.ToString())
			return nil
		}
		return errors.New("not updating as new isn't between")
	} else {
		// not easy, wrapping around the horn
		if nID < lnID || pID < nID {
			// yep, closer, change it
			ln.predecessor = n
			glog.Infof("predescessor set to: %s\n", ln.predecessor.ToString())
			return nil
		}
		return errors.New("not updating as new isn't between")
	}
	return nil
}
