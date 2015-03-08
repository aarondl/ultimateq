package data

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestStoredUser(t *testing.T) {
	t.Parallel()

	var masks = []string{`*!*@host`, `*!user@*`}

	s, err := NewStoredUser(uname, password, masks...)
	if err != nil {
		t.Error(err)
	}
	s.Grant("", "", 15)
	s.Put("hello", "there")

	clone := s.Clone()

	if clone.Username != s.Username {
		t.Error("Expected username to be cloned, got:", clone.Username)
	}
	if 0 != bytes.Compare(clone.Password, s.Password) {
		t.Errorf("Expected password to be cloned, got: %s", clone.Password)
	}
	if clone.Masks[0] != masks[0] || clone.Masks[1] != masks[1] {
		t.Error("Expected masks to be cloned.")
	}
	if val, _ := clone.Get("hello"); val != "there" {
		t.Error("Expected JSON storage to be copied.")
	}
	if !clone.HasLevel("", "", 15) {
		t.Error("Expected access to be cloned.")
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

	s.Grant("", "", 100, "a")
	s.Grant(network, "", 100, "a")
	s.Grant(network, channel, 100, "a")

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

	if !b.HasLevel("", "", 100) || !b.HasFlags("", "", "a") {
		t.Error("Lost global permissions in serialization.")
	}
	if !b.HasLevel(network, "", 100) || !b.HasFlags(network, "", "a") {
		t.Error("Lost network permissions in serialization.")
	}
	if !b.HasLevel(network, channel, 100) ||
		!b.HasFlags(network, channel, "a") {

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

func TestStoredUser_RemoveMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`, `nick!*@*`}
	s := createStoredUser(masks...)
	if len(s.Masks) != 3 {
		t.Error("User should have the masks:", masks)
	}
	if !s.RemoveMask(masks[1]) || s.HasMask(masks[1]) {
		t.Error("The mask should have been deleted.")
	}
	if !s.RemoveMask(masks[2]) || s.HasMask(masks[2]) {
		t.Error("The mask should have been deleted.")
	}
	if len(s.Masks) != 1 {
		t.Error("Two masks should have been deleted.")
	}
}

func TestStoredUser_HasMasks(t *testing.T) {
	t.Parallel()
	masks := []string{`*!*@host`, `*!user@*`}
	s := createStoredUser(masks[1:]...)

	if s.HasMask(masks[0]) {
		t.Error("Should not have validated this mask.")
	}
	if !s.HasMask(masks[1]) {
		t.Error("Should have validated this mask.")
	}

	s = createStoredUser(masks[1:]...)
	if !s.HasMask(masks[1]) {
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
		return ""
	}

	var str string
	if str = check(1, "a", false, false, false); len(str) != 0 {
		t.Error(str)
	}
	s.Grant(network, channel, 0, "a")
	if str = check(1, "a", false, false, true); len(str) != 0 {
		t.Error(str)
	}
	s.Grant(network, channel, 1)
	if str = check(1, "a", true, true, true); len(str) != 0 {
		t.Error(str)
	}
	s.Grant(network, "", 0, "b")
	if str = check(2, "ab", false, false, true); len(str) != 0 {
		t.Error(str)
	}
	s.Grant(network, "", 2)
	if str = check(2, "ab", true, true, true); len(str) != 0 {
		t.Error(str)
	}
	s.Grant("", "", 0, "c")
	if str = check(3, "abc", false, false, true); len(str) != 0 {
		t.Error(str)
	}
	s.Grant("", "", 3)
	if str = check(3, "abc", true, true, true); len(str) != 0 {
		t.Error(str)
	}

	if s.HasFlags(network, channel, "ad") {
		t.Error("Should not have had flag d.")
	}
}

func TestStoredUser_GrantGlobal(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.Grant("", "", 100)
	a, ok := s.GetAccess("", "")
	if !ok || a.Level != 100 {
		t.Error("Level not set.")
	}

	s = createStoredUser()
	s.Grant("", "", 0, "aB")
	a, ok = s.GetAccess("", "")
	if !ok || !a.HasFlag('a') || !a.HasFlag('a') {
		t.Error("Flags not set.")
	}

	s = createStoredUser()
	s.Grant("", "", 100, "aB")
	a, ok = s.GetAccess("", "")
	if !ok || a.Level != 100 {
		t.Error("Level not set.")
	}
	if !ok || !a.HasFlag('a') || !a.HasFlag('a') {
		t.Error("Flags not set.")
	}
}

func TestStoredUser_RevokeGlobal(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.Grant("", "", 100, "aB")
	s.RevokeLevel("", "")
	a, _ := s.GetAccess("", "")
	if a.Level != 0 {
		t.Error("Level not revoked.")
	}
	s.RevokeFlags("", "", "a")
	a, _ = s.GetAccess("", "")
	if a.HasFlag('a') || !a.HasFlag('B') {
		t.Error("Flags not revoked.")
	}
	s.Revoke("", "")
	_, ok := s.GetAccess("", "")
	if ok {
		t.Error("Global should be nil.")
	}
}

func TestStoredUser_HasGlobalLevel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasLevel("", "", 50) {
		t.Error("Should not have any access.")
	}
	s.Grant("", "", 50)
	if !s.HasLevel("", "", 50) {
		t.Error("Should have access.")
	}
	if s.HasLevel("", "", 51) {
		t.Error("Should not have that high access.")
	}
}

func TestStoredUser_HasGlobalFlags(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasFlags("", "", "ab") {
		t.Error("Should not have any flags.")
	}
	s.Grant("", "", 0, "ab")
	if !s.HasFlags("", "", "a") || !s.HasFlags("", "", "b") {
		t.Error("Should have ab flags.")
	}
	if s.HasFlags("", "", "abc") {
		t.Error("Should not have c flag.")
	}
	if s.HasFlags("", "", "c") {
		t.Error("Should not have c flag.")
	}
}

func TestStoredUser_GrantNetwork(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	a, ok := s.GetAccess(network, "")
	if ok {
		t.Error("There should be no network access.")
	}

	s = createStoredUser()
	s.Grant(network, "", 100, "aB")
	a, ok = s.GetAccess(network, "")
	if !ok {
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
	s.Grant(network, "", 100)
	a, ok = s.GetAccess(network, "")
	if !ok {
		t.Error("There should be network access.")
	} else if a.Level != 100 {
		t.Error("Level not set.")
	}
	s = createStoredUser()
	s.Grant(network, "", 0, "aB")
	a, ok = s.GetAccess(network, "")
	if !ok {
		t.Error("There should be network access.")
	} else if !a.HasFlag('a') || !a.HasFlag('B') {
		t.Error("Flags not added.")
	}
}

func TestStoredUser_RevokeNetwork(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.Grant(network, "", 100, "abc")
	if _, ok := s.GetAccess(network, ""); !ok {
		t.Error("Network permissions not granted.")
	}
	s.Revoke(network, "")
	if _, ok := s.GetAccess(network, ""); ok {
		t.Error("Network permissions not revoked.")
	}

	s.Grant(network, "", 100, "abc")
	s.RevokeLevel(network, "")
	if a, ok := s.GetAccess(network, ""); !ok || a.Level > 0 {
		t.Error("Network level not revoked.")
	}

	s.RevokeFlags(network, "", "ab")
	if a, ok := s.GetAccess(network, ""); !ok || a.HasFlags("ab") {
		t.Error("Network flags not revoked.")
	}
}

func TestStoredUser_HasNetworkLevel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasLevel(network, "", 50) {
		t.Error("Should not have any access.")
	}
	s.Grant(network, "", 50)
	if !s.HasLevel(network, "", 50) {
		t.Error("Should have access.")
	}
	if s.HasLevel(network, "", 51) {
		t.Error("Should not have that high access.")
	}
}

func TestStoredUser_HasNetworkFlags(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasFlags(network, "", "ab") {
		t.Error("Should not have any flags.")
	}
	s.Grant(network, "", 0, "ab")
	if !s.HasFlags(network, "", "a") || !s.HasFlags(network, "", "b") {
		t.Error("Should have ab flags.")
	}
	if !s.HasFlags(network, "", "ab") {
		t.Error("Should have ab flags.")
	}
	if s.HasFlags(network, "", "abc") {
		t.Error("Should not have c flag.")
	}
	if s.HasFlags(network, "", "c") {
		t.Error("Should not have c flag.")
	}
}

func TestStoredUser_GrantChannel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	a, ok := s.GetAccess(network, channel)
	if ok {
		t.Error("There should be no channel access.")
	}

	s = createStoredUser()
	s.Grant(network, channel, 100, "aB")
	a, ok = s.GetAccess(network, channel)
	if !ok {
		t.Error("There should be channel access.")
	} else {
		if a.Level != 100 {
			t.Error("Level not set.")
		}
		if !a.HasFlag('a') || !a.HasFlag('B') {
			t.Error("Flags not added.")
		}
	}

	s = createStoredUser()
	s.Grant(network, channel, 100)
	a, ok = s.GetAccess(network, channel)
	if !ok {
		t.Error("There should be channel access.")
	} else if a.Level != 100 {
		t.Error("Level not set.")
	}
	s = createStoredUser()
	s.Grant(network, channel, 0, "aB")
	a, ok = s.GetAccess(network, channel)
	if !ok {
		t.Error("There should be channel access.")
	} else if !a.HasFlag('a') || !a.HasFlag('B') {
		t.Error("Flags not added.")
	}
}

func TestStoredUser_RevokeChannel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	s.Grant(network, channel, 100, "abc")
	if _, ok := s.GetAccess(network, channel); !ok {
		t.Error("Channel permissions not granted.")
	}
	s.Revoke(network, channel)
	if _, ok := s.GetAccess(network, channel); ok {
		t.Error("Channel permissions not revoked.")
	}

	s.Grant(network, channel, 100, "abc")
	s.RevokeLevel(network, channel)
	if a, ok := s.GetAccess(network, channel); !ok || a.Level > 0 {
		t.Error("Channel level not revoked.")
	}

	s.RevokeFlags(network, channel, "ab")
	if a, ok := s.GetAccess(network, channel); !ok || a.HasFlags("ab") {
		t.Error("Channel flags not revoked.")
	}
}

func TestStoredUser_HasChannelLevel(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasLevel(network, channel, 50) {
		t.Error("Should not have any access.")
	}
	s.Grant(network, channel, 50)
	if !s.HasLevel(network, channel, 50) {
		t.Error("Should have access.")
	}
	if s.HasLevel(network, channel, 51) {
		t.Error("Should not have that high access.")
	}
}

func TestStoredUser_HasChannelFlags(t *testing.T) {
	t.Parallel()
	s := createStoredUser()
	if s.HasFlags(network, channel, "ab") {
		t.Error("Should not have any flags.")
	}
	s.Grant(network, channel, 0, "ab")
	if !s.HasFlags(network, channel, "a") ||
		!s.HasFlags(network, channel, "b") {
		t.Error("Should have ab flags.")
	}
	if s.HasFlags(network, channel, "abc") {
		t.Error("Should not have c flag.")
	}
	if s.HasFlags(network, channel, "c") {
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
			"G(100 abc),irc.network.net(150 abc),irc.network.net:#chan1(200 abc)",
			"G(100 abc),irc.network.net(150 abc),irc.network.net:#chan1(200 abc),irc.network.net:#chan2(200 abc)"},
		{false, 100, "abc", true, 150, "abc", true, 200, "abc",
			"irc.network.net(150 abc),irc.network.net:#chan1(200 abc)",
			"irc.network.net(150 abc),irc.network.net:#chan1(200 abc),irc.network.net:#chan2(200 abc)"},
		{false, 0, "", false, 0, "", true, 200, "abc",
			"irc.network.net:#chan1(200 abc)",
			"irc.network.net:#chan1(200 abc),irc.network.net:#chan2(200 abc)"},
		{false, 0, "", false, 0, "", false, 0, "", "none", "none"},
		{false, 0, "", false, 0, "", true, 0, "", "none", "none"},
	}

	for i, test := range table {
		s := createStoredUser()
		if test.HasGlobal {
			s.Grant("", "", test.GlobalLevel, test.GlobalFlags)
		}
		if test.HasNetwork {
			s.Grant(network, "", test.NetworkLevel, test.NetworkFlags)
			s.Grant("other", "", test.NetworkLevel, test.NetworkFlags)
		}
		if test.HasChannel {
			s.Grant(network, "#chan1",
				test.ChannelLevel, test.ChannelFlags)
			s.Grant(network, "#chan2",
				test.ChannelLevel, test.ChannelFlags)
		}
		spew.Dump(s.Access)

		var was string
		var expList []string

		was = s.String(network, "#chan1")
		expList = strings.Split(test.ExpectChannel, ",")
		for _, exp := range expList {
			if !strings.Contains(was, exp) {
				t.Errorf("1.%d) Expected: '%s' to contain access: '%s'", i, was, exp)
			}
		}

		was = s.String(network, "")
		expList = strings.Split(test.ExpectChannel, ",")
		for _, exp := range expList {
			if !strings.Contains(was, exp) {
				t.Errorf("2.%d) Expected: '%s' to contain access: '%s'", i, was, exp)
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
