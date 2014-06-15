package data

import (
	"bytes"
	"encoding/gob"
)

// StoredChannel stores attributes for channels.
type StoredChannel struct {
	Name string
	JSONStorer
}

// NewStoredChannel creates a new stored channel.
func NewStoredChannel(name string) *StoredChannel {
	return &StoredChannel{name, make(JSONStorer)}
}

// serialize turns the StoredChannel into bytes for storage.
func (a *StoredChannel) serialize() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(a)
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
