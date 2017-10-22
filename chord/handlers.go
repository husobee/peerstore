package chord

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
)

func init() {
	gob.Register(SuccessorRequest{})
}

// SuccessorRequest - this is the chord successor request strurture, the ID
// is the key we are looking to find a successor for.
type SuccessorRequest struct {
	ID models.Identifier
}

// SuccessorHandler - the handler to handle all server calls to get successor for this local node
func (ln *LocalNode) SuccessorHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	// get the request, pull out the ID from the request body
	var (
		response = protocol.Response{
			Status: protocol.Success,
		}
		body = bytes.NewBuffer(r.Data)
		in   = &SuccessorRequest{}
		out  = &bytes.Buffer{}
	)

	// create a gob decoder to decode the body
	dec := gob.NewDecoder(body)

	err := dec.Decode(in)
	if err != nil {
		glog.Infof("decode successor request error: %v\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	// this point we have the ID, time to call successor on ln
	node, err := ln.Successor(in.ID)
	glog.Infof("successor found: %s\n",
		node.ToString())

	enc := gob.NewEncoder(out)
	if err := enc.Encode(node); err != nil {
		glog.Infof("encode successor response error: %v\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	// write the response to the bytes of the response data
	response.Data = out.Bytes()

	glog.Infof("response for successor handler: %s\n",
		node.ToString())

	return response
}

// GetPredecessorHandler - the handler to handle all server calls to get predecessor for this local node
func (ln *LocalNode) GetPredecessorHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var (
		response = protocol.Response{
			Status: protocol.Success,
		}
		out = new(bytes.Buffer)
	)
	enc := gob.NewEncoder(out)
	predecessor, _ := ln.GetPredecessor()
	if err := enc.Encode(predecessor); err != nil {
		glog.Infof("encode successor response error: %v\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	// write the response to the bytes of the response data
	response.Data = out.Bytes()

	glog.Infof("response for get predecessor handler: Node.Addr=%s, Node.ID=%s\n",
		predecessor.Addr,
		hex.EncodeToString(predecessor.ID[:]))

	return response
}

// SetPredecessorHandler - the handler to handle all server calls to get predecessor for this local node
func (ln *LocalNode) SetPredecessorHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	// get the request, pull out the ID from the request body
	var (
		response = protocol.Response{
			Status: protocol.Success,
		}
		body = bytes.NewBuffer(r.Data)
		in   = &models.Node{}
	)

	// create a gob decoder to decode the body
	dec := gob.NewDecoder(body)

	err := dec.Decode(in)
	if err != nil {
		glog.Infof("decode successor request error: %v\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	glog.Infof("Set Predecessor Handler is getting set to: %s", in)

	// set the predecessor in ln
	err = ln.SetPredecessor(*in)
	if err != nil {
		glog.Infof("set predecessor failed: %v\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	newPredecessor, _ := ln.GetPredecessor()
	glog.Infof("!!! Set New Predecessor: Node.Addr=%s, Node.ID=%s\n",
		newPredecessor.Addr,
		hex.EncodeToString(newPredecessor.ID[:]))

	return response
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
