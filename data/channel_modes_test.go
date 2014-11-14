package data

import (
	"regexp"
	"testing"
)

func TestChannelModes_Create(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(testChannelKinds, testUserKinds)
	if modes == nil {
		t.Error("Unexpected nil.")
	}
	if modes.modes == nil {
		t.Error("Unexpected nil.")
	}
	if modes.argModes == nil {
		t.Error("Unexpected nil.")
	}
	if modes.addressModes == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := modes.addresses, 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if modes.ChannelModeKinds == nil {
		t.Error("Unexpected nil.")
	}
	if modes.userModeKinds == nil {
		t.Error("Unexpected nil.")
	}

	var _ moder = NewChannelModes(testChannelKinds, testUserKinds)
}

func TestChannelModes_Apply(t *testing.T) {
	t.Parallel()

	m := NewChannelModes(testChannelKinds, testUserKinds)
	pos, neg := m.Apply("abbcd host1 host2 10 arg")
	if exp, got := len(pos), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("abbcd host1 host2 10 arg"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModes(testChannelKinds, testUserKinds)
	pos, neg = m.Apply("+avbbcdo user1 host1 host2 10 arg user2")
	if exp, got := len(pos), 2; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[0].Mode, 'v'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[0].Arg, "user1"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[1].Mode, 'o'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[1].Arg, "user2"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("abbcd host1 host2 10 arg"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModes(testChannelKinds, testUserKinds)
	pos, neg = m.Apply(" +ab-c 10")
	if exp, got := m.IsSet("a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("b 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModes(testChannelKinds, testUserKinds)
	pos, neg = m.Apply("+oxbvy-ozv user1 ban1 user2 user3 user4")
	if exp, got := len(pos), 2; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := len(neg), 2; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[0].Mode, 'o'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[0].Arg, "user1"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[1].Mode, 'v'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := pos[1].Arg, "user2"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := neg[0].Mode, 'o'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := neg[0].Arg, "user3"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := neg[1].Mode, 'v'; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := neg[1].Arg, "user4"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	pos, neg = m.Apply("+o")
	if exp, got := len(pos), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := len(neg), 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModes(testChannelKinds, testUserKinds)
	m.Apply("b 10")
	if exp, got := m.IsSet("b 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.Apply("-b 10 ")
	if exp, got := m.IsSet("b 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModes(testChannelKinds, testUserKinds)
	m.Apply("x-y+z")
	if exp, got := m.IsSet("x"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("y"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("z"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	m = NewChannelModes(testChannelKinds, testUserKinds)
	m.Apply("+cdb 10")
	if exp, got := m.IsSet("c"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("b"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	m.Apply("-c 10")
	if exp, got := m.IsSet("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("b"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModes_ApplyDiff(t *testing.T) {
	t.Parallel()

	m := NewChannelModes(testChannelKinds, testUserKinds)
	m.Set("abbcd host1 host2 10 arg")

	d := NewModeDiff(testChannelKinds, testUserKinds)
	d.Apply("-a-b+z-d+bc host1 host3 15")
	m.ApplyDiff(d)
	if exp, got := m.IsSet("b host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("b host3"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("z"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("c 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("c 15"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := m.IsSet("a"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModes_IsSet(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(testChannelKinds, testUserKinds)
	modes.modes['a'] = true
	modes.addressModes['b'] = []string{"*!*@host1", "*!*@host2"}
	modes.argModes['c'] = "10"
	modes.argModes['d'] = "arg"

	check(modes, t)
}

func TestChannelModes_GetArgs(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("bbc host1 host2 10")
	if exp, got := modes.GetArg('c'), "10"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	addresses := modes.GetAddresses('b')
	if exp, got := addresses[0], "host1"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := addresses[1], "host2"; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	if exp, got := modes.GetArg('d'), ""; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if got := modes.GetAddresses('z'); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
}

func check(modes *ChannelModes, t *testing.T) {
	// Blanks
	if exp, got := modes.IsSet(), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet(""), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet(" "), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	// Spacing
	if exp, got := modes.IsSet("a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("a "), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet(" a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet(" a "), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	// Associative
	if exp, got := modes.IsSet("a", "b"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b", "z"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("z"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("a", "z"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("z", "a"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	// Simple Args
	if exp, got := modes.IsSet("b *!*@host1"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b *!*@host2"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b *!*@host3"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 15"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d arg"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d noarg"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("z 20"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c *!*@host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	// Multiple args
	if exp, got := modes.IsSet("a", "c 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 10", "a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("a", "c 20"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 10", "b *!*@host1"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 15", "b *!*@not"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 10", "b *!*@host1"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c 15", "b *!*@host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c *!*@host1", "b 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	// Combined Args
	if exp, got := modes.IsSet("ac 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("ca 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("a", "c 20"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("cb 10 *!*@host1"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("cb 15 *!*@not"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("cb 10 *!*@host1"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("cb 15 *!*@host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("cb *!*@host 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	// Missing Args
	if exp, got := modes.IsSet("abc"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("acb 10"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("abc 10"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("abc *!*@host1"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("acb *!*@host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModes_Set(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(testChannelKinds, testUserKinds)

	modes.Set()
	modes.Set("")
	modes.Set(" ")
	modes.Set("a")
	modes.Set("b *!*@host1")
	modes.Set("b *!*@host2")
	modes.Set("c 10")
	modes.Set("d arg")
	check(modes, t)

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	check(modes, t)

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("abbcd *!*@host1 *!*@host2 10 arg")
	check(modes, t)

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("cb")
	if exp, got := modes.IsSet("b"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModes_AddressTracking(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(NewChannelModeKinds("yz", "", "", ""),
		testUserKinds)
	if exp, got := modes.addresses, 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	modes.Set("y *!*@host1", "y *!*@host2", "z *!*@host3")
	if exp, got := modes.addresses, 3; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	modes.Unset("y *!*@host1")
	if exp, got := modes.addresses, 2; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	modes.Unset("yz *!*@host2 *!*@host3")
	if exp, got := modes.addresses, 0; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("yz"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModes_Unset(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset()
	modes.Unset("")
	modes.Unset("ab")
	if exp, got := modes.IsSet("a"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("a", "b", "d")
	if exp, got := modes.IsSet("a"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("b *!*@host1", "c 10")
	if exp, got := modes.IsSet("a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b *!*@host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b *!*@host2"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("dbb *!*@host1 *!*@host2")
	modes.Unset("c")
	if exp, got := modes.IsSet("a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("dbc *!*@host1 10")
	if exp, got := modes.IsSet("a"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b *!*@host1"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b *!*@host2"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("bad *!*@not.host1")
	if exp, got := modes.IsSet("a"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("a", "b *!*@not.host1")
	if exp, got := modes.IsSet("a"), false; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("b"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("c"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
	if exp, got := modes.IsSet("d"), true; exp != got {
		t.Error("Expected: %v, got: %v", exp, got)
	}
}

func TestChannelModes_String(t *testing.T) {
	t.Parallel()

	modes := NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("a", "b host1", "b host2", "c 10", "d arg")
	str := modes.String()
	matched, err := regexp.MatchString(
		`^[abbcd]{5}( arg| 10){2}( host1| host2){2}$`, str)
	if err != nil {
		t.Error("Regexp failed to compile:", err)
	}
	if !matched {
		t.Errorf("Expected: %q to match the pattern.", str)
	}

	modes = NewChannelModes(testChannelKinds, testUserKinds)
	modes.Set("xyz")
	str = modes.String()
	matched, err = regexp.MatchString(`^[xyz]{3}$`, str)
	if err != nil {
		t.Error("Regexp failed to compile:", err)
	}
	if !matched {
		t.Errorf("Expected: %q to match the pattern.", str)
	}
}
