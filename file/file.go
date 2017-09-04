package file

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

// Get - get a file based on the key, returns an io.Reader
// which will be used to read the file
func Get(path string, key [20]byte) (io.ReadCloser, error) {
	f, err := os.OpenFile(
		fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])),
		os.O_RDWR|os.O_CREATE, 0600,
	)
	if err != nil {
		log.Println(err)
		return f, errors.Wrap(err, "error opening file")
	}
	return f, err
}

// Post - create or update a file based on the key, returns
// boolean success as well as an error
func Post(path string, key [20]byte, data io.Reader) (bool, error) {
	f, err := os.OpenFile(
		fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])),
		os.O_RDWR|os.O_CREATE, 0600,
	)
	if err != nil {
		log.Println(err)
		return false, errors.Wrap(err, "error opening file")
	}
	if _, err := io.Copy(f, data); err != nil {
		return false, errors.Wrap(err, "error writing file")
	}

	if err := f.Close(); err != nil {
		log.Println(err)
		return false, errors.Wrap(err, "error closing file")
	}
	return true, errors.New("not implemented")
}

// Delete - delete a file based on the key, returns
// boolean success as well as an error
func Delete(path string, key [20]byte) (bool, error) {
	if err := os.Remove(
		fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])),
	); err != nil {
		return false, errors.Wrap(err, "failed to remove file: ")
	}
	return true, nil
}
