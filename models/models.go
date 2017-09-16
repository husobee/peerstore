package models

import "encoding/gob"

func init() {
	gob.Register(Node{})
}

type Identifier [20]byte

type Node struct {
	ID   Identifier
	Addr string
}

type FingerTable []Node
