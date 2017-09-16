package chord

import (
	"crypto/sha1"

	"github.com/husobee/peerstore/models"
	"github.com/pkg/errors"
)

// LocalNode - Implementation of ChordNode which holds the
// datastructure representing a local chord node.
type LocalNode struct {
	*models.Node
	fingerTable models.FingerTable
	successor   ChordNode
	predecessor ChordNode
}

func NewLocalNode(addr string) (ChordNode, error) {
	return &LocalNode{
		&models.Node{
			Addr: addr,
			ID: models.Identifier(
				sha1.Sum([]byte(addr)),
			),
		},
		models.FingerTable{},
		nil, nil,
	}, nil
}

func (ln *LocalNode) GetSuccessor(id models.Identifier) (ChordNode, error) {
	if ln.successor == nil {
		return nil, errors.New("successor is nil")
	}
	return ln.successor, nil
}

func (ln *LocalNode) GetPredecessor(id models.Identifier) (ChordNode, error) {
	if ln.predecessor == nil {
		return nil, errors.New("predecessor is nil")
	}
	return ln.predecessor, nil
}

func (ln *LocalNode) GetFingerTable() (models.FingerTable, error) {
	return ln.fingerTable, nil
}

func (ln *LocalNode) SetSuccessor(n ChordNode) error {
	ln.successor = n
	return nil
}

func (ln *LocalNode) SetPredecessor(n ChordNode) error {
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
