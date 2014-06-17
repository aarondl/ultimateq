package data

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestStoredUser(t *testing.T) {
	t.Parallel()
	var s *StoredUser
	var err error
	var masks = []string{`*!*@host`, `*!user@*`}

	s = &StoredUser{}
	s, err = NewStoredUser(uname, password, masks...)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if s.Username != uname {
		t.Errorf("Username is %v not %v", s.Username, uname)
	}
	if len(s.Password) == 0 {
		t.Error("Password not set properly:", err)
	}
	if len(s.Masks) != len(masks) {
		t.Errorf("Masks are %#v not %#v", s.Masks, masks)
	}

	s, err = NewStoredUser("", password, masks...)
	if s != nil || err != errMissingUnameOrPwd {
		t.Error("Empty username should fail creation.")
	}
	s, err = NewStoredUser(uname, "", masks...)
	if s != nil || err != errMissingUnameOrPwd {
		t.Error("Empty password should fail creation.")
	}
	s, err = NewStoredUser(uname, password, "a", "a")
	if s != nil || err != errDuplicateMask {
		t.Error("Duplicate masks should generate an error.")
	}
}

func TestStoredUser_VerifyPassword(t *testing.T) {
	t.Parallel()
	s, err := NewStoredUser(uname, password)
	if err != nil {
		t.Fatal("Unexpected Error:", err)
	}
	if !s.VerifyPassword(password) {
		t.Error("Real password was rejected.")
	}
	if s.VerifyPassword(password + password) {
		t.Error("Fake password was accepted.")
	}
}

func TestStoredUser_SerializeDeserialize(t *testing.T) {
	t.Parallel()
	var masks = []string{`*!*@host`, `*!user@*`}
	s, err := NewStoredUser(uname, password, masks...)
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}

	s.GrantGlobal(100, "a")
	s.GrantNetwork(network, 100, "a")
	s.GrantChannel(network, channel, 100, "a")

	serialized, err := s.serialize()
	if err != nil {
		t.Fatal("Unexpected error:", err)
	}
	if len(serialized) == 0 {
		t.Error("Serialization did not yield s serialized copy.")
	}

	b, err := deserializeUser(serialized)
	if err != nil {
		t.Fatal("Deserialization failed.")
	}
	if s.Username != b.Username || bytes.Compare(s.Password, b.Password) != 0 {
		t.Error("Username or Password did not deserializeUser.")
	}
	if len(s.Masks) != len(b.Masks) {
		t.Error("Masks were not serialized.")
	} else {
		for i := range s.Masks {
			if s.Masks[i] != b.Masks[i] {
				t.Errorf("Serialized mask not found:", s.Masks[i])
			}
		}
	}

	if !b.HasGlobalLevel(100) || !b.HasGlobalFlag('a') {
		t.Error("Lost global permissions in serialization.")
	}
	if !b.HasNetworkLevel(network, 100) || !b.HasNetworkFlag(network, 'a') {
		t.Error("Lost network permissions in serialization.")
	}
	if !b.HasChannelLevel(network, channel, 100) ||
		!b.HasChannelFlag(network, channel, 'a') {

		t.Error("Lost channel permissions in serialization.")
	}
}

func TestStoredUser_AddMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`}
	s := createStoredUser()
	if len(s.Masks) != 0 {
		t.Error("Masks should be empty.")
	}

	if !s.AddMask(masks[0]) && strings.ToLower(masks[0]) != s.Masks[0] {
		t.Error("The mask was not set correctly.")
	}
	if !s.AddMask(masks[1]) && strings.ToLower(masks[1]) != s.Masks[1] {
		t.Error("The mask was not set correctly.")
	}
	if s.AddMask(masks[0]) && len(s.Masks) > 2 {
		t.Error("The duplicate mask should not be accepted.")
	}
}

func TestStoredUser_DelMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`, `nick!*@*`}
	s := createStoredUser(masks...)
	if len(s.Masks) != 3 {
		t.Error("User should have the masks:", masks)
	}
	if !s.DelMask(masks[1]) || s.ValidateMask(masks[1]) {
		t.Error("The mask should have been deleted.")
	}
	if !s.DelMask(masks[2]) || s.ValidateMask(masks[2]) {
		t.Error("The mask should have been deleted.")
	}
	if len(s.Masks) != 1 {
		t.Error("Two masks should have been deleted.")
	}
}

