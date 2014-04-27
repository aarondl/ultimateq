package data

import (
	"testing"
)

func TestStore(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	if err != nil {
		t.Fatal(err)
	}

	if s.cache == nil {
		t.Error("Cache not instantiated.")
	}

	err = s.Close()
	if err != nil {
		t.Error("Closing database failed.")
	}
}

func TestStore_AddUser(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(s.cache) > 0 {
		t.Error("Pre-warmed cache somehow exists.")
	}

	ua1 := &UserAccess{Username: uname}
	ua2 := &UserAccess{Username: uname + uname}

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] == nil {
		t.Error("User was not cached.")
	}

	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] != nil {
		t.Error("User should no longer be cached due to caching limits.")
	}
	if s.cache[ua2.Username] == nil {
		t.Error("User was not cached.")
	}

	found, err := s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("The user was not found.")
	}
	found, err = s.fetchUser(ua2.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("The user was not found.")
	}
}

func TestStore_RemoveUser(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1 := &UserAccess{Username: uname}

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] == nil {
		t.Error("User was not cached.")
	}

	found, err := s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("Error fetching user.")
	}

	var removed bool
	removed, err = s.RemoveUser(ua1.Username)
	if err != nil {
		t.Fatal("Error removing user:", err)
	}
	if !removed {
		t.Error("User was not reported as removed.")
	}
	if s.cache[ua1.Username] != nil {
		t.Error("User is still cached.")
	}

	found, err = s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found != nil {
		t.Error("User should be removed.")
	}
}

func TestStore_AuthUser(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1, err := NewUserAccess(uname, password, `*!*@host`)
	if err != nil {
		t.Fatal("Error creating user:", err)
	}
	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	user, err := s.AuthUser(server, host, uname+uname, password)
	if user != nil || err == nil {
		t.Error("Failed to reject bad authentication.")
	}
	if autherr, ok := err.(AuthError); ok {
		if autherr.FailureType != AuthErrUserNotFound {
			t.Error("Wrong failure type:", autherr.FailureType)
		}
	} else {
		t.Error("Error was not an AuthError:", err)
	}

	user, err = s.AuthUser(server, `nick!user@host.com`, uname, password)
	if user != nil || err == nil {
		t.Error("Failed to reject bad authentication.")
	}
	if autherr, ok := err.(AuthError); ok {
		if autherr.FailureType != AuthErrHostNotFound {
			t.Error("Wrong failure type:", autherr.FailureType)
		}
	} else {
		t.Error("Error was not an AuthError:", err)
	}

	user, err = s.AuthUser(server, host, uname, password+password)
	if user != nil {
		t.Error("Failed to reject bad authentication.")
	}
	if autherr, ok := err.(AuthError); ok {
		if autherr.FailureType != AuthErrBadPassword {
			t.Error("Wrong failure type:", autherr.FailureType)
		}
	} else {
		t.Error("Error was not an AuthError:", err)
	}

	user, err = s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	if s.authed[server+host] == nil {
		t.Error("User is not authenticated.")
	}

	// Testing previously authenticated look up.
	user, err = s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}
}

func TestStore_AuthLogout(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1, err := NewUserAccess(uname, password)
	if err != nil {
		t.Fatal("Error creating user:", err)
	}
	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	s.cache = make(map[string]*UserAccess)

	user, err := s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	if len(s.cache) == 0 {
		t.Error("Auth is not using cache.")
	}
	if s.GetAuthedUser(server, host) == nil {
		t.Error("User is not authenticated.")
	}

	s.Logout(server, host)

	if s.authed[server+host] != nil {
		t.Error("User is still authenticated.")
	}

	user, err = s.AuthUser(server, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	s.LogoutByUsername(uname)

	if s.authed[server+host] != nil {
		t.Error("User is still authenticated.")
	}
}

func TestStore_Finding(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(s.cache) > 0 {
		t.Error("Pre-warmed cache somehow exists.")
	}

	ua1 := &UserAccess{Username: uname}
	ua2 := &UserAccess{Username: uname + uname}

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Could not add user.")
	}
	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Could not add user.")
	}

	s.cache = make(map[string]*UserAccess)

	found, err := s.fetchUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}
	found, err = s.fetchUser(ua2.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}

	if len(s.cache) > 0 {
		t.Error("Cache should not be warmed by fetchUser.")
	}

	found, err = s.FindUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}
	// Cached lookup, for test coverage.
	found, err = s.FindUser(ua1.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}
	found, err = s.FindUser(ua2.Username)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if found == nil {
		t.Error("User should have been found.")
	}

	if len(s.cache) != nMaxCache {
		t.Error("Cache should be being used.")
	}
}

func TestStore_IsFirst(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	var isFirst bool

	isFirst, err = s.IsFirst()
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if isFirst != true {
		t.Error("The first call to isFirst should return true.")
	}

	isFirst, err = s.IsFirst()
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if isFirst != false {
		t.Error("The subsequent calls to IsFirst should return false.")
	}
}

func TestStore_GlobalUsers(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	list, err := s.GlobalUsers()
	if list != nil || err != nil {
		t.Error("When db is empty both return params should be nil.")
	}

	ua1 := &UserAccess{Username: uname}
	ua1.GrantGlobalLevel(5)
	ua2 := &UserAccess{Username: uname + uname}
	ua2.GrantServerLevel(server, 5)
	ua2.GrantChannelLevel(server, channel, 5)

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	list, err = s.GlobalUsers()
	if len(list) != 1 {
		t.Error("There should be exactly 1 global user now.")
	}
	if list[0].Username != ua1.Username {
		t.Error("The wrong user was found!")
	}
}

func TestStore_ServerUsers(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	list, err := s.ServerUsers(server)
	if list != nil || err != nil {
		t.Error("When db is empty both return params should be nil.")
	}

	ua1 := &UserAccess{Username: uname}
	ua1.GrantServerLevel(server, 5)
	ua2 := &UserAccess{Username: uname + uname}
	ua2.GrantGlobalLevel(5)
	ua2.GrantChannelLevel(server, channel, 5)

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	list, err = s.ServerUsers(server)
	if len(list) != 1 {
		t.Error("There should be exactly 1 global user now.")
	}
	if list[0].Username != ua1.Username {
		t.Error("The wrong user was found!")
	}
}

func TestStore_ChanUsers(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	list, err := s.ChanUsers(server, channel)
	if list != nil || err != nil {
		t.Error("When db is empty both return params should be nil.")
	}

	ua1 := &UserAccess{Username: uname}
	ua1.GrantChannelLevel(server, channel, 5)
	ua2 := &UserAccess{Username: uname + uname}
	ua2.GrantGlobalLevel(5)
	ua2.GrantServerLevel(server, 5)

	err = s.AddUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.AddUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	list, err = s.ChanUsers(server, channel)
	if len(list) != 1 {
		t.Fatal("There should be exactly 1 global user now.")
	}
	if list[0].Username != ua1.Username {
		t.Error("The wrong user was found!")
	}
}

func TestStore_AuthError(t *testing.T) {
	var err1 error = AuthError{
		errFmtBadHost,
		[]interface{}{"h", "u"},
		AuthErrHostNotFound,
	}
	if err1.Error() != "Host [h] does not match stored hosts for user [u]." {
		t.Error("The error message builder is not working correctly.")
	}

	var err2 error = AuthError{
		"msg",
		nil,
		AuthErrHostNotFound,
	}
	if err2.Error() != "msg" {
		t.Error("The error message builder is not working correctly.")
	}
}
