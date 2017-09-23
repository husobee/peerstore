package protocol

import (
	"encoding/gob"
	"net"

	"github.com/pkg/errors"
)

func init() {
	gob.Register(Header{})
}

// RoundTripper - interface which will perform the request, and
// return the Response
type RoundTripper interface {
	RoundTrip(*Request) (Response, error)
}

type Encoder interface {
	Encode(interface{}) error
}
type Decoder interface {
	Decode(interface{}) error
}

// Transport - a transport structure that will implement RoundTripper
type Transport struct {
	conn net.Conn
	enc  Encoder
	dec  Decoder
}

// Close - close the connection transport
func (t *Transport) Close() {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
}

// NewTransport - create a new transport structure
func NewTransport(proto, remote string) (*Transport, error) {
	conn, err := net.Dial(proto, remote)
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)
	return &Transport{
		conn: conn,
		enc:  enc,
		dec:  dec,
	}, err
}

// RoundTrip - Implementation of a round tripper interface,
// effectively this is how the request will be serialized,
// and put on the wire, and how the response will be deserialized
func (t *Transport) RoundTrip(request *Request) (Response, error) {
	var response Response
	// serialize request
	if err := t.enc.Encode(request); err != nil {
		return Response{}, errors.Wrap(err, "failure encoding request: ")
	}
	// unserialize response
	if err := t.dec.Decode(&response); err != nil {
		return Response{}, errors.Wrap(err, "failure decoding response: ")
	}
	// validate response
	if err := response.Validate(); err != nil {
		return Response{}, errors.Wrap(err, "failure validating response: ")
	}
	return response, nil
}

// Header - protocol header, used in every message, contains
// the peerstore version, the key (if applicable), from and to nodes
// and the length of the data in the data section of the message.
type Header struct {
	Key        [20]byte
	DataLength uint64
}

func (h *Header) Validate() error {
	// TODO: figure out validation of header
	return nil
}
