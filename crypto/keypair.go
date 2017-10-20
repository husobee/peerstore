package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

func init() {
	gob.Register(rsa.PublicKey{})
}

const RSAKeySize int = 2048

// GenerateKeyPair - generate an RSA keypair
func GenerateKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, RSAKeySize)
}

// WriteKeypairAsPem - convert a keypair to PEM formatting for storage.  This
// will be used for storing the keypair to disk.
func WriteKeypairAsPem(w io.Writer, key *rsa.PrivateKey) error {
	// encode the private key block first
	if err := pem.Encode(w, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return errors.Wrap(err, "failed to encode private key of keypair: ")
	}
	return nil
}

// GobEncodePublicKey - encode the public key to gob formatting.
func GobEncodePublicKey(pub *rsa.PublicKey) ([]byte, error) {
	var buf = bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buf)
	if err := encoder.Encode(pub); err != nil {
		return nil, errors.Wrap(err, "failed to encode public key: ")
	}
	return buf.Bytes(), nil
}

// GobDecodePublicKey - decode the public key from gob formatting.
func GobDecodePublicKey(b []byte) (*rsa.PublicKey, error) {
	var pub = new(rsa.PublicKey)
	decoder := gob.NewDecoder(bytes.NewBuffer(b))
	if err := decoder.Decode(pub); err != nil {
		return nil, errors.Wrap(err, "failed to decode public key: ")
	}
	return pub, nil
}

func ReadKeypairAsPem(r io.Reader) (*rsa.PrivateKey, error) {
	var (
		key   *rsa.PrivateKey
		err   error
		rest  []byte
		block *pem.Block
	)

	rest, err = ioutil.ReadAll(r)

	var (
		//pubFound  = false
		privFound = false
	)

	// forever
	for {
		// decode the next block
		if len(rest) == 0 {
			return nil, errors.New(
				"pem encoded key file did not include a pub and private key")
		}
		block, rest = pem.Decode(rest)
		if block == nil {
			return nil, errors.New("invalid pem encoded key file")
		}
		// if this block is a private key block...
		if block.Type == "PRIVATE KEY" {
			privFound = true
			if key, err = x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
				return nil, errors.New("unable to parse private key from block")
			}
		}
		if privFound {
			break
		}
	}
	return key, nil
}

// Sign
