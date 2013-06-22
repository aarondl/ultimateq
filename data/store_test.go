package data

import (
	"github.com/cznic/kv"
	. "testing"
)

func TestStore(t *T) {
	t.Parallel()
	s, err := CreateStore(func() (*kv.DB, error) {
		return kv.CreateMem(&kv.Options{})
	})
	if err != nil {
		t.Fatal(err)
	}

	if s.cache == nil {
		t.Error("Cache not instantiated.")
	}
}

/*
func TestStore_AddUser(t *T) {
	t.Parallel()
	s, err := CreateStore(func() (*kv.DB, error) {
		return kv.CreateMem(&kv.Options{})
	})
	if err != nil {
		t.Fatal(err)
	}

	ua := createUserAccess(irc.WildMask(`*!*@*`))

	s.AddUser(ua)
	if s.Users[0] != ua {
		t.Error("The user was not added.")
	}
}

/*
func TestStore_RemoveUser(t *T) {
	t.Parallel()
	s := CreateStore()
	ua := CreateUserAccess(irc.WildMask(`*!*@host`))

	s.AddUser(ua)
	if s.Users[0] != ua {
		t.Error("The user was not added.")
	}

	removed := s.RemoveUser(`nick!user@host`)
	if removed == nil || len(s.Users) > 0 {
		t.Error("The user was not removed.")
	}
}

func TestStore_AuthUser(t *T) {
	t.Parallel()
	s := CreateStore()
	ua := CreateUserAccess(irc.WildMask(`*!*@host`))

	host := irc.Mask(`nick!user@host`)

	auth := s.AuthUser(host)
	if auth != nil {
		t.Error("The user was somehow authed against nothing.")
	}

	s.AddUser(ua)
	auth = s.AuthUser(host)
	if auth != ua {
		t.Error("The user was not authed.")
	}

	if _, ok := s.cache[host]; !ok {
		t.Error("The lookup was not cached.")
	}

	// Warmed cache
	auth = s.AuthUser(host)
	if auth != ua {
		t.Error("The user was not authed.")
	}
}

func TestStore_FindUser(t *T) {
	t.Parallel()
	s := CreateStore()
	ua := CreateUserAccess(irc.WildMask(`*!*@host`))

	s.AddUser(ua)
	if s.Users[0] != ua {
		t.Error("The user was not added.")
	}

	found := s.FindUser(`nick!user@host`)
	if found == nil {
		t.Error("The user was not found.")
	}

	found = s.FindUser(`nick!user@host.com`)
	if found != nil {
		t.Error("A bad user was found.")
	}
}
*/
