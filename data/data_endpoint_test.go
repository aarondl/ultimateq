package data

import (
	"bytes"
	"sync"
	"testing"

	"github.com/aarondl/ultimateq/irc"
)

func TestDataEndpoint(t *testing.T) {
	var stateMutex, storeMutex sync.RWMutex
	state, err := NewState(irc.NewNetworkInfo())
	if err != nil {
		t.Error("Could not create state:", err)
	}
	store, err := NewStore(MemStoreProvider)
	if err != nil {
		t.Error("Could not create store:", err)
	}
	ep := NewDataEndpoint("key", &bytes.Buffer{}, state, store,
		&stateMutex, &storeMutex)
	if ep == nil {
		t.Fatal("EP was not created.")
	}

	if ep.GetKey() != "key" {
		t.Error("Key was unexpected:", ep.GetKey())
	}

	var called, reallyCalled bool
	called = ep.UsingState(func(st *State) {
		reallyCalled = true
	})
	if !called || !reallyCalled {
		t.Error("The state callback was not called:", called, reallyCalled)
	}
	called = ep.UsingStore(func(st *Store) {
		reallyCalled = true
	})
	if !called || !reallyCalled {
		t.Error("The store callback was not called:", called, reallyCalled)
	}

	ostate := ep.OpenState()
	if ostate != state {
		t.Error("Wrong object came back:", ostate)
	}
	ep.CloseState()

	ostore := ep.OpenStore()
	if ostore != store {
		t.Error("Wrong object came back:", ostore)
	}
	ep.CloseStore()
}
