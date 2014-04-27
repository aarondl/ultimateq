package data

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestUserAccess(t *testing.T) {
	t.Parallel()
	var a *UserAccess
	var err error
	var masks = []string{`*!*@host`, `*!user@*`}

	a = &UserAccess{}
	a, err = NewUserAccess(uname, password, masks...)
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

	a, err = NewUserAccess("", password, masks...)
	if a != nil || err != errMissingUnameOrPwd {
		t.Error("Empty username should fail creation.")
	}
	a, err = NewUserAccess(uname, "", masks...)
	if a != nil || err != errMissingUnameOrPwd {
		t.Error("Empty password should fail creation.")
	}
	a, err = NewUserAccess(uname, password, "a", "a")
	if a != nil || err != errDuplicateMask {
		t.Error("Duplicate masks should generate an error.")
	}
}

func TestUserAccess_VerifyPassword(t *testing.T) {
	t.Parallel()
	a, err := NewUserAccess(uname, password)
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

func TestUserAccess_SerializeDeserialize(t *testing.T) {
	var masks = []string{`*!*@host`, `*!user@*`}
	a, err := NewUserAccess(uname, password, masks...)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	a.GrantGlobal(100, "a")
	a.GrantServer(server, 100, "a")
	a.GrantChannel(server, channel, 100, "a")

	serialized, err := a.serialize()
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

func TestUserAccess_AddMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`}
	a := createUserAccess()
	if len(a.Masks) != 0 {
		t.Error("Masks should be empty.")
	}

	if !a.AddMask(masks[0]) && strings.ToLower(masks[0]) != a.Masks[0] {
		t.Error("The mask was not set correctly.")
	}
	if !a.AddMask(masks[1]) && strings.ToLower(masks[1]) != a.Masks[1] {
		t.Error("The mask was not set correctly.")
	}
	if a.AddMask(masks[0]) && len(a.Masks) > 2 {
		t.Error("The duplicate mask should not be accepted.")
	}
}

func TestUserAccess_DelMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`, `nick!*@*`}
	a := createUserAccess(masks...)
	if len(a.Masks) != 3 {
		t.Error("User should have the masks:", masks)
	}
	if !a.DelMask(masks[1]) || a.ValidateMask(masks[1]) {
		t.Error("The mask should have been deleted.")
	}
	if !a.DelMask(masks[2]) || a.ValidateMask(masks[2]) {
		t.Error("The mask should have been deleted.")
	}
	if len(a.Masks) != 1 {
		t.Error("Two masks should have been deleted.")
	}
}

