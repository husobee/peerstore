package crypto

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/pkg/errors"
)

const sessionKeySize = 32

// GenerateSessionKey - helper that will return a random session id,
// in both plaintext and encrypted form
func GenerateSessionKey(key *rsa.PublicKey) ([]byte, []byte, error) {
	b := make([]byte, sessionKeySize)
	_, err := rand.Read(b)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read from random: ")
	}
	// now crypt it with RSA
	ciphertext, err := EncryptRSA(key, b[:])
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to encrypt with RSA: ")
	}
	return b[:], ciphertext, nil
}
