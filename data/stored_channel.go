package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/aarondl/ultimateq/api"
)

// StoredChannel stores attributes for channels.
type StoredChannel struct {
	NetID      string `json:"netid"`
	Name       string `json:"name"`
	JSONStorer `json:"data"`
}

// NewStoredChannel creates a new stored channel.
func NewStoredChannel(netID, name string) *StoredChannel {
	return &StoredChannel{netID, name, make(JSONStorer)}
}

// Clone deep copies this StoredChannel.
func (s *StoredChannel) Clone() *StoredChannel {
	return &StoredChannel{s.NetID, s.Name, s.JSONStorer.Clone()}
}

// makeID is used to create a key to store this instance by.
func (s *StoredChannel) makeID() string {
	return strings.ToLower(fmt.Sprintf("%s.%s", s.Name, s.NetID))
}

// serialize turns the StoredChannel into bytes for storage.
func (s *StoredChannel) serialize() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// deserializeChannel reverses the Serialize process.
func deserializeChannel(serialized []byte) (*StoredChannel, error) {
	buffer := &bytes.Buffer{}
	decoder := gob.NewDecoder(buffer)
	if _, err := buffer.Write(serialized); err != nil {
		return nil, err
	}

	dec := &StoredChannel{}
	err := decoder.Decode(dec)
	return dec, err
}

func (s *StoredChannel) ToProto() *api.StoredChannel {
	var proto api.StoredChannel

	proto.Net = s.NetID
	proto.Name = s.Name

	if len(s.JSONStorer) != 0 {
		proto.Data = make(map[string]string, len(s.JSONStorer))
		for k, v := range s.JSONStorer {
			proto.Data[k] = v
		}
	}

	return &proto
}

func (s *StoredChannel) FromProto(proto *api.StoredChannel) {
	s.NetID = proto.Net
	s.Name = proto.Name

	if len(proto.Data) != 0 {
		s.JSONStorer = make(JSONStorer, len(proto.Data))
		for k, v := range proto.Data {
			s.JSONStorer[k] = v
		}
	}
}