func TestUserAccess_ValidateMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`}
	a := createUserAccess(masks[1:]...)

	if a.ValidateMask(masks[0]) {
		t.Error("Should not have validated this mask.")
	}
	if !a.ValidateMask(masks[1]) {
		t.Error("Should have validated this mask.")
	}

	a = createUserAccess(masks[1:]...)
	if !a.ValidateMask(masks[1]) {
		t.Error("When masks are empty should validate any mask.")
	}
}

func TestUserAccess_Has(t *testing.T) {
	t.Parallel()
	a := createUserAccess()

	var check = func(
		level uint8, flags string, has, hasLevel, hasFlags bool) string {

		if ret := a.Has(server, channel, level, flags); ret != has {
			return fmt.Sprintf("Expected (%v, %v) to be: %v but got %v",
				level, flags, has, ret)
		}
		if ret := a.HasLevel(server, channel, level); ret != hasLevel {
			return fmt.Sprintf("Expected level (%v) to be: %v but got %v",
				level, hasLevel, ret)
		}
		if ret := a.HasFlags(server, channel, flags); ret != hasFlags {
			return fmt.Sprintf("Expected flags (%v) to be: %v but got %v",
				flags, hasFlags, ret)
		}
		for _, f := range flags {
			if ret := a.HasFlag(server, channel, f); ret != hasFlags {
				return fmt.Sprintf("Expected flag (%v) to be: %v but got %v")
			}
		}
		return ""
	}

	var s string
	if s = check(1, "a", false, false, false); len(s) != 0 {
		t.Error(s)
	}
	a.GrantChannelFlags(server, channel, "a")
	if s = check(1, "a", false, false, true); len(s) != 0 {
		t.Error(s)
	}
	a.GrantChannelLevel(server, channel, 1)
	if s = check(1, "a", true, true, true); len(s) != 0 {
		t.Error(s)
	}

	a.GrantServerFlags(server, "b")
	if s = check(2, "ab", false, false, true); len(s) != 0 {
		t.Error(s)
	}
	a.GrantServerLevel(server, 2)
	if s = check(2, "ab", true, true, true); len(s) != 0 {
		t.Error(s)
	}

	a.GrantGlobalFlags("c")
	if s = check(3, "abc", false, false, true); len(s) != 0 {
		t.Error(s)
	}
	a.GrantGlobalLevel(3)
	if s = check(3, "abc", true, true, true); len(s) != 0 {
		t.Error(s)
	}

	if a.HasFlags(server, channel, "ad") == false {
		t.Error("Should have had flag a.")
	}
}

func TestUserAccess_GrantGlobal(t *testing.T) {
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

func TestUserAccess_RevokeGlobal(t *testing.T) {
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

func TestUserAccess_HasGlobalLevel(t *testing.T) {
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

func TestUserAccess_HasGlobalFlags(t *testing.T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasGlobalFlags("ab") {
		t.Error("Should not have any flags.")
	}
	a.GrantGlobalFlags("ab")
	if !a.HasGlobalFlag('a') || !a.HasGlobalFlag('b') {
		t.Error("Should have ab flags.")
	}
	if !a.HasGlobalFlags("abc") {
		t.Error("Should have a or b flags.")
	}
	if a.HasGlobalFlag('c') {
		t.Error("Should not have c flag.")
	}
}

func TestUserAccess_GrantServer(t *testing.T) {
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

func TestUserAccess_RevokeServer(t *testing.T) {
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

func TestUserAccess_HasServerLevel(t *testing.T) {
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

func TestUserAccess_HasServerFlags(t *testing.T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasServerFlags(server, "ab") {
		t.Error("Should not have any flags.")
	}
	a.GrantServerFlags(server, "ab")
	if !a.HasServerFlag(server, 'a') || !a.HasServerFlag(server, 'b') {
		t.Error("Should have ab flags.")
	}
	if !a.HasServerFlags(server, "abc") {
		t.Error("Should have a or b flags.")
	}
	if a.HasServerFlag(server, 'c') {
		t.Error("Should not have c flag.")
	}
}

func TestUserAccess_GrantChannel(t *testing.T) {
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

func TestUserAccess_RevokeChannel(t *testing.T) {
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

func TestUserAccess_HasChannelLevel(t *testing.T) {
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

func TestUserAccess_HasChannelFlags(t *testing.T) {
	t.Parallel()
	a := createUserAccess()
	if a.HasChannelFlags(server, channel, "ab") {
		t.Error("Should not have any flags.")
	}
	a.GrantChannelFlags(server, channel, "ab")
	if !a.HasChannelFlag(server, channel, 'a') ||
		!a.HasChannelFlag(server, channel, 'b') {
		t.Error("Should have ab flags.")
	}
	if !a.HasChannelFlags(server, channel, "abc") {
		t.Error("Should have a or b flags.")
	}
	if a.HasChannelFlag(server, channel, 'c') {
		t.Error("Should not have c flag.")
	}
}

func TestUserAccess_String(t *testing.T) {
	var table = []struct {
		HasGlobal       bool
		GlobalLevel     uint8
		GlobalFlags     string
		HasServer       bool
		ServerLevel     uint8
		ServerFlags     string
		HasChannel      bool
		ChannelLevel    uint8
		ChannelFlags    string
		ExpectChannel   string
		ExpectNoChannel string
	}{
		{true, 100, "abc", true, 150, "abc", true, 200, "abc",
			"G(100 abc) S(150 abc) #chan1(200 abc)",
			"G(100 abc) S(150 abc) #chan1(200 abc) #chan2(200 abc)"},
		{false, 100, "abc", true, 150, "abc", true, 200, "abc",
			"S(150 abc) #chan1(200 abc)",
			"S(150 abc) #chan1(200 abc) #chan2(200 abc)"},
		{false, 0, "", false, 0, "", true, 200, "abc",
			"#chan1(200 abc)",
			"#chan1(200 abc) #chan2(200 abc)"},
		{false, 0, "", false, 0, "", false, 0, "", "none", "none"},
		{false, 0, "", false, 0, "", true, 0, "", "none", "none"},
	}

	for _, test := range table {
		a := createUserAccess()
		if test.HasGlobal {
			a.GrantGlobal(test.GlobalLevel, test.GlobalFlags)
		}
		if test.HasServer {
			a.GrantServer(server, test.ServerLevel, test.ServerFlags)
			a.GrantServer("other", test.ServerLevel, test.ServerFlags)
		}
		if test.HasChannel {
			a.GrantChannel(server, "#chan1",
				test.ChannelLevel, test.ChannelFlags)
			a.GrantChannel(server, "#chan2",
				test.ChannelLevel, test.ChannelFlags)
		}

		if was := a.String(server, "#chan1"); was != test.ExpectChannel {
			t.Errorf("Wrong output:\n\twant:'%s'\n\twas: '%s'",
				test.ExpectChannel, was)
		}
		if was := a.String(server, ""); was != test.ExpectNoChannel {
			t.Errorf("Wrong output:\n\twant:'%s'\n\twas: '%s'",
				test.ExpectNoChannel, was)
		}
	}
}

func TestUserAccess_ResetPassword(t *testing.T) {
	t.Parallel()
	a, err := NewUserAccess(uname, password)
	if err != nil {
		t.Error(err)
	}
	oldpasswd := a.Password
	newpasswd, err := a.ResetPassword()
	if err != nil {
		t.Error(err)
	}
	if newpasswd == password {
		t.Error("Not very random password occurred.")
	}
	if bytes.Compare(oldpasswd, a.Password) == 0 {
		t.Error("Password not set correctly.")
	}
	if m, err := regexp.MatchString("^[A-Za-z0-9]+$", newpasswd); err != nil {
		t.Error("Regular Expression did not compile.")
	} else if !m {
		t.Error("New password was malformed:", newpasswd)
	}
}