func TestStoredUser_ValidateMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`}
	s := createStoredUser(masks[1:]...)

	if s.ValidateMask(masks[0]) {
		t.Error("Should not have validated this mask.")
	}
	if !s.ValidateMask(masks[1]) {
		t.Error("Should have validated this mask.")
	}

	s = createStoredUser(masks[1:]...)
	if !s.ValidateMask(masks[1]) {
		t.Error("When masks are empty should validate any mask.")
	}
}

func TestStoredUser_Has(t *testing.T) {
	t.Parallel()
	s := createStoredUser()

	var check = func(
		level uint8, flags string, has, hasLevel, hasFlags bool) string {

		if ret := s.Has(network, channel, level, flags); ret != has {
			return fmt.Sprintf("Expected (%v, %v) to be: %v but got %v",
				level, flags, has, ret)
		}
		if ret := s.HasLevel(network, channel, level); ret != hasLevel {
			return fmt.Sprintf("Expected level (%v) to be: %v but got %v",
				level, hasLevel, ret)
		}
		if ret := s.HasFlags(network, channel, flags); ret != hasFlags {
			return fmt.Sprintf("Expected flags (%v) to be: %v but got %v",
				flags, hasFlags, ret)
		}
		for _, f := range flags {
			if ret := s.HasFlag(network, channel, f); ret != hasFlags {
				return fmt.Sprintf("Expected flag (%v) to be: %v but got %v")
			}
		}
		return ""
	}

	var str string
	if str = check(1, "a", false, false, false); len(str) != 0 {
		t.Error(s)
	}
	s.GrantChannelFlags(network, channel, "a")
	if str = check(1, "a", false, false, true); len(str) != 0 {
		t.Error(s)
	}
	s.GrantChannelLevel(network, channel, 1)
	if str = check(1, "a", true, true, true); len(str) != 0 {
		t.Error(s)
	}

	s.GrantNetworkFlags(network, "b")
	if str = check(2, "ab", false, false, true); len(str) != 0 {
		t.Error(s)
	}
	s.GrantNetworkLevel(network, 2)
	if str = check(2, "ab", true, true, true); len(str) != 0 {
		t.Error(s)
	}

	s.GrantGlobalFlags("c")
	if str = check(3, "abc", false, false, true); len(str) != 0 {
		t.Error(s)
	}
	s.GrantGlobalLevel(3)
	if str = check(3, "abc", true, true, true); len(str) != 0 {
		t.Error(s)
	}

	if s.HasFlags(network, channel, "ad") == false {
		t.Error("Should have had flag s.")
	}
}

func TestStoredUser_GrantGlobal(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.GrantGlobalLevel(100)
	a := s.GetGlobal()
	if a.Level != 100 {
		t.Error("Level not set.")
	}

	s = createStoredUser()
	s.GrantGlobalFlags("aB")
	a = s.GetGlobal()
	if !a.HasFlag('a') || !a.HasFlag('a') {
		t.Error("Flags not set.")
	}

	s = createStoredUser()
	s.GrantGlobal(100, "aB")
	a = s.GetGlobal()
	if a.Level != 100 {
		t.Error("Level not set.")
	}
	if !a.HasFlag('a') || !a.HasFlag('a') {
		t.Error("Flags not set.")
	}
}

func TestStoredUser_RevokeGlobal(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.GrantGlobal(100, "aB")
	s.RevokeGlobalLevel()
	if s.Global.Level != 0 {
		t.Error("Level not revoked.")
	}
	s.RevokeGlobalFlags("a")
	if s.Global.HasFlag('a') || !s.Global.HasFlag('B') {
		t.Error("Flags not revoked.")
	}
	s.RevokeGlobal()
	if s.Global != nil {
		t.Error("Global should be nil.")
	}
}

func TestStoredUser_HasGlobalLevel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasGlobalLevel(50) {
		t.Error("Should not have any access.")
	}
	s.GrantGlobalLevel(50)
	if !s.HasGlobalLevel(50) {
		t.Error("Should have access.")
	}
	if s.HasGlobalLevel(51) {
		t.Error("Should not have that high access.")
	}
}

func TestStoredUser_HasGlobalFlags(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasGlobalFlags("ab") {
		t.Error("Should not have any flags.")
	}
	s.GrantGlobalFlags("ab")
	if !s.HasGlobalFlag('a') || !s.HasGlobalFlag('b') {
		t.Error("Should have ab flags.")
	}
	if !s.HasGlobalFlags("abc") {
		t.Error("Should have s or b flags.")
	}
	if s.HasGlobalFlag('c') {
		t.Error("Should not have c flag.")
	}
}

func TestStoredUser_GrantNetwork(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	a := s.GetNetwork(network)
	if a != nil {
		t.Error("There should be no network access.")
	}

	s = createStoredUser()
	s.GrantNetwork(network, 100, "aB")
	a = s.GetNetwork(network)
	if a == nil {
		t.Error("There should be network access.")
	} else {
		if a.Level != 100 {
			t.Error("Level not set.")
		}
		if !a.HasFlag('a') || !a.HasFlag('B') {
			t.Error("Flags not added.")
		}
	}

	s = createStoredUser()
	s.GrantNetworkLevel(network, 100)
	a = s.GetNetwork(network)
	if s == nil {
		t.Error("There should be network access.")
	} else if a.Level != 100 {
		t.Error("Level not set.")
	}
	s = createStoredUser()
	s.GrantNetworkFlags(network, "aB")
	a = s.GetNetwork(network)
	if a == nil {
		t.Error("There should be network access.")
	} else if !a.HasFlag('a') || !a.HasFlag('B') {
		t.Error("Flags not added.")
	}
}

func TestStoredUser_RevokeNetwork(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.GrantNetwork(network, 100, "abc")
	if s.GetNetwork(network) == nil {
		t.Error("Network permissions not granted.")
	}
	s.RevokeNetwork(network)
	if s.GetNetwork(network) != nil {
		t.Error("Network permissions not revoked.")
	}

	s.GrantNetwork(network, 100, "abc")
	s.RevokeNetworkLevel(network)
	if s.GetNetwork(network).Level > 0 {
		t.Error("Network level not revoked.")
	}

	s.RevokeNetworkFlags(network, "ab")
	if s.GetNetwork(network).HasFlags("ab") {
		t.Error("Network flags not revoked.")
	}
}

func TestStoredUser_HasNetworkLevel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasNetworkLevel(network, 50) {
		t.Error("Should not have any access.")
	}
	s.GrantNetworkLevel(network, 50)
	if !s.HasNetworkLevel(network, 50) {
		t.Error("Should have access.")
	}
	if s.HasNetworkLevel(network, 51) {
		t.Error("Should not have that high access.")
	}
}

func TestStoredUser_HasNetworkFlags(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasNetworkFlags(network, "ab") {
		t.Error("Should not have any flags.")
	}
	s.GrantNetworkFlags(network, "ab")
	if !s.HasNetworkFlag(network, 'a') || !s.HasNetworkFlag(network, 'b') {
		t.Error("Should have ab flags.")
	}
	if !s.HasNetworkFlags(network, "abc") {
		t.Error("Should have s or b flags.")
	}
	if s.HasNetworkFlag(network, 'c') {
		t.Error("Should not have c flag.")
	}
}

func TestStoredUser_GrantChannel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	a := s.GetChannel(network, channel)
	if a != nil {
		t.Error("There should be no global access.")
	}

	s = createStoredUser()
	s.GrantChannel(network, channel, 100, "aB")
	a = s.GetChannel(network, channel)
	if a == nil {
		t.Error("There should be global access.")
	} else {
		if a.Level != 100 {
			t.Error("Level not set.")
		}
		if !a.HasFlag('a') || !a.HasFlag('B') {
			t.Error("Flags not added.")
		}
	}

	s = createStoredUser()
	s.GrantChannelLevel(network, channel, 100)
	a = s.GetChannel(network, channel)
	if a == nil {
		t.Error("There should be global access.")
	} else if a.Level != 100 {
		t.Error("Level not set.")
	}
	s = createStoredUser()
	s.GrantChannelFlags(network, channel, "aB")
	a = s.GetChannel(network, channel)
	if a == nil {
		t.Error("There should be global access.")
	} else if !a.HasFlag('a') || !a.HasFlag('B') {
		t.Error("Flags not added.")
	}
}

func TestStoredUser_RevokeChannel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.GrantChannel(network, channel, 100, "abc")
	if s.GetChannel(network, channel) == nil {
		t.Error("Channel permissions not granted.")
	}
	s.RevokeChannel(network, channel)
	if s.GetChannel(network, channel) != nil {
		t.Error("Channel permissions not revoked.")
	}

	s.GrantChannel(network, channel, 100, "abc")
	s.RevokeChannelLevel(network, channel)
	if s.GetChannel(network, channel).Level > 0 {
		t.Error("Channel level not revoked.")
	}

	s.RevokeChannelFlags(network, channel, "ab")
	if s.GetChannel(network, channel).HasFlags("ab") {
		t.Error("Channel flags not revoked.")
	}
}

func TestStoredUser_HasChannelLevel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasChannelLevel(network, channel, 50) {
		t.Error("Should not have any access.")
	}
	s.GrantChannelLevel(network, channel, 50)
	if !s.HasChannelLevel(network, channel, 50) {
		t.Error("Should have access.")
	}
	if s.HasChannelLevel(network, channel, 51) {
		t.Error("Should not have that high access.")
	}
}

func TestStoredUser_HasChannelFlags(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasChannelFlags(network, channel, "ab") {
		t.Error("Should not have any flags.")
	}
	s.GrantChannelFlags(network, channel, "ab")
	if !s.HasChannelFlag(network, channel, 'a') ||
		!s.HasChannelFlag(network, channel, 'b') {
		t.Error("Should have ab flags.")
	}
	if !s.HasChannelFlags(network, channel, "abc") {
		t.Error("Should have s or b flags.")
	}
	if s.HasChannelFlag(network, channel, 'c') {
		t.Error("Should not have c flag.")
	}
}

func TestStoredUser_String(t *testing.T) {
	t.Parallel()

	var table = []struct {
		HasGlobal       bool
		GlobalLevel     uint8
		GlobalFlags     string
		HasNetwork      bool
		NetworkLevel    uint8
		NetworkFlags    string
		HasChannel      bool
		ChannelLevel    uint8
		ChannelFlags    string
		ExpectChannel   string
		ExpectNoChannel string
	}{
		{true, 100, "abc", true, 150, "abc", true, 200, "abc",
			"G(100 abc),S(150 abc),#chan1(200 abc)",
			"G(100 abc),S(150 abc),#chan1(200 abc),#chan2(200 abc)"},
		{false, 100, "abc", true, 150, "abc", true, 200, "abc",
			"S(150 abc),#chan1(200 abc)",
			"S(150 abc),#chan1(200 abc),#chan2(200 abc)"},
		{false, 0, "", false, 0, "", true, 200, "abc",
			"#chan1(200 abc)",
			"#chan1(200 abc),#chan2(200 abc)"},
		{false, 0, "", false, 0, "", false, 0, "", "none", "none"},
		{false, 0, "", false, 0, "", true, 0, "", "none", "none"},
	}

	for _, test := range table {
		s := createStoredUser()
		if test.HasGlobal {
			s.GrantGlobal(test.GlobalLevel, test.GlobalFlags)
		}
		if test.HasNetwork {
			s.GrantNetwork(network, test.NetworkLevel, test.NetworkFlags)
			s.GrantNetwork("other", test.NetworkLevel, test.NetworkFlags)
		}
		if test.HasChannel {
			s.GrantChannel(network, "#chan1",
				test.ChannelLevel, test.ChannelFlags)
			s.GrantChannel(network, "#chan2",
				test.ChannelLevel, test.ChannelFlags)
		}

		var was string
		var expList []string

		was = s.String(network, "#chan1")
		expList = strings.Split(test.ExpectChannel, ",")
		for _, exp := range expList {
			if !strings.Contains(was, exp) {
				t.Errorf("Expected: '%s' to contain access: '%s'", was, exp)
			}
		}

		was = s.String(network, "")
		expList = strings.Split(test.ExpectChannel, ",")
		for _, exp := range expList {
			if !strings.Contains(was, exp) {
				t.Errorf("Expected: '%s' to contain access: '%s'", was, exp)
			}
		}
	}
}

func TestStoredUser_ResetPassword(t *testing.T) {
	t.Parallel()
	s, err := NewStoredUser(uname, password)
	if err != nil {
		t.Error(err)
	}
	oldpasswd := s.Password
	newpasswd, err := s.ResetPassword()
	if err != nil {
		t.Error(err)
	}
	if newpasswd == password {
		t.Error("Not very random password occurred.")
	}
	if bytes.Compare(oldpasswd, s.Password) == 0 {
		t.Error("Password not set correctly.")
	}
	if m, err := regexp.MatchString("^[A-Za-z0-9]+$", newpasswd); err != nil {
		t.Error("Regular Expression did not compile.")
	} else if !m {
		t.Error("New password was malformed:", newpasswd)
	}
}
