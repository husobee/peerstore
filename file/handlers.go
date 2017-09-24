package file

import (
	"bytes"
	"context"
	"io"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
)

// GetFileHandler - This is the server handler which manages Get File Requests
func GetFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value(models.DataPathContextKey).(string)
	var response = protocol.Response{
		Status: protocol.Success,
	}
	// perform file get based on key
	buf, err := Get(dataPath, r.Header.Key)
	if err != nil {
		glog.Infof("ERR: %v\n", err)
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
			glog.Infof("ERR: %v\n", err)
			buf.Close()
			return protocol.Response{
				Status: protocol.Error,
			}
		}
	}
	buf.Close()
	return response
}

// PostFileHandler - This is the server handler which manages Post File Requests
func PostFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value(models.DataPathContextKey).(string)
	if err := Post(
		dataPath, r.Header.Key, bytes.NewBuffer(r.Data),
	); err != nil {
		glog.Infof("ERR: %s", err.Error())
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	return protocol.Response{
		Status: protocol.Success,
	}
}

// DeleteFileHandler - This is the server handler which manages Delete File Requests
func DeleteFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value(models.DataPathContextKey).(string)
	if err := Delete(dataPath, r.Header.Key); err != nil {
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	return protocol.Response{
		Status: protocol.Success,
	}
}
