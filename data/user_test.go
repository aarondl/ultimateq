package data

import (
	"fmt"
	"testing"
)

func TestUser_Create(t *testing.T) {
	t.Parallel()

	u := NewUser("")
	if got := u; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}

	u = NewUser("nick")
	if u == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := u.Nick(), "nick"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.Host(), "nick"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	u = NewUser("nick!user@host")
	if u == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := u.Nick(), "nick"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.Username(), "user"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.Hostname(), "host"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u.Host(), "nick!user@host"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUser_Realname(t *testing.T) {
	t.Parallel()

	u := NewUser("nick!user@host")
	u.SetRealname("realname realname")
	if exp, got := u.Realname(), "realname realname"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUser_String(t *testing.T) {
	t.Parallel()

	u := NewUser("nick")
	str := fmt.Sprint(u)
	if exp, got := str, "nick"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	u = NewUser("nick!user@host")
	str = fmt.Sprint(u)
	if exp, got := str, "nick nick!user@host"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	u = NewUser("nick")
	u.SetRealname("realname realname")
	str = fmt.Sprint(u)
	if exp, got := str, "nick realname realname"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	u = NewUser("nick!user@host")
	u.SetRealname("realname realname")
	str = fmt.Sprint(u)
	if exp, got := str, "nick nick!user@host realname realname"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}
