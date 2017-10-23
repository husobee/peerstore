package protocol

import (
	"crypto/aes"
	"crypto/rsa"
	"encoding/gob"
	"net"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
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

type encoder interface {
	Encode(interface{}) error
}
type decoder interface {
	Decode(interface{}) error
}

// Transport - a transport structure that will implement RoundTripper
// transport will also handle all encryption/decryption of the messages
type Transport struct {
	Type    CallerType
	conn    net.Conn
	from    models.Identifier
	peerKey *rsa.PublicKey
	selfKey *rsa.PrivateKey
	enc     encoder
	dec     decoder
}

// Close - close the connection transport
func (t *Transport) Close() {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
}

// NewTransport - create a new transport structure
func NewTransport(proto, addr string, t CallerType, id models.Identifier, peerKey *rsa.PublicKey, selfKey *rsa.PrivateKey) (*Transport, error) {
	conn, err := net.Dial(proto, addr)
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)
	return &Transport{
		Type:    t,
		conn:    conn,
		enc:     enc,
		dec:     dec,
		selfKey: selfKey,
		peerKey: peerKey,
		from:    id,
	}, err
}

// RoundTrip - Implementation of a round tripper interface,
// effectively this is how the request will be serialized,
// and put on the wire, and how the response will be deserialized
func (t *Transport) RoundTrip(request *Request) (Response, error) {
	err := encryptAndEncode(t.enc, request, t.Type, t.peerKey, t.from, t.selfKey)
	if err != nil {
		glog.Infof("failed to encrypt and encode in roundtrip: %s", err)
		return Response{}, errors.Wrap(err, "failure encoding request: ")
	}
	_, response, _, err := decryptAndDecodeResponse(t.dec, t.selfKey)
	return *response, err
}

type CallerType uint8

const (
	UserType CallerType = iota
	NodeType
)

// Header - protocol header, used in every message, contains
// the peerstore version, the key (if applicable), from and to nodes
// and the length of the data in the data section of the message.
type Header struct {
	Key        models.Identifier
	From       models.Identifier
	FromAddr   string
	Type       CallerType
	PubKey     *rsa.PublicKey
	SignedBy   models.Identifier
	Signature  []byte
	DataLength uint64
}

// Validate - Implement validate for the header validation
func (h *Header) Validate() error {
	return nil
}

// EncryptedMessage - this will be the "wrapper" to add
// encryption to the messages.  The transport will pack
// the existing request/response into this encrypted message
type EncryptedMessage struct {
	Header     Header
	SessionKey []byte
	IV         []byte
	CipherText []byte
}

// Validate - Implement validate for the header validation
func (em *EncryptedMessage) Validate() error {
	if em.SessionKey == nil || len(em.SessionKey) == 0 {
		return errors.New("invalid session id in encrypted message")
	}
	if em.IV == nil || len(em.IV) == 0 {
		return errors.New("invalid iv in encrypted message")
	}
	if em.CipherText == nil || len(em.CipherText)%aes.BlockSize != 0 {
		return errors.New("invalid ciphertext in encrypted message")
	}
	return nil
}
