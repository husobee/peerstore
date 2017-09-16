package chord

import "github.com/husobee/peerstore/models"

// ChordNode - interface for a chord node, contains all the
// potential methods a node should be able to perform.
type ChordNode interface {
	GetPredecessor(models.Identifier) (ChordNode, error)
	GetSuccessor(models.Identifier) (ChordNode, error)
	GetFingerTable() (models.FingerTable, error)

	SetSuccessor(ChordNode) error
	SetPredecessor(ChordNode) error
	UpdateFingerTable(int, models.Node) error
}

const (
	MaxFingerTableSize int = 8 * 20
)
