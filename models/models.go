package models

import (
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
)

// ContextKey - this is a type which is used as keys for the context
type ContextKey uint64

const (
	// DataPathContextKey - this is the context key for the data path
	DataPathContextKey ContextKey = iota
	// NumRequestWorkerContextKey - this is the context key for the data path
	NumRequestWorkerContextKey
)

func init() {
	gob.Register(Node{})
	gob.Register(Identifier{})
}

// Identifier - This is a common Chord Identifier, also used for
// file names
type Identifier [20]byte

// Node - This is a peer node representation
type Node struct {
	ID   Identifier
	Addr string
}

// Compare - Given a Node, compare the parameter nPrime with this
// node to see which is greater/less/equal
func (n Node) Compare(nPrime Node) int {
	nNum := big.NewInt(0)
	nNum.SetBytes(n.ID[:])

	nPrimeNum := big.NewInt(0)
	nPrimeNum.SetBytes(nPrime.ID[:])

	return nNum.Cmp(nPrimeNum)
}

// CompareID - Given a Node, compare it's ID to the id parameter
func (n Node) CompareID(id Identifier) int {
	nNum := big.NewInt(0)
	nNum.SetBytes(n.ID[:])

	idNum := big.NewInt(0)
	idNum.SetBytes(id[:])

	return nNum.Cmp(idNum)
}

// ToString - Implementation of String
func (n Node) ToString() string {
	return fmt.Sprintf("addr=%s, id=%s", n.Addr,
		hex.EncodeToString(n.ID[:]))
}

// M - This is the max number of nodes in a finger table
const M = 20 * 8

// Interval - This is the interval in which a successor in the
// finger table is responsible
type Interval struct {
	Low  int
	High int
}

// Finger - This is a finger entry
type Finger struct {
	Start     int
	Interval  Interval
	Successor Node
}

// FingerTable - This is the structure of a finger table
type FingerTable struct {
	table [M]Finger
	mu    *sync.RWMutex
}

// NewFingerTable - Provision a new finger table
func NewFingerTable() *FingerTable {
	return &FingerTable{
		table: [M]Finger{},
		mu:    new(sync.RWMutex),
	}
}

// GetIth - Get the i'th entry from the given finger table, and return that
// finger.
func (ft *FingerTable) GetIth(i int) (Finger, error) {
	if i < 1 || i > M {
		return Finger{},
			errors.New("i'th entry of finger table must be between 1 and 160")
	}
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.table[i], nil
}

// SetIth - Set the i'th entry in a given finger table
func (ft *FingerTable) SetIth(i int, interval Interval, successor Node) error {
	if i < 1 || i > M {
		return errors.New("i'th entry of finger table must be between 1 and 160")
	}
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.table[i] = Finger{
		Start:     i,
		Interval:  interval,
		Successor: successor,
	}
	return nil
}

// ToString - string representation of a finger table
func (ft *FingerTable) ToString() string {
	ft.mu.RLock()
	var fmtFingerTable = "["
	for _, v := range ft.table {
		fmtFingerTable = fmt.Sprintf("%s{%s}, ",
			fmtFingerTable, v.Successor.ToString())
	}
	fmtFingerTable += "]"
	ft.mu.RUnlock()
	return fmtFingerTable
}
