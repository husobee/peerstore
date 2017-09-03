package protocol

import (
	"bytes"
	"encoding/gob"

	"github.com/pkg/errors"
)

func init() {
	gob.Register(Request{})
}

// RequestMethod - this is the indication of what request is to
// be performed.
type RequestMethod uint64

const (
	// GetFileMethod - Get File Method to be used when getting files
	GetFileMethod RequestMethod = 1 << iota
	// PostFileMethod - Post File Method to be used when inserting or updating
	PostFileMethod
	// DeleteFileMethod - Delete File Method to be used when deleting files
	DeleteFileMethod
)

// Request - the standard request, includes a header,
// method and data.  The resource is defined in the header
// and the data length is defined in the header as well.
type Request struct {
	Header Header
	Method RequestMethod
	Data   []byte
}

// Serialize - convert the request to the wire format
func (r *Request) Serialize() ([]byte, error) {
	b := bytes.Buffer{}
	encoder := gob.NewEncoder(&b)
	err := encoder.Encode(r)
	return b.Bytes(), errors.Wrap(err, "request serialization failure: ")
}

// Deserialize - convert the request from the wire format
func DeserializeRequest(requestBytes []byte) (Request, error) {
	b := bytes.Buffer{}
	b.Write(requestBytes)
	decoder := gob.NewDecoder(&b)
	req := &Request{}
	err := decoder.Decode(req)
	return *req, errors.Wrap(err, "request deserialization failure: ")
}
