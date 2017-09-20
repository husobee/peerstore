package file

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/husobee/peerstore/protocol"
)

func GetFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value("dataPath").(string)
	var response = protocol.Response{
		Status: protocol.Success,
	}
	// perform file get based on key
	buf, err := Get(dataPath, r.Header.Key)
	if err != nil {
		log.Printf("ERR: %v\n", err)
		// write the get file error out.
		return protocol.Response{
			Status: protocol.Error,
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
			return protocol.Response{
				Status: protocol.Error,
			}
		}
	}
	buf.Close()
	return response
}

func PostFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value("dataPath").(string)
	if err := Post(
		dataPath, r.Header.Key, bytes.NewBuffer(r.Data),
	); err != nil {
		log.Printf("ERR: %s", err.Error())
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	return protocol.Response{
		Status: protocol.Success,
	}
}

func DeleteFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value("dataPath").(string)
	if err := Delete(dataPath, r.Header.Key); err != nil {
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	return protocol.Response{
		Status: protocol.Success,
	}
}
