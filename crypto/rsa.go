package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"

	"github.com/pkg/errors"
)

// Sign - Create a digital signature with the RSA keypair that
// can be validated.  Function will create the hash and then sign
func Sign(key *rsa.PrivateKey, message []byte) ([]byte, error) {
	// sha256 hash the message
	hashed := sha256.Sum256(message)
	// sign the hash
	signature, err := rsa.SignPKCS1v15(
		rand.Reader, key, crypto.SHA256, hashed[:],
	)
	// handle error
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign message: ")
	}
	return signature, nil
}

// Verify - Verify a digitial signature with an RSA Public Key
// will return error if not able to verify the signature
func Verify(key *rsa.PublicKey, signature, message []byte) error {
	// hash the message
	hashed := sha256.Sum256(message)
	// verify the signature
	err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return errors.Wrap(err, "failed to verify message: ")
	}
	return nil
}

// EncryptRSA - Encrypt using RSA Public Key
func EncryptRSA(key *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, key, plaintext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt plaintext: ")
	}
	return ciphertext, nil
}

// DecryptRSA - Decrypt using RSA Private Key
func DecryptRSA(key *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	session, err := rsa.DecryptPKCS1v15(rand.Reader, key, ciphertext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt ciphertext: ")
	}
	return session, nil
}
