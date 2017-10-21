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

// WritePrivateKeyAsPem - convert a keypair to PEM formatting for storage.  This
// will be used for storing the keypair to disk.
func WritePrivateKeyAsPem(w io.Writer, key *rsa.PrivateKey) error {
	// encode the private key block first
	if err := pem.Encode(w, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return errors.Wrap(err, "failed to encode private key of keypair: ")
	}
	return nil
}

// WritePublicKeyAsPem - convert a keypair to PEM formatting for storage.  This
// will be used for storing the keypair to disk.
func WritePublicKeyAsPem(w io.Writer, key *rsa.PublicKey) error {

	pubKey, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return errors.Wrap(err, "failed to marshal public key of keypair: ")
	}
	// encode the private key block first
	if err := pem.Encode(w, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKey,
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

func ReadPublicKeyAsPem(r io.Reader) (rsa.PublicKey, error) {
	var (
		key   *rsa.PublicKey
		rest  []byte
		block *pem.Block
		err   error
	)

	rest, err = ioutil.ReadAll(r)
	if err != nil {
		// unable to read file
		return rsa.PublicKey{}, errors.Wrap(err, "unable to read file: ")
	}

	var (
		pubFound = false
		ok       = false
	)

	// forever
	for {
		// decode the next block
		if len(rest) == 0 {
			return rsa.PublicKey{}, errors.New(
				"pem encoded key file did not include a pub and private key")
		}
		block, rest = pem.Decode(rest)
		if block == nil {
			return rsa.PublicKey{}, errors.New("invalid pem encoded key file")
		}
		// if this block is a private key block...
		if block.Type == "PUBLIC KEY" {
			pubFound = true
			v, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return rsa.PublicKey{}, errors.New("unable to parse Public key from block")
			}

			key, ok = v.(*rsa.PublicKey)
			if !ok {
				return rsa.PublicKey{}, errors.New("key block is not an rsa key")
			}

		}
		if pubFound {
			break
		}
	}
	return *key, nil
}
