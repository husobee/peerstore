package protocol

import (
	"context"
)

// Handler - This is what a server handler signature should be
type Handler = func(ctx context.Context, r *Request) Response
