package file

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/golang/glog"
	"github.com/husobee/peerstore/models"
	"github.com/husobee/peerstore/protocol"
)

var fileMu = &sync.Mutex{}

// GetFileHandler - This is the server handler which manages Get File Requests
func GetFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value(models.DataPathContextKey).(string)

	glog.Infof("GetFileHandler Request: %v, %x", r.Header.ResourceName, r.Header.Key)

	var response = protocol.Response{
		Status: protocol.Success,
	}
	fileMu.Lock()
	defer fileMu.Unlock()
	// perform file get based on key
	buf, err := Get(dataPath, r.Header.Key)
	defer buf.Close()
	if err != nil {
		glog.Infof("ERR: %v\n", err)
		// write the get file error out.
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	// read the owner id out of the "header" of the file
	idSlice := make([]byte, 20)
	n, err := buf.Read(idSlice)
	glog.Infof("header is: %x", idSlice)
	if n != 20 {
		glog.Infof("ERR: could not read header from file\n")
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	if err != nil {
		glog.Infof("ERR: %s\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	id := models.Identifier{}
	copy(id[:], idSlice)

	// all we need to do here is compare the from in the request
	// header to what the file "header" has, as we have already
	// authenticated the request against that from id
	if bytes.Compare(id[:], r.Header.From[:]) != 0 {
		glog.Infof("invalid ownership of this resource requested\n")
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
			return protocol.Response{
				Status: protocol.Error,
			}
		}
	}
	glog.Infof("!!!!!!!!!!!!!!!!!!!!! GET FILE response: !!!!!!!!!!! %s", string(response.Data))
	return response
}

// PostFileHandler - This is the server handler which manages Post File Requests
func PostFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value(models.DataPathContextKey).(string)
	// add the request owner id to the file "header"
	fileMu.Lock()
	defer fileMu.Unlock()
	if err := Post(
		dataPath, r.Header.Key, bytes.NewBuffer(append(r.Header.From[:], r.Data...)),
	); err != nil {
		glog.Infof("ERR: %s", err.Error())
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	glog.Infof("!!!!!!!!!!!!!!!!!!!!! POST FILE request: !!!!!!!!!!! %s", string(r.Data))

	var timestamp = models.IncrementClock(r.Header.Clock)

	return protocol.Response{
		Header: protocol.Header{
			Clock: timestamp,
		},
		Status: protocol.Success,
	}
}

// DeleteFileHandler - This is the server handler which manages Delete File Requests
func DeleteFileHandler(ctx context.Context, r *protocol.Request) protocol.Response {
	var dataPath = ctx.Value(models.DataPathContextKey).(string)
	fileMu.Lock()
	defer fileMu.Unlock()

	// perform file get based on key
	buf, err := Get(dataPath, r.Header.Key)
	if err != nil {
		glog.Infof("ERR: %v\n", err)
		// write the get file error out.
		buf.Close()
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	// read the owner id out of the "header" of the file
	id := models.Identifier{}
	n, err := buf.Read(id[:])
	buf.Close()
	if n != 20 {
		glog.Infof("ERR: could not read header from file\n")
		return protocol.Response{
			Status: protocol.Error,
		}
	}
	if err != nil {
		glog.Infof("ERR: %s\n", err)
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	if bytes.Compare(id[:], r.Header.From[:]) != 0 {
		glog.Infof("invalid ownership of this resource requested\n")
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	if err := Delete(dataPath, r.Header.Key); err != nil {
		glog.Infof("failed to delete")
		return protocol.Response{
			Status: protocol.Error,
		}
	}

	var timestamp = models.IncrementClock(r.Header.Clock)

	return protocol.Response{
		Header: protocol.Header{
			Clock: timestamp,
		},
		Status: protocol.Success,
	}
}
