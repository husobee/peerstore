package protocol

import (
	"context"
)

// Handler - This is what a server handler signature should be
type Handler = func(ctx context.Context, r *Request) Response

// NodeRegistrationHandler - this handler handles all node registrations.  A node
// registration consists of the node giving the server it's public key, and the
// server signing that key, and returning the signed key as well as a list of
// other nodes with coreseponding public keys that it knows about
func (s *Server) NodeRegistrationHandler(ctx context.Context, r *Request) Response {
	// validate invite
	// add requested node to trustedNodes list
	// sign the requested node's public key with our private key
	// send back the signature and the list of trusted Nodes
	return Response{}
}

// NodeTruestHandler - this handler handles all node trust requests.  A node
// registration consists of the node giving the server it's public key, and the
// server signing that key, and returning the signed key as well as a list of
// other nodes with coreseponding public keys that it knows about
func (s *Server) NodeTrustHandler(ctx context.Context, r *Request) Response {
	// validate signature of node of trust
	// add requested node to trustedNodes list
	// sign the requested node's public key with our private key
	// send back the signature and the list of trusted Nodes
	return Response{}
}
