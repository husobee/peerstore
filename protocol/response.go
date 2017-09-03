package protocol

import (
	"bytes"
	"encoding/gob"

	"github.com/pkg/errors"
)

func init() {
	gob.Register(Response{})
}

// ResponseStatus - the type which indicates the response status for a request
type ResponseStatus int64

const (
	// Success - the message request was successful
	Success ResponseStatus = 1 << iota
	// Error - the message request was not successful
	Error
)

// Response - the response structure for any given request
type Response struct {
	Header Header
	Status ResponseStatus
	Data   []byte
}

// Serialize - convert the response to the wire format
func (r *Response) Serialize() ([]byte, error) {
	b := bytes.Buffer{}
	encoder := gob.NewEncoder(&b)
	err := encoder.Encode(r)
	return b.Bytes(), errors.Wrap(err, "response serialization failure: ")
}

// Deserialize - convert the response from the wire format
func DeserializeResponse(responseBytes []byte) (Response, error) {
	b := bytes.Buffer{}
	b.Write(responseBytes)
	decoder := gob.NewDecoder(&b)
	resp := &Response{}
	err := decoder.Decode(resp)
	return *resp, errors.Wrap(err, "response deserialization failure: ")
}
