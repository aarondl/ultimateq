package data

import (
	"github.com/aarondl/ultimateq/irc"
	. "testing"
)

// var server = "irc.server.net - in state_test.go
var channel = "#chan"

func TestUserAccess(t *T) {
	t.Parallel()
	var a *UserAccess
	masks := []irc.WildMask{`*!*@host`, `*!user@*`}

	a = CreateUserAccess(masks)
	if len(a.Masks) != len(masks) {
		t.Error("Masks not set right.")
	}
}

func TestUserAccess_AddMask(t *T) {
	t.Parallel()
	var mask irc.WildMask = "*!*@host"
	a := CreateUserAccess(nil)
	if len(a.Masks) != 0 {
		t.Error("Masks should be empty.")
	}
	a.AddMask(mask)
	if len(a.Masks) != 1 || a.Masks[0] != mask {
		t.Error("Masks should have:", mask)
	}
}

func TestUserAccess_DelMask(t *T) {
	t.Parallel()
	var mask irc.WildMask = "*!*@host"
	a := CreateUserAccess(nil)
	a.AddMask(mask)
	if len(a.Masks) != 1 || a.Masks[0] != mask {
		t.Error("Masks should have:", mask)
	}
	a.DelMask(mask)
	if len(a.Masks) != 0 {
		t.Error("Masks should be empty.")
	}
}

func TestUserAccess_IsMatch(t *T) {
	t.Parallel()
	var wmasks = []irc.WildMask{"*!*@host1", "*!user2@*"}
	var mask1, mask2 irc.Mask = "nick1!user1@host1", "nick2!user2@host2"
	a := CreateUserAccess(nil)
	if a.IsMatch(mask1) || a.IsMatch(mask2) {
		t.Error("No masks should match.")
	}

	a = CreateUserAccess(wmasks)
	if !a.IsMatch(mask1) || !a.IsMatch(mask2) {
		t.Error(mask1, "and", mask2, "should match")
	}
}

func TestUserAccess_GrantGlobal(t *T) {
	t.Parallel()
	a := CreateUserAccess(nil)
	a.GrantGlobalLevel(100)
	s := a.GetGlobal()
	if s.Level != 100 {
		t.Error("Level not set.")
	}

	a = CreateUserAccess(nil)
	a.GrantGlobalFlags("aB")
	s = a.GetGlobal()
	if !s.HasFlag('a') || !s.HasFlag('a') {
		t.Error("Flags not set.")
	}

	a = CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
	s := a.GetServer(server)
	if s != nil {
		t.Error("There should be no server access.")
	}

	a = CreateUserAccess(nil)
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

	a = CreateUserAccess(nil)
	a.GrantServerLevel(server, 100)
	s = a.GetServer(server)
	if s == nil {
		t.Error("There should be server access.")
	} else if s.Level != 100 {
		t.Error("Level not set.")
	}
	a = CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
	s := a.GetChannel(server, channel)
	if s != nil {
		t.Error("There should be no global access.")
	}

	a = CreateUserAccess(nil)
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

	a = CreateUserAccess(nil)
	a.GrantChannelLevel(server, channel, 100)
	s = a.GetChannel(server, channel)
	if s == nil {
		t.Error("There should be global access.")
	} else if s.Level != 100 {
		t.Error("Level not set.")
	}
	a = CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
	a := CreateUserAccess(nil)
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
