package data

import (
	"github.com/aarondl/ultimateq/irc"
	"io"
	"sync"
)

// DataEndpoint is an endpoint that can present both state and store data
// objects to a caller.
type DataEndpoint struct {
	key string
	*irc.Helper
	state        *State
	store        *Store
	protectState *sync.RWMutex
	protectStore *sync.RWMutex
}

// CreateDataEndpoint creates a data endpoint for use.
func CreateDataEndpoint(key string, write io.Writer, state *State, store *Store,
	stateMutex, storeMutex *sync.RWMutex) *DataEndpoint {

	return &DataEndpoint{
		key, &irc.Helper{write},
		state, store,
		stateMutex, storeMutex,
	}
}

// GetKey gets a key to identify the endpoint.
func (d *DataEndpoint) GetKey() string {
	return d.key
}

// UsingState calls a callback if this DataEndpoint can present a data state
// object. The returned boolean is whether or not the function was called.
func (d *DataEndpoint) UsingState(fn func(*State)) (called bool) {
	d.protectState.RLock()
	defer d.protectState.RUnlock()
	if d.state != nil {
		fn(d.state)
		called = true
	}
	return
}

// OpenState locks the data state, and returns it. CloseState must be called or
// the lock will never be released and the bot will sieze up. The state must
// be checked for nil.
func (d *DataEndpoint) OpenState() *State {
	d.protectState.RLock()
	return d.state
}

// CloseState unlocks the data state after use by OpenState.
func (d *DataEndpoint) CloseState() {
	d.protectState.RUnlock()
}

// UsingStore calls a callback if this DataEndpoint can present a data store
// object. The returned boolean is whether or not the function was called.
func (d *DataEndpoint) UsingStore(fn func(*Store)) (called bool) {
	d.protectStore.RLock()
	defer d.protectStore.RUnlock()
	if d.store != nil {
		fn(d.store)
		called = true
	}
	return
}

// OpenStore locks the data store, and returns it. CloseStore must be called or
// the lock will never be released and the bot will sieze up. The store must
// be checked for nil.
func (d *DataEndpoint) OpenStore() *Store {
	d.protectStore.RLock()
	return d.store
}

// CloseStore unlocks the data store after use by OpenState.
func (d *DataEndpoint) CloseStore() {
	d.protectStore.RUnlock()
}
