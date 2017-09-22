package chord

import "github.com/husobee/peerstore/models"

// ChordNode - interface for a chord node, contains all the
// potential methods a node should be able to perform.
type ChordNode interface {
	Successor(models.Identifier) (models.Node, error)
	FingerTable() (models.FingerTable, error)

	SetPredecessor(models.Node) error
	UpdateFingerTable(int, models.Node) error
}

const (
	// MaxFingerTableSize - the maximum number of entries in a finger table which
	// is the number of bits in the hash, 8 bits per byte, 20 bytes in hash
	MaxFingerTableSize int = 8 * 20
)
