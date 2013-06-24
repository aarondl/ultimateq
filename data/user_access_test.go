package data

import (
	"bytes"
	"github.com/aarondl/ultimateq/irc"
	. "testing"
)

func TestUserAccess(t *T) {
	t.Parallel()
	var a *UserAccess
	var err error
	var masks = []irc.WildMask{`*!*@host`, `*!user@*`}

	a = &UserAccess{}
	a, err = CreateUserAccess(uname, password, masks...)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if a.Username != uname {
		t.Errorf("Username is %v not %v", a.Username, uname)
	}
	if len(a.Password) == 0 {
		t.Error("Password not set properly:", err)
	}
	if len(a.Masks) != len(masks) {
		t.Errorf("Masks are %#v not %#v", a.Masks, masks)
	}

	a, err = CreateUserAccess("", password, masks...)
	if a != nil || err != errMissingUnameOrPwd {
		t.Error("Empty username should fail creation.")
	}
	a, err = CreateUserAccess(uname, "", masks...)
	if a != nil || err != errMissingUnameOrPwd {
		t.Error("Empty password should fail creation.")
	}
}

func TestUserAccess_VerifyPassword(t *T) {
	t.Parallel()
	a, err := CreateUserAccess(uname, password)
	if err != nil {
		t.Fatal("Unexpected Error:", err)
	}
	if !a.VerifyPassword(password) {
		t.Error("Real password was rejected.")
	}
	if a.VerifyPassword(password + password) {
		t.Error("Fake password was accepted.")
	}
}

func TestUserAccess_SerializeDeserialize(t *T) {
	var masks = []irc.WildMask{`*!*@host`, `*!user@*`}
	a, err := CreateUserAccess(uname, password, masks...)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	a.GrantGlobal(100, "a")
	a.GrantServer(server, 100, "a")
	a.GrantChannel(server, channel, 100, "a")

	serialized, err := a.Serialize()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(serialized) == 0 {
		t.Error("Serialization did not yield a serialized copy.")
	}

	b, err := deserialize(serialized)
	if err != nil {
		t.Fatal("Deserialization failed.")
	}
	if a.Username != b.Username || bytes.Compare(a.Password, b.Password) != 0 {
		t.Error("Username or Password did not deserialize.")
	}
	if len(a.Masks) != len(b.Masks) {
		t.Error("Masks were not serialized.")
	} else {
		for i := range a.Masks {
			if a.Masks[i] != b.Masks[i] {
				t.Errorf("Serialized mask not found:", a.Masks[i])
			}
		}
	}

	if !b.HasGlobalLevel(100) || !b.HasGlobalFlag('a') {
		t.Error("Lost global permissions in serialization.")
	}
	if !b.HasServerLevel(server, 100) || !b.HasServerFlag(server, 'a') {
		t.Error("Lost server permissions in serialization.")
	}
	if !b.HasChannelLevel(server, channel, 100) ||
		!b.HasChannelFlag(server, channel, 'a') {

		t.Error("Lost channel permissions in serialization.")
	}
}

func TestUserAccess_AddMasks(t *T) {
	t.Parallel()
	masks := []irc.WildMask{`*!*@host`, `*!user@*`}
	a := createUserAccess()
	if len(a.Masks) != 0 {
		t.Error("Masks should be empty.")
	}
	a.AddMasks(masks...)
	if len(a.Masks) != 2 || a.Masks[0] != masks[0] || a.Masks[1] != masks[1] {
		t.Error("Masks should have:", masks)
	}
}

func TestUserAccess_DelMasks(t *T) {
	t.Parallel()
	masks := []irc.WildMask{`*!*@host`, `*!user@*`, `nick!*@*`}
	a := createUserAccess(masks...)
	if len(a.Masks) != 3 {
		t.Error("Masks should have:", masks)
	}
	a.DelMasks(masks[1:]...)
	if len(a.Masks) != 1 {
		t.Error("Two masks should have been deleted.")
	}
	for _, mask := range masks[1:] {
		for _, hasMask := range a.Masks {
			if mask == hasMask {
				t.Errorf("Mask %v should have been deleted.", mask)
			}
		}
	}
}

