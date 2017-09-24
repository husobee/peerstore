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
	gob.Register(SetPredecessorRequest{})
}

type SuccessorRequest struct {
	ID models.Identifier
}

type SetPredecessorRequest struct {
	Node models.Node
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
	glog.Infof("successor found: addr=%s, id=%s\n",
		node.Addr,
		hex.EncodeToString(node.ID[:]))

	enc := gob.NewEncoder(out)
	if err := enc.Encode(node); err != nil {
		glog.Infof("encode successor response error: %v\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	// write the response to the bytes of the response data
	response.Data = out.Bytes()

	glog.Infof("response for successor handler: Node.Addr=%s, Node.ID=%s\n",
		node.Addr,
		hex.EncodeToString(node.ID[:]))

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
