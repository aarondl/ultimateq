package data

import (
	"testing"
)

var modes = new(int)

func TestChannelUser(t *testing.T) {
	t.Parallel()

	user := NewUser("nick")
	modes := NewUserModes(testUserKinds)

	cu := NewChannelUser(
		user,
		modes,
	)

	if cu == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := cu.User, user; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := cu.UserModes, modes; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUserChannel(t *testing.T) {
	t.Parallel()

	ch := NewChannel("", testChannelKinds, testUserKinds)
	modes := NewUserModes(testUserKinds)

	uc := NewUserChannel(
		ch,
		modes,
	)

	if uc == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := uc.Channel, ch; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := uc.UserModes, modes; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}
