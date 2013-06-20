package data

import (
	. "testing"
)

func TestStore(t *T) {
	t.Parallel()
	s := CreateStore()
	if s == nil {
		t.Error()
	}
}

func TestStore_AddUser(t *T) {
	t.Parallel()
	t.SkipNow()
	s := CreateStore()
	u := s.AddUser("*!*@host", 100, "a", "b")
	if u == nil {
		t.Fatal("User wasn't added.")
	}

	if u.Level != 100 || !u.HasFlag('a') || !u.HasFlag('b') {
		t.Error("Initialization didn't take.")
	}
}

func TestStore_AuthUser(t *T) {
	t.Parallel()
	t.SkipNow()
	s := CreateStore()
	s.AddUser("*!*@host", 100, "a", "b")

	a := s.AuthUser("nick!user@host", "server", "")
	if a == nil {
		t.Fatal("User was not authed.")
	}

	if a.Level != 100 || !a.HasFlag('a') || !a.HasFlag('b') {
		t.Error("Proper levels not returned.")
	}
}

func TestStore_FindUser(t *T) {
	t.Parallel()
	t.SkipNow()
}
