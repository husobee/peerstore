package chord

import "github.com/husobee/peerstore/models"

const (
	// MaxFingerTableSize - the maximum number of entries in a finger table which
	// is the number of bits in the hash, 8 bits per byte, 20 bytes in hash
	MaxFingerTableSize int = models.M
)
