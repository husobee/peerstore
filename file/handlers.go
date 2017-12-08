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

type idSecret struct {
	ID     models.Identifier
	Secret []byte
}

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

	// We need to read the first byte of the file to know
	// how many id/secret pairs are in the file
	ownerCount := make([]byte, 1)
	n, err := buf.Read(ownerCount)
	if n != 1 {
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

	idSecrets := []idSecret{}

	for i := byte(0); i < ownerCount[0]; i++ {
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

		secretSlice := make([]byte, 32)
		n, err = buf.Read(secretSlice)
		glog.Infof("secret is: %x", secretSlice)
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

		idSecrets = append(idSecrets, idSecret{
			ID: id, Secret: secretSlice})
	}

	// check each id in the list
	found := false
	for _, pair := range idSecrets {
		// all we need to do here is compare the from in the request
		// header to what the file "header" has, as we have already
		// authenticated the request against that from id
		if bytes.Compare(pair.ID[:], r.Header.From[:]) == 0 {
			found = true
			response.Header.Secret = pair.Secret
		}
	}

	// all we need to do here is compare the from in the request
	// header to what the file "header" has, as we have already
	// authenticated the request against that from id
	if !found {
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

	// TODO: we need to check if this is an existing file or not, if existing,
	// we need to pull the original ownership, validate user has permissions
	// then update the data, then also include the new "shareWith" header values
	// perform file get based on key
	buf, err := Get(dataPath, r.Header.Key)
	defer buf.Close()

	var timestamp = models.IncrementClock(r.Header.Clock)
	response := protocol.Response{
		Header: protocol.Header{
			Clock: timestamp,
		},
	}

	if err != nil {
		// this can mean it doesn't exist, so we should make it

		header := []byte{}
		header = append(header, byte(1+len(r.Header.SharedWith)))
		// user's id and secret
		header = append(header, r.Header.From[:]...)
		header = append(header, r.Header.Secret...)

		// shared with
		for _, shareWith := range r.Header.SharedWith {
			header = append(header, shareWith.ID[:]...)
			header = append(header, shareWith.Secret...)
		}

		if err := Post(
			dataPath, r.Header.Key, bytes.NewBuffer(append(header, r.Data...)),
		); err != nil {
			glog.Infof("ERR: %s", err.Error())
			return protocol.Response{
				Status: protocol.Error,
			}
		}

	} else {
		// We need to read the first byte of the file to know
		// how many id/secret pairs are in the file
		ownerCount := make([]byte, 1)
		n, err := buf.Read(ownerCount)
		if n != 1 {
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

		idSecrets := []idSecret{}

		for i := byte(0); i < ownerCount[0]; i++ {
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

			secretSlice := make([]byte, 32)
			n, err = buf.Read(secretSlice)
			glog.Infof("secret is: %x", secretSlice)
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

			idSecrets = append(idSecrets, idSecret{
				ID: id, Secret: secretSlice})
		}

		// check each id in the list
		found := false
		for _, pair := range idSecrets {
			// all we need to do here is compare the from in the request
			// header to what the file "header" has, as we have already
			// authenticated the request against that from id
			if bytes.Compare(pair.ID[:], r.Header.From[:]) == 0 {
				found = true
				response.Header.Secret = pair.Secret
			}
		}

		if !found {
			glog.Infof("Unauthorized Post Request: %v", r)
			return protocol.Response{
				Status: protocol.Error,
			}
		}
		// package up the number of shared owners, and keys

		header := []byte{}

		header = append(header, byte(len(idSecrets)+len(r.Header.SharedWith)))
		for _, pair := range idSecrets {
			header = append(header, pair.ID[:]...)
			header = append(header, pair.Secret...)
		}

		for _, shareWith := range r.Header.SharedWith {
			header = append(header, shareWith.ID[:]...)
			header = append(header, shareWith.Secret...)
		}
		// now we have all our old state, lets post the data changes
		if err := Post(
			dataPath, r.Header.Key, bytes.NewBuffer(append(header, r.Data...)),
		); err != nil {
			glog.Infof("ERR: %s", err.Error())
			return protocol.Response{
				Status: protocol.Error,
			}
		}
	}

	glog.Infof("!!!!!!!!!!!!!!!!!!!!! POST FILE request: !!!!!!!!!!! %s", string(r.Data))

	response.Status = protocol.Success
	return response
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

	ownerCount := make([]byte, 1)
	n, err := buf.Read(ownerCount)
	if n != 1 {
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

	idSecrets := []idSecret{}

	for i := byte(0); i < ownerCount[0]; i++ {
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

		secretSlice := make([]byte, 32)
		n, err = buf.Read(secretSlice)
		glog.Infof("secret is: %x", secretSlice)
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

		idSecrets = append(idSecrets, idSecret{
			ID: id, Secret: secretSlice})
	}

	var timestamp = models.IncrementClock(r.Header.Clock)
	response := protocol.Response{
		Header: protocol.Header{
			Clock: timestamp,
		},
		Status: protocol.Success,
	}

	// check each id in the list
	found := false
	for _, pair := range idSecrets {
		// all we need to do here is compare the from in the request
		// header to what the file "header" has, as we have already
		// authenticated the request against that from id
		if bytes.Compare(pair.ID[:], r.Header.From[:]) == 0 {
			found = true
			response.Header.Secret = pair.Secret
		}
	}

	// all we need to do here is compare the from in the request
	// header to what the file "header" has, as we have already
	// authenticated the request against that from id
	if !found {
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

	return response
}
