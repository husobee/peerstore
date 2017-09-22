package chord

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
)

// SuccessorHandler - the handler to handle all server calls to get successor for this local node
func (ln *LocalNode) SuccessorHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	// get the request, pull out the ID from the request body
	var (
		response = protocol.Response{
			Status: protocol.Success,
		}
		body = bytes.NewBuffer(r.Data)
		id   models.Identifier
		out  = &bytes.Buffer{}
	)

	// create a gob decoder to decode the body
	dec := gob.NewDecoder(body)

	err := dec.Decode(&id)
	if err != nil {
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	// this point we have the ID, time to call successor on ln
	node, err := ln.Successor(id)

	enc := gob.NewEncoder(out)
	if err := enc.Encode(node); err != nil {
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	// write the response to the bytes of the response data
	response.Data = out.Bytes()

	return response
}

// PredecessorHandler - the handler to handle all server calls to get predecessor for this local node
func (ln *LocalNode) PredecessorHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	return protocol.Response{}
}

// FingerTableHandler - the handler to handle all server calls to get the finger table for the local node
func (ln *LocalNode) FingerTableHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	// get the request, pull out the ID from the request body
	var (
		response = protocol.Response{
			Status: protocol.Success,
		}
		out = &bytes.Buffer{}
	)

	enc := gob.NewEncoder(out)
	if err := enc.Encode(ln.fingerTable); err != nil {
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	// write the response to the bytes of the response data
	response.Data = out.Bytes()

	return response
}
