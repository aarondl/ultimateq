package data

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aarondl/ultimateq/irc"
)

func TestStore(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	if err != nil {
		t.Fatal(err)
	}

	if s.authed == nil {
		t.Error("Auth list not instantiated.")
	}

	if s.timeouts == nil {
		t.Error("Timeouts not instantiated.")
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

	user, err := s.AuthUserPerma(network, host, uname+uname, password)
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

	user, err = s.AuthUserPerma(network, `nick!user@host.com`, uname, password)
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

	user, err = s.AuthUserPerma(network, host, uname, password+password)
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

	user, err = s.AuthUserPerma(network, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	if s.authed[network+host] == nil {
		t.Error("User is not authenticated.")
	}

	// Testing previously authenticated look up.
	user, err = s.AuthUserPerma(network, host, uname, password)
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

	user, err := s.AuthUserPerma(network, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	if len(s.cache) == 0 {
		t.Error("Auth is not using cache.")
	}
	if s.GetAuthedUser(network, host) == nil {
		t.Error("User is not authenticated.")
	}

	s.Logout(network, host)

	if s.authed[network+host] != nil {
		t.Error("User is still authenticated.")
	}

	user, err = s.AuthUserPerma(network, host, uname, password)
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if user == nil {
		t.Error("Rejected good authentication.")
	}

	s.LogoutByUsername(uname)

	if s.authed[network+host] != nil {
		t.Error("User is still authenticated.")
	}
}

func setupUpdateTest(t *testing.T) (s *Store) {
	var err error
	s, err = NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Fatal(err)
	}

	ua1, err := NewStoredUser(uname, password)
	if err != nil {
		t.Fatal("Could not create user:", err)
	}
	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	return s
}

func TestStore_UpdateSeen(t *testing.T) {
	t.Parallel()
	s := setupUpdateTest(t)

	_, err := s.AuthUserTmp(network, host, uname, password)
	if err != nil {
		t.Error("Could not auth user:", err)
	}

	if timeout, ok := s.timeouts[network+host]; !ok {
		t.Error("Expected there to be a timeout for the user.")
	} else if timeout.Before(time.Now().UTC()) {
		t.Error("Timeout must be set in the future.")
	}

	s.Update(network, StateUpdate{
		Seen: []string{host},
	})

	if _, ok := s.timeouts[network+host]; ok {
		t.Error("Expected there to be no timeout for the user.")
	}
}

func TestStore_UpdateUnseen(t *testing.T) {
	t.Parallel()
	s := setupUpdateTest(t)

	_, err := s.AuthUserPerma(network, host, uname, password)
	if err != nil {
		t.Error("Could not auth user:", err)
	}

	if _, ok := s.timeouts[network+host]; ok {
		t.Error("Expected there to be no timeout for the user.")
	}

	s.Update(network, StateUpdate{
		Unseen: []string{host},
	})

	if timeout, ok := s.timeouts[network+host]; !ok {
		t.Error("Expected there to be a timeout for the user.")
	} else if timeout.Before(time.Now().UTC()) {
		t.Error("Timeout must be set in the future.")
	}
}

func TestStore_UpdateNick(t *testing.T) {
	t.Parallel()
	s := setupUpdateTest(t)

	_, err := s.AuthUserPerma(network, host, uname, password)
	if err != nil {
		t.Error("Could not auth user:", err)
	}

	newNick := "helloworld"
	oldNick := irc.Nick(host)
	newHost := strings.Replace(host, oldNick, newNick, 1)

	if _, ok := s.authed[network+host]; !ok {
		t.Error("Not authenticated somehow?")
	}

	s.Update(network, StateUpdate{
		Nick: []string{host, newHost},
	})

	if _, ok := s.authed[network+host]; ok {
		t.Error("This authentication record should have been removed.")
	}

	if _, ok := s.authed[network+newHost]; !ok {
		t.Error("This authentication record should have been created.")
	}
}

func TestStore_UpdateQuit(t *testing.T) {
	t.Parallel()
	s := setupUpdateTest(t)

	_, err := s.AuthUserTmp(network, host, uname, password)
	if err != nil {
		t.Error("Could not auth user:", err)
	}

	if timeout, ok := s.timeouts[network+host]; !ok {
		t.Error("Expected there to be a timeout for the user.")
	} else if timeout.Before(time.Now().UTC()) {
		t.Error("Timeout must be set in the future.")
	}
	if _, ok := s.authed[network+host]; !ok {
		t.Error("This authentication record should exist.")
	}

	s.Update(network, StateUpdate{Quit: host})

	if _, ok := s.authed[network+host]; ok {
		t.Error("This authentication record should not exist.")
	}
	if _, ok := s.timeouts[network+host]; ok {
		t.Error("Expected there to be no timeout for the user.")
	}
}

func TestStore_Reap(t *testing.T) {
	t.Parallel()
	s := setupUpdateTest(t)

	_, err := s.AuthUserTmp(network, host, uname, password)
	if err != nil {
		t.Error("Could not auth user:", err)
	}
	_, err = s.AuthUserTmp("net2", host, uname, password)
	if err != nil {
		t.Error("Could not auth user:", err)
	}

	if len(s.authed) != 2 || len(s.timeouts) != 2 {
		t.Error("There should be 2 temporary authentications.")
	}

	s.timeouts[network+host] = time.Now().UTC().AddDate(0, 0, -1)

	s.reap()

	if _, ok := s.authed[network+host]; ok {
		t.Error("This authentication record should not exist.")
	}
	if _, ok := s.timeouts[network+host]; ok {
		t.Error("Expected there to be no timeout for the user.")
	}
	if timeout, ok := s.timeouts["net2"+host]; !ok {
		t.Error("Expected there to be a timeout for the user.")
	} else if timeout.Before(time.Now().UTC()) {
		t.Error("Timeout must be set in the future.")
	}
	if _, ok := s.authed["net2"+host]; !ok {
		t.Error("This authentication record should exist.")
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

func TestStore_HasAny(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	var has bool

	has, err = s.HasAny()
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if has {
		t.Error("We initially should have none")
	}

	su := StoredUser{Username: "hi"}
	s.SaveUser(&su)

	has, err = s.HasAny()
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if !has {
		t.Error("It should be true after someone's been added")
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

	ua1 := createStoredUser()
	ua1.Username = uname
	ua1.Grant("", "", 5)
	ua2 := createStoredUser()
	ua2.Username = uname + uname
	ua2.Grant(network, "", 5)
	ua2.Grant(network, channel, 5)

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

func TestStore_NetworkUsers(t *testing.T) {
	t.Parallel()
	s, err := NewStore(MemStoreProvider)
	defer s.Close()
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	list, err := s.NetworkUsers(network)
	if list != nil || err != nil {
		t.Error("When db is empty both return params should be nil.")
	}

	ua1 := createStoredUser()
	ua1.Username = uname
	ua1.Grant(network, "", 5)
	ua2 := createStoredUser()
	ua2.Username = uname + uname
	ua2.Grant(network, channel, 5)

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.SaveUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	list, err = s.NetworkUsers(network)
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

	list, err := s.ChanUsers(network, channel)
	if list != nil || err != nil {
		t.Error("When db is empty both return params should be nil.")
	}

	ua1 := createStoredUser()
	ua1.Username = uname
	ua1.Grant(network, channel, 5)
	ua2 := createStoredUser()
	ua2.Username = uname + uname
	ua2.Grant(network, "", 5)

	err = s.SaveUser(ua1)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}
	err = s.SaveUser(ua2)
	if err != nil {
		t.Fatal("Error adding user:", err)
	}

	list, err = s.ChanUsers(network, channel)
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
		fmt.Sprintf(errFmtBadHost, "h", "u"),
		AuthErrHostNotFound,
	}
	if err1.Error() != "Host [h] does not match stored hosts for user [u]." {
		t.Error("The error message builder is not working correctly.")
	}

	var err2 error = AuthError{
		"msg",
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
