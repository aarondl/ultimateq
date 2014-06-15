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

func TestStore_SaveUser(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(s.cache) > 0 {
		t.Error("Pre-warmed cache somehow exists.")
	}

	ua1 := &StoredUser{Username: uname}
	ua2 := &StoredUser{Username: uname + uname}

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	if s.cache[ua1.Username] == nil {
		t.Error("User was not cached.")
	}

	err = s.SaveUser(ua2)
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

	ua1 := &StoredUser{Username: uname}

	err = s.SaveUser(ua1)
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

	ua1, err := NewStoredUser(uname, password, `*!*@host`)
	if err != nil {
		t.Fatal("Error creating user:", err)
	}
	err = s.SaveUser(ua1)
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

	ua1, err := NewStoredUser(uname, password)
	if err != nil {
		t.Fatal("Error creating user:", err)
	}
	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	s.cache = make(map[string]*StoredUser)

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

	ua1 := &StoredUser{Username: uname}
	ua2 := &StoredUser{Username: uname + uname}

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Could not add user.")
	}
	err = s.SaveUser(ua2)
	if err != nil {
		t.Fatal("Could not add user.")
	}

	s.cache = make(map[string]*StoredUser)

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

	ua1 := &StoredUser{Username: uname}
	ua1.GrantGlobalLevel(5)
	ua2 := &StoredUser{Username: uname + uname}
	ua2.GrantServerLevel(server, 5)
	ua2.GrantChannelLevel(server, channel, 5)

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.SaveUser(ua2)
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

	ua1 := &StoredUser{Username: uname}
	ua1.GrantServerLevel(server, 5)
	ua2 := &StoredUser{Username: uname + uname}
	ua2.GrantGlobalLevel(5)
	ua2.GrantChannelLevel(server, channel, 5)

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.SaveUser(ua2)
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

	ua1 := &StoredUser{Username: uname}
	ua1.GrantChannelLevel(server, channel, 5)
	ua2 := &StoredUser{Username: uname + uname}
	ua2.GrantGlobalLevel(5)
	ua2.GrantServerLevel(server, 5)

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.SaveUser(ua2)
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
	t.Parallel()
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

func TestStore_SaveChannel(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	netID := "netID"
	ua1 := &StoredChannel{NetID: netID, Name: uname}
	ua2 := &StoredChannel{NetID: netID, Name: uname + uname}

	err = s.SaveChannel(ua1)
	if err != nil {
		t.Fatal("Error adding channel:", err)
	}

	err = s.SaveChannel(ua2)
	if err != nil {
		t.Fatal("Error adding channel:", err)
	}

	c1, err := s.FindChannel(netID, ua1.Name)

	if err != nil {
		t.Fatal("Cannot get channel", err)
	}

	c2, err := s.FindChannel(netID, ua2.Name)

	if err != nil {
		t.Fatal("Cannot get channel", err)
	}

	if ua1.Name != c1.Name {
		t.Error("Name mismatch", ua1.Name, c1.Name)
	}
	if ua1.NetID != c1.NetID {
		t.Error("NetID mismatch", ua1.NetID, c1.NetID)
	}

	if ua2.Name != c2.Name {
		t.Error("Name mismatch", ua2.Name, c2.Name)
	}
	if ua2.NetID != c2.NetID {
		t.Error("NetID mismatch", ua2.NetID, c2.NetID)
	}
}

func TestStore_RemoveChannel(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	netID := "netID"
	ua1 := &StoredChannel{NetID: netID, Name: uname}

	err = s.SaveChannel(ua1)
	if err != nil {
		t.Fatal("Error adding channel:", err)
	}

	c1, err := s.FindChannel(netID, ua1.Name)

	if err != nil {
		t.Fatal("Cannot get channel", err)
	}

	var removed bool
	removed, err = s.RemoveChannel(netID, ua1.Name)
	if err != nil {
		t.Fatal("Error removing channel:", err)
	}
	if !removed {
		t.Error("Channel was not reported as removed.")
	}

	c1, err = s.FindChannel(netID, ua1.Name)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if c1 != nil {
		t.Error("Channel should be removed.")
	}
}

func TestStore_Channels(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	list, err := s.Channels()
	if list != nil || err != nil {
		t.Error("When db is empty both return params should be nil.")
	}

	ua1 := &StoredChannel{Name: uname}
	ua2 := &StoredChannel{Name: uname + uname}

	err = s.SaveChannel(ua1)
	if err != nil {
		t.Fatal("Error adding channel:", err)
	}
	err = s.SaveChannel(ua2)
	if err != nil {
		t.Fatal("Error adding channel:", err)
	}

	list, err = s.Channels()
	if len(list) != 2 {
		t.Error("There should be exactly 2 global channels.")
	}

	found := false
	for _, ch := range list {
		if ch.Name == ua1.Name {
			found = true
			break
		}
	}

	if !found {
		t.Error("ua1 not found.")
	}

	found = false
	for _, ch := range list {
		if ch.Name == ua2.Name {
			found = true
			break
		}
	}

	if !found {
		t.Error("ua2 not found.")
	}
}