func TestUserAccess_IsMatch(t *T) {
	t.Parallel()
	var wmasks = []irc.WildMask{"*!*@host1", "*!user2@*"}
	var mask1, mask2 irc.Mask = "nick1!user1@host1", "nick2!user2@host2"
	a := createUserAccess()
	if a.IsMatch(mask1) || a.IsMatch(mask2) {
		t.Error("No masks should match.")
	}

	a = createUserAccess(wmasks...)
	if !a.IsMatch(mask1) || !a.IsMatch(mask2) {
		t.Error(mask1, "and", mask2, "should match")
	}
}

func TestUserAccess_GrantGlobal(t *T) {
	t.Parallel()
	a := createUserAccess()
	a.GrantGlobalLevel(100)
	s := a.GetGlobal()
	if s.Level != 100 {
		t.Error("Level not set.")
	}

	a = createUserAccess()
	a.GrantGlobalFlags("aB")
	s = a.GetGlobal()
	if !s.HasFlag('a') || !s.HasFlag('a') {
		t.Error("Flags not set.")
	}

	a = createUserAccess()
	a.GrantGlobal(100, "aB")
	s = a.GetGlobal()
	if s.Level != 100 {
		t.Error("Level not set.")
	}
	if !s.HasFlag('a') || !s.HasFlag('a') {
		t.Error("Flags not set.")
	}
}

func TestUserAccess_RevokeGlobal(t *T) {
	t.Parallel()
	a := createUserAccess()
	a.GrantGlobal(100, "aB")
	a.RevokeGlobalLevel()
	if a.Global.Level != 0 {
		t.Error("Level not revoked.")
	}
	a.RevokeGlobalFlags("a")
	if a.Global.HasFlag('a') || !a.Global.HasFlag('B') {
		t.Error("Flags not revoked.")
	}
	a.RevokeGlobal()
	if a.Global != nil {
		t.Error("Global should be nil.")
	}
}

func TestUserAccess_HasGlobalLevel(t *T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasGlobalLevel(50) {
		t.Error("Should not have any access.")
	}
	a.GrantGlobalLevel(50)
	if !a.HasGlobalLevel(50) {
		t.Error("Should have access.")
	}
	if a.HasGlobalLevel(51) {
		t.Error("Should not have that high access.")
	}
}

func TestUserAccess_HasGlobalFlags(t *T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasGlobalFlags("ab") {
		t.Error("Should not have any flags.")
	}
	a.GrantGlobalFlags("ab")
	if !a.HasGlobalFlags("ab") {
		t.Error("Should have ab flags.")
	}
	if !a.HasGlobalFlag('a') {
		t.Error("Should have a flag.")
	}
	if a.HasGlobalFlag('c') {
		t.Error("Should not have c flag.")
	}
}

func TestUserAccess_GrantServer(t *T) {
	t.Parallel()
	a := createUserAccess()
	s := a.GetServer(server)
	if s != nil {
		t.Error("There should be no server access.")
	}

	a = createUserAccess()
	a.GrantServer(server, 100, "aB")
	s = a.GetServer(server)
	if s == nil {
		t.Error("There should be server access.")
	} else {
		if s.Level != 100 {
			t.Error("Level not set.")
		}
		if !s.HasFlag('a') || !s.HasFlag('B') {
			t.Error("Flags not added.")
		}
	}

	a = createUserAccess()
	a.GrantServerLevel(server, 100)
	s = a.GetServer(server)
	if s == nil {
		t.Error("There should be server access.")
	} else if s.Level != 100 {
		t.Error("Level not set.")
	}
	a = createUserAccess()
	a.GrantServerFlags(server, "aB")
	s = a.GetServer(server)
	if s == nil {
		t.Error("There should be server access.")
	} else if !s.HasFlag('a') || !s.HasFlag('B') {
		t.Error("Flags not added.")
	}
}

