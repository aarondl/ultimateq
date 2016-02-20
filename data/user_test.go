package data

import (
	"encoding/json"
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
	if exp, got := u.Host.String(), "nick"; exp != got {
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
	if exp, got := u.Host.String(), "nick!user@host"; exp != got {
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
	u.Realname = "realname realname"
	str = fmt.Sprint(u)
	if exp, got := str, "nick realname realname"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	u = NewUser("nick!user@host")
	u.Realname = "realname realname"
	str = fmt.Sprint(u)
	if exp, got := str, "nick nick!user@host realname realname"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestUser_JSONify(t *testing.T) {
	t.Parallel()

	a := NewUser("fish!fish@fish")
	a.Realname = "Fish"
	var b User

	str, err := json.Marshal(a)
	if err != nil {
		t.Error(err)
	}

	if string(str) != `{"host":"fish!fish@fish","realname":"Fish"}` {
		t.Errorf("Wrong JSON: %s", str)
	}

	if err = json.Unmarshal(str, &b); err != nil {
		t.Error(err)
	}

	if *a != b {
		t.Error("A and B differ:", a, b)
	}
}
