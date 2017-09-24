package protocol

import (
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

var (
	// ValidResponseStatus - Used for verification that a response is right
	ValidResponseStatus = map[ResponseStatus]bool{
		Success: true, Error: true,
	}
)

// Response - the response structure for any given request
type Response struct {
	Header Header
	Status ResponseStatus
	Data   []byte
}

// Validate - implementation of Validatable, makes sure the response is
// a valid response
func (r *Response) Validate() error {
	if err := r.Header.Validate(); err != nil {
		return errors.Wrap(err, "failed to validate response header: ")
	}

	if !ValidResponseStatus[r.Status] {
		return errors.New("failed to validate response status")
	}
	return nil
}
