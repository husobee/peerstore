package protocol

import (
	"context"
)

type Handler = func(ctx context.Context, r *Request) Response
