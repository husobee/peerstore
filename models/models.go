package models

import (
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
)

func init() {
	gob.Register(Node{})
	gob.Register(Identifier{})
}

type Identifier [20]byte

type Node struct {
	ID   Identifier
	Addr string
}

func (n Node) Compare(nPrime Node) int {
	nNum := big.NewInt(0)
	nNum.SetBytes(n.ID[:])

	nPrimeNum := big.NewInt(0)
	nPrimeNum.SetBytes(nPrime.ID[:])

	return nNum.Cmp(nPrimeNum)
}

func (n Node) CompareID(id Identifier) int {
	nNum := big.NewInt(0)
	nNum.SetBytes(n.ID[:])

	idNum := big.NewInt(0)
	idNum.SetBytes(id[:])

	return nNum.Cmp(idNum)
}

func (n Node) ToString() string {
	return fmt.Sprintf("addr=%s, id=%s", n.Addr,
		hex.EncodeToString(n.ID[:]))
}

const M = 20 * 8

type Interval struct {
	Low  int
	High int
}

type Finger struct {
	Start     int
	Interval  Interval
	Successor Node
}

type FingerTable struct {
	table [M]Finger
	mu    *sync.RWMutex
}

func NewFingerTable() *FingerTable {
	return &FingerTable{
		table: [M]Finger{},
		mu:    new(sync.RWMutex),
	}
}

func (ft *FingerTable) GetIth(i int) (Finger, error) {
	if i < 1 || i > M {
		return Finger{},
			errors.New("i'th entry of finger table must be between 1 and 160")
	}
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.table[i], nil
}

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