func TestUserAccess_RekoveServer(t *T) {
	t.Parallel()
	a := createUserAccess()
	a.GrantServer(server, 100, "abc")
	if a.GetServer(server) == nil {
		t.Error("Server permissions not granted.")
	}
	a.RevokeServer(server)
	if a.GetServer(server) != nil {
		t.Error("Server permissions not revoked.")
	}

	a.GrantServer(server, 100, "abc")
	a.RevokeServerLevel(server)
	if a.GetServer(server).Level > 0 {
		t.Error("Server level not revoked.")
	}

	a.RevokeServerFlags(server, "ab")
	if a.GetServer(server).HasFlags("ab") {
		t.Error("Server flags not revoked.")
	}
}

func TestUserAccess_HasServerLevel(t *T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasServerLevel(server, 50) {
		t.Error("Should not have any access.")
	}
	a.GrantServerLevel(server, 50)
	if !a.HasServerLevel(server, 50) {
		t.Error("Should have access.")
	}
	if a.HasServerLevel(server, 51) {
		t.Error("Should not have that high access.")
	}
}

func TestUserAccess_HasServerFlags(t *T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasServerFlags(server, "ab") {
		t.Error("Should not have any flags.")
	}
	a.GrantServerFlags(server, "ab")
	if !a.HasServerFlags(server, "ab") {
		t.Error("Should have ab flags.")
	}
	if !a.HasServerFlag(server, 'a') {
		t.Error("Should have a flag.")
	}
	if a.HasServerFlag(server, 'c') {
		t.Error("Should not have c flag.")
	}
}

func TestUserAccess_GrantChannel(t *T) {
	t.Parallel()
	a := createUserAccess()
	s := a.GetChannel(server, channel)
	if s != nil {
		t.Error("There should be no global access.")
	}

	a = createUserAccess()
	a.GrantChannel(server, channel, 100, "aB")
	s = a.GetChannel(server, channel)
	if s == nil {
		t.Error("There should be global access.")
	} else {
		if s.Level != 100 {
			t.Error("Level not set.")
		}
		if !s.HasFlag('a') || !s.HasFlag('B') {
			t.Error("Flags not added.")
		}
	}

	a = createUserAccess()
	a.GrantChannelLevel(server, channel, 100)
	s = a.GetChannel(server, channel)
	if s == nil {
		t.Error("There should be global access.")
	} else if s.Level != 100 {
		t.Error("Level not set.")
	}
	a = createUserAccess()
	a.GrantChannelFlags(server, channel, "aB")
	s = a.GetChannel(server, channel)
	if s == nil {
		t.Error("There should be global access.")
	} else if !s.HasFlag('a') || !s.HasFlag('B') {
		t.Error("Flags not added.")
	}
}

func TestUserAccess_RekoveChannel(t *T) {
	t.Parallel()
	a := createUserAccess()
	a.GrantChannel(server, channel, 100, "abc")
	if a.GetChannel(server, channel) == nil {
		t.Error("Channel permissions not granted.")
	}
	a.RevokeChannel(server, channel)
	if a.GetChannel(server, channel) != nil {
		t.Error("Channel permissions not revoked.")
	}

	a.GrantChannel(server, channel, 100, "abc")
	a.RevokeChannelLevel(server, channel)
	if a.GetChannel(server, channel).Level > 0 {
		t.Error("Channel level not revoked.")
	}

	a.RevokeChannelFlags(server, channel, "ab")
	if a.GetChannel(server, channel).HasFlags("ab") {
		t.Error("Channel flags not revoked.")
	}
}

func TestUserAccess_HasChannelLevel(t *T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasChannelLevel(server, channel, 50) {
		t.Error("Should not have any access.")
	}
	a.GrantChannelLevel(server, channel, 50)
	if !a.HasChannelLevel(server, channel, 50) {
		t.Error("Should have access.")
	}
	if a.HasChannelLevel(server, channel, 51) {
		t.Error("Should not have that high access.")
	}
}

func TestUserAccess_HasChannelFlags(t *T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasChannelFlags(server, channel, "ab") {
		t.Error("Should not have any flags.")
	}
	a.GrantChannelFlags(server, channel, "ab")
	if !a.HasChannelFlags(server, channel, "ab") {
		t.Error("Should have ab flags.")
	}
	if !a.HasChannelFlag(server, channel, 'a') {
		t.Error("Should have a flag.")
	}
	if a.HasChannelFlag(server, channel, 'c') {
		t.Error("Should not have c flag.")
	}
}