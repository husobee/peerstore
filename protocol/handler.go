package protocol

import (
	"context"
)

var (
	MethodHandlerMap = map[RequestMethod]Handler{}
)

type Handler = func(ctx context.Context, r *Request) Response
