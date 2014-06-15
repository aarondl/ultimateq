package data

// StoredChannel stores attributes for channels.
type StoredChannel struct {
	Name string
	JSONStorer
}

// NewStoredChannel creates a new stored channel.
func NewStoredChannel(name string) *StoredChannel {
	return &StoredChannel{name, make(JSONStorer)}
}
