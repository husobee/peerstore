package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/pkg/errors"
)

// padPKCS7 - pad input in using the PKCS7 algorithm and return results
func padPKCS7(in []byte) []byte {
	padding := 16 - (len(in) % 16)
	for i := 0; i < padding; i++ {
		in = append(in, byte(padding))
	}
	return in
}

// unpadPKCS7 - unpad in using PKCS7 algorithm and return results
func unpadPKCS7(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	padding := in[len(in)-1]
	if int(padding) > len(in) || padding > aes.BlockSize {
		return nil
	} else if padding == 0 {
		return nil
	}

	for i := len(in) - 1; i > len(in)-int(padding)-1; i-- {
		if in[i] != padding {
			return nil
		}
	}
	return in[:len(in)-int(padding)]
}

// Encrypt - encrypt encrypts with aes256 in cbc mode, returns
// ciphertext, iv and error
func Encrypt(key, plaintext []byte) ([]byte, []byte, error) {
	// pad the text before trying to encrypt
	paddedPlaintext := padPKCS7(plaintext)
	// create a new block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create new cipher: ")
	}

	// create IV
	var iv = make([]byte, aes.BlockSize)

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate iv: ")
	}

	// our encrypter
	mode := cipher.NewCBCEncrypter(block, iv)
	// do the encryption
	// will do encryption in place if arguments are the same..
	mode.CryptBlocks(paddedPlaintext, paddedPlaintext)

	return paddedPlaintext, iv, nil
}

// Decrypt - decrypt with aes256 in cbc mode, returns
// plaintext and error
func Decrypt(key, ciphertext, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create new cipher: ")
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext is too short")
	}

	// CBC mode always works in whole blocks.
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	return unpadPKCS7(ciphertext), nil
}
