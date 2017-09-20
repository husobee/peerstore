package protocol

import (
	"context"

	"github.com/husobee/peerstore/file"
)

var (
	MethodHandlerMap = map[RequestMethod]Handler{
		GetFileMethod:    file.GetFileHandler,
		PostFileMethod:   file.PostFileHandler,
		DeleteFileMethod: file.DeleteFileHandler,
	}
)

type Handler = func(ctx context.Context, r *Request) Response
