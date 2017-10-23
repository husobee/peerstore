package protocol

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/gob"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/crypto"
	"github.com/husobee/peerstore/models"
)

func init() {
	gob.Register(NodeRegistrationResponse{})
}

// Handler - This is what a server handler signature should be
type Handler = func(ctx context.Context, r *Request) Response

type NodeRegistrationResponse struct {
	Signature []byte
	SignedBy  models.Identifier
	Nodes     []models.Node
}

// NodeRegistrationHandler - this handler handles all node registrations.  A node
// registration consists of the node giving the server it's public key, and the
// server signing that key, and returning the signed key as well as a list of
// other nodes with coreseponding public keys that it knows about
func (s *Server) NodeRegistrationHandler(ctx context.Context, r *Request) Response {
	// validate invite
	// add requested node to trustedNodes list
	if _, err := s.getTrustedNode(r.Header.From); err == nil {
		// we already have this node, response should error
		return Response{
			Status: Error,
		}
	}
	node := models.Node{
		ID:        r.Header.From,
		Addr:      r.Header.FromAddr,
		PublicKey: r.Header.PubKey,
	}
	glog.Infof("adding this node to trustedNode: %s", node.ToString())
	// we do not have this node, so we should add it
	s.addTrustedNode(node)
	// sign the requested node's public key with our private key
	buf := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buf)
	encoder.Encode(r.Header.PubKey)

	signature, err := crypto.Sign(s.PrivateKey, buf.Bytes())
	if err != nil {
		glog.Infof("failed to sign signature: %s", err)
		return Response{
			Status: Error,
		}
	}

	nrr := NodeRegistrationResponse{
		Signature: signature,
		SignedBy:  s.id,
		Nodes:     s.getAllTrustedNodes(),
	}

	buf = bytes.NewBuffer([]byte{})
	encoder = gob.NewEncoder(buf)
	encoder.Encode(nrr)

	// send back the signature and the list of trusted Nodes
	return Response{
		Status: Success,
		Data:   buf.Bytes(),
	}
}

// NodeTrustHandler - this handler handles all node trust requests.  A node
// registration consists of the node giving the server it's public key, and the
// server signing that key, and returning the signed key as well as a list of
// other nodes with coreseponding public keys that it knows about
func (s *Server) NodeTrustHandler(ctx context.Context, r *Request) Response {
	// validate signature of node of trust
	// add requested node to trustedNodes list
	// sign the requested node's public key with our private key
	// send back the signature and the list of trusted Nodes

	// validate invite
	// add requested node to trustedNodes list
	signer, err := s.getTrustedNode(r.Header.SignedBy)
	if err == nil {
		// we already have this node, response should error
		glog.Infof("signer node is not trusted")
		return Response{
			Status: Error,
		}
	}

	buf := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buf)
	encoder.Encode(r.Header.PubKey)

	if err := crypto.Verify(signer.PublicKey, r.Header.Signature, buf.Bytes()); err != nil {
		glog.Infof("failed to verify signature of signer: %s", err)
		return Response{
			Status: Error,
		}
	}
	// we do not have this node, so we should add it
	s.addTrustedNode(models.Node{
		ID:        r.Header.From,
		Addr:      r.Header.FromAddr,
		PublicKey: r.Header.PubKey,
	})
	// sign the requested node's public key with our private key
	signature, err := crypto.Sign(s.PrivateKey, buf.Bytes())
	if err != nil {
		glog.Infof("failed to sign signature: %s", err)
		return Response{
			Status: Error,
		}
	}

	nrr := NodeRegistrationResponse{
		Signature: signature,
		Nodes:     s.getAllTrustedNodes(),
	}

	buf = bytes.NewBuffer([]byte{})
	encoder = gob.NewEncoder(buf)
	encoder.Encode(nrr)

	// send back the signature and the list of trusted Nodes
	return Response{
		Status: Success,
		Data:   buf.Bytes(),
	}
}

// UserRegistrationHandler - this handler handles all user registrations.  A user
// registration consists of the user giving the server it's public key, and the
// server will place that public key in the DHT for future validations
func (s *Server) UserRegistrationHandler(ctx context.Context, r *Request) Response {
	// take the request pubkey and figure out which node it belongs to,
	// and write the public key to a file using the file request to said
	// node for others to lookup as needed
	buf := bytes.NewBuffer([]byte{})
	err := crypto.WritePublicKeyAsPem(buf, r.Header.PubKey)
	if err != nil {
		glog.Infof("failed to write pub key as pem: %s", err)
		return Response{Status: Error}
	}

	// figure out where to connect to, by asking self
	t, err := NewTransport("tcp", s.addr, NodeType, s.id, s.PrivateKey.Public().(*rsa.PublicKey), s.PrivateKey)
	defer t.Close()
	if err != nil {
		glog.Infof("ERR: %v", err)
		return Response{Status: Error}
	}
	// serialize our get successor request
	var idBuf = new(bytes.Buffer)
	enc := gob.NewEncoder(idBuf)
	enc.Encode(models.SuccessorRequest{
		models.Identifier(r.Header.Key),
	})

	resp, err := t.RoundTrip(&Request{
		Header: Header{
			From: s.id,
			Key:  r.Header.From,
		},
		Method: GetSuccessorMethod,
		Data:   idBuf.Bytes(),
	})
	if err != nil {
		glog.Infof("Failed to round trip the successor request: %v", err)
		return Response{Status: Error}
	}
	// connect to that host for this file
	// pull node out of response, and connect to that host
	var node = models.Node{}
	dec := gob.NewDecoder(bytes.NewBuffer(resp.Data))
	err = dec.Decode(&node)
	if err != nil {
		glog.Infof("Failed to deserialize the node data: %v", err)
		return Response{Status: Error}
	}

	// OKAY, NOW connect to it, and store the file
	// figure out where to connect to, by asking self
	st, err := NewTransport("tcp", node.Addr, NodeType, s.id, node.PublicKey, s.PrivateKey)
	defer st.Close()
	if err != nil {
		glog.Infof("ERR: %v", err)
		return Response{Status: Error}
	}

	glog.Infof("server id is : %+v", s.id)
	response, err := st.RoundTrip(&Request{
		Header: Header{
			Key:        r.Header.From,
			From:       s.id,
			DataLength: uint64(len(buf.Bytes())),
		},
		Method: PostFileMethod,
		Data:   buf.Bytes(),
	})
	if err != nil {
		glog.Infof("ERR: %v\n", err)
		return Response{Status: Error}
	}
	glog.Infof("response from file post: %+v", response)

	return Response{Status: Success}
}
