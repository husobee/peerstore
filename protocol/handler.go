package protocol

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/husobee/peerstore/file"
)

var (
	MethodHandlerMap = map[RequestMethod]Handler{
		GetFileMethod:    GetFileHandler,
		PostFileMethod:   PostFileHandler,
		DeleteFileMethod: DeleteFileHandler,
	}
)

type Handler = func(ctx context.Context, r *Request) Response

func GetFileHandler(ctx context.Context, r *Request) Response {
	var dataPath = ctx.Value("dataPath").(string)
	var response = Response{
		Status: Success,
	}
	// perform file get based on key
	buf, err := file.Get(dataPath, r.Header.Key)
	if err != nil {
		log.Printf("ERR: %v\n", err)
		// write the get file error out.
		return Response{
			Status: Error,
		}
	}
	for n := 1; n > 0; {
		var err error
		tmp := make([]byte, 256)
		n, err = buf.Read(tmp)
		response.Data = append(response.Data, tmp[:n]...)
		if err != nil {
			if err == io.EOF {
				// file is fully read, continue
				continue
			}
			log.Printf("ERR: %v\n", err)
			buf.Close()
			return Response{
				Status: Error,
			}
		}
	}
	buf.Close()
	return response
}

func PostFileHandler(ctx context.Context, r *Request) Response {
	var dataPath = ctx.Value("dataPath").(string)
	if err := file.Post(
		dataPath, r.Header.Key, bytes.NewBuffer(r.Data),
	); err != nil {
		log.Printf("ERR: %s", err.Error())
		return Response{
			Status: Error,
		}
	}
	return Response{
		Status: Success,
	}
}

func DeleteFileHandler(ctx context.Context, r *Request) Response {
	var dataPath = ctx.Value("dataPath").(string)
	if err := file.Delete(dataPath, r.Header.Key); err != nil {
		return Response{
			Status: Error,
		}
	}
	return Response{
		Status: Success,
	}
}
