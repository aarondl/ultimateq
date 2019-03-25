package data

import (
	"testing"
)

var modes = new(int)

func TestChannelUser(t *testing.T) {
	t.Parallel()

	user := NewUser("nick")
	modes := NewUserModes(testKinds)

	cu := newChannelUser(user, &modes)

	if got, exp := cu.User, user; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := cu.UserModes, &modes; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestUserChannel(t *testing.T) {
	t.Parallel()

	ch := NewChannel("", testKinds)
	modes := NewUserModes(testKinds)

	uc := newUserChannel(ch, &modes)

	if got, exp := uc.Channel, ch; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := uc.UserModes, &modes; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}
