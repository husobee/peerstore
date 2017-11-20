package file

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/golang/glog"
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
		glog.Info(err)
		return f, errors.Wrap(err, "error opening file")
	}
	return f, err
}

// Post - create or update a file based on the key, returns
// boolean success as well as an error
func Post(path string, key [20]byte, data io.Reader) error {
	glog.Info("opening destination file",
		fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])),
	)
	// rm existing file first...
	os.Remove(fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])))

	f, err := os.OpenFile(
		fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])),
		os.O_RDWR|os.O_CREATE, 0600,
	)
	if err != nil {
		glog.Info(err)
		return errors.Wrap(err, "error opening file")
	}
	glog.Info("Writing file to storage")
	if _, err := io.Copy(f, data); err != nil {
		return errors.Wrap(err, "error writing file")
	}

	glog.Info("Closing file to storage")
	if err := f.Close(); err != nil {
		glog.Info(err)
		return errors.Wrap(err, "error closing file")
	}
	return nil
}

// Delete - delete a file based on the key, returns
// boolean success as well as an error
func Delete(path string, key [20]byte) error {
	if err := os.Remove(
		fmt.Sprintf("%s/%s", path, hex.EncodeToString(key[:])),
	); err != nil {
		return errors.Wrap(err, "failed to remove file: ")
	}
	return nil
}
