package data

import (
	. "launchpad.net/gocheck"
)

var testKinds = CreateModeKinds("b", "c", "d")

func (s *s) TestModeset_Create(c *C) {
	modes := CreateModeset(testKinds)
	c.Check(modes, NotNil)
	c.Check(modes.modes, NotNil)
	c.Check(modes.argModes, NotNil)
	c.Check(modes.addressModes, NotNil)
	c.Check(modes.addresses, Equals, 0)
	c.Check(&modes.kinds, Equals, &testKinds.kinds)

	var _ moder = CreateModeset(testKinds)
}

func (s *s) TestModeset_Apply(c *C) {
	m := CreateModeset(testKinds)
	m.Apply("abbcd host1 host2 10 arg")
	c.Check(m.IsSet("abbcd host1 host2 10 arg"), Equals, true)

	m = CreateModeset(testKinds)
	m.Apply("+abbcd host1 host2 10 arg")
	c.Check(m.IsSet("abbcd host1 host2 10 arg"), Equals, true)

	m = CreateModeset(testKinds)
	m.Apply(" +ab-c 10")
	c.Check(m.IsSet("a"), Equals, true)
	c.Check(m.IsSet("b 10"), Equals, true)
	c.Check(m.IsSet("c"), Equals, false)

	m = CreateModeset(testKinds)
	m.Apply("b 10")
	c.Check(m.IsSet("b 10"), Equals, true)
	m.Apply("-b 10 ")
	c.Check(m.IsSet("b 10"), Equals, false)

	m = CreateModeset(testKinds)
	m.Apply("x-y+z")
	c.Check(m.IsSet("x"), Equals, true)
	c.Check(m.IsSet("y"), Equals, false)
	c.Check(m.IsSet("z"), Equals, true)

	m = CreateModeset(testKinds)
	m.Apply("+cdb 10")
	c.Check(m.IsSet("c"), Equals, true)
	c.Check(m.IsSet("d"), Equals, false)
	c.Check(m.IsSet("b"), Equals, false)
	m.Apply("-c 10")
	c.Check(m.IsSet("c"), Equals, false)
	c.Check(m.IsSet("d"), Equals, false)
	c.Check(m.IsSet("b"), Equals, false)
}

func (s *s) TestModeset_ApplyDiff(c *C) {
	m := CreateModeset(testKinds)
	m.Set("abbcd host1 host2 10 arg")

	d := CreateModeDiff(testKinds)
	d.Apply("-a-b+z-d+bc host1 host3 15")
	m.ApplyDiff(d)
	c.Check(m.IsSet("b host1"), Equals, false)
	c.Check(m.IsSet("b host3"), Equals, true)
	c.Check(m.IsSet("z"), Equals, true)
	c.Check(m.IsSet("c 10"), Equals, false)
	c.Check(m.IsSet("c 15"), Equals, true)
	c.Check(m.IsSet("d"), Equals, false)
	c.Check(m.IsSet("a"), Equals, false)
}

func (s *s) TestModeset_IsSet(c *C) {
	modes := CreateModeset(testKinds)
	modes.modes['a'] = true
	modes.addressModes['b'] = []string{"*!*@host1", "*!*@host2"}
	modes.argModes['c'] = "10"
	modes.argModes['d'] = "arg"

	check(modes, c)
}

func (s *s) TestModeset_GetArgs(c *C) {
	modes := CreateModeset(testKinds)
	modes.Set("bbc host1 host2 10")
	c.Check(modes.GetArg('c'), Equals, "10")
	addresses := modes.GetAddresses('b')
	c.Check(addresses[0], Equals, "host1")
	c.Check(addresses[1], Equals, "host2")

	c.Check(modes.GetArg('d'), Equals, "")
	c.Check(modes.GetAddresses('z'), IsNil)
}

func check(modes *Modeset, c *C) {
	// Blanks
	c.Check(modes.IsSet(), Equals, false)
	c.Check(modes.IsSet(""), Equals, false)
	c.Check(modes.IsSet(" "), Equals, false)

	// Spacing
	c.Check(modes.IsSet("a"), Equals, true)
	c.Check(modes.IsSet("a "), Equals, true)
	c.Check(modes.IsSet(" a"), Equals, true)
	c.Check(modes.IsSet(" a "), Equals, true)

	// Associative
	c.Check(modes.IsSet("a", "b"), Equals, true)
	c.Check(modes.IsSet("b", "z"), Equals, false)
	c.Check(modes.IsSet("z"), Equals, false)
	c.Check(modes.IsSet("a", "z"), Equals, false)
	c.Check(modes.IsSet("z", "a"), Equals, false)

	// Simple Args
	c.Check(modes.IsSet("b *!*@host1"), Equals, true)
	c.Check(modes.IsSet("b *!*@host2"), Equals, true)
	c.Check(modes.IsSet("b *!*@host3"), Equals, false)
	c.Check(modes.IsSet("c 10"), Equals, true)
	c.Check(modes.IsSet("c 15"), Equals, false)
	c.Check(modes.IsSet("d arg"), Equals, true)
	c.Check(modes.IsSet("d noarg"), Equals, false)
	c.Check(modes.IsSet("z 20"), Equals, false)
	c.Check(modes.IsSet("c *!*@host1"), Equals, false)
	c.Check(modes.IsSet("b 10"), Equals, false)

	// Multiple args
	c.Check(modes.IsSet("a", "c 10"), Equals, true)
	c.Check(modes.IsSet("c 10", "a"), Equals, true)
	c.Check(modes.IsSet("a", "c 20"), Equals, false)
	c.Check(modes.IsSet("c 10", "b *!*@host1"), Equals, true)
	c.Check(modes.IsSet("c 15", "b *!*@not"), Equals, false)
	c.Check(modes.IsSet("c 10", "b *!*@host1"), Equals, true)
	c.Check(modes.IsSet("c 15", "b *!*@host1"), Equals, false)
	c.Check(modes.IsSet("c *!*@host1", "b 10"), Equals, false)

	// Combined Args
	c.Check(modes.IsSet("ac 10"), Equals, true)
	c.Check(modes.IsSet("ca 10"), Equals, true)
	c.Check(modes.IsSet("a", "c 20"), Equals, false)
	c.Check(modes.IsSet("cb 10 *!*@host1"), Equals, true)
	c.Check(modes.IsSet("cb 15 *!*@not"), Equals, false)
	c.Check(modes.IsSet("cb 10 *!*@host1"), Equals, true)
	c.Check(modes.IsSet("cb 15 *!*@host1"), Equals, false)
	c.Check(modes.IsSet("cb *!*@host 10"), Equals, false)

	// Missing Args
	c.Check(modes.IsSet("abc"), Equals, true)
	c.Check(modes.IsSet("acb 10"), Equals, true)
	c.Check(modes.IsSet("abc 10"), Equals, false)
	c.Check(modes.IsSet("abc *!*@host1"), Equals, true)
	c.Check(modes.IsSet("acb *!*@host1"), Equals, false)
}

func (s *s) TestModeset_Set(c *C) {
	modes := CreateModeset(testKinds)

	modes.Set()
	modes.Set("")
	modes.Set(" ")
	modes.Set("a")
	modes.Set("b *!*@host1")
	modes.Set("b *!*@host2")
	modes.Set("c 10")
	modes.Set("d arg")
	check(modes, c)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	check(modes, c)

	modes = CreateModeset(testKinds)
	modes.Set("abbcd *!*@host1 *!*@host2 10 arg")
	check(modes, c)

	modes = CreateModeset(testKinds)
	modes.Set("cb")
	c.Check(modes.IsSet("b"), Equals, false)
	c.Check(modes.IsSet("c"), Equals, false)
}

func (s *s) TestModeset_AddressTracking(c *C) {
	modes := CreateModeset(CreateModeKinds("yz", "", ""))
	c.Check(modes.addresses, Equals, 0)
	modes.Set("y *!*@host1", "y *!*@host2", "z *!*@host3")
	c.Check(modes.addresses, Equals, 3)
	modes.Unset("y *!*@host1")
	c.Check(modes.addresses, Equals, 2)
	modes.Unset("yz *!*@host2 *!*@host3")
	c.Check(modes.addresses, Equals, 0)
	c.Check(modes.IsSet("yz"), Equals, false)
}

func (s *s) TestModeset_Unset(c *C) {
	modes := CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset()
	modes.Unset("")
	modes.Unset("ab")
	c.Check(modes.IsSet("a"), Equals, false)
	c.Check(modes.IsSet("b"), Equals, true)
	c.Check(modes.IsSet("c"), Equals, true)
	c.Check(modes.IsSet("d"), Equals, true)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("a", "b", "d")
	c.Check(modes.IsSet("a"), Equals, false)
	c.Check(modes.IsSet("b"), Equals, true)
	c.Check(modes.IsSet("c"), Equals, true)
	c.Check(modes.IsSet("d"), Equals, false)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("b *!*@host1", "c 10")
	c.Check(modes.IsSet("a"), Equals, true)
	c.Check(modes.IsSet("b *!*@host1"), Equals, false)
	c.Check(modes.IsSet("b *!*@host2"), Equals, true)
	c.Check(modes.IsSet("c"), Equals, false)
	c.Check(modes.IsSet("d"), Equals, true)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("dbb *!*@host1 *!*@host2")
	modes.Unset("c")
	c.Check(modes.IsSet("a"), Equals, true)
	c.Check(modes.IsSet("b"), Equals, false)
	c.Check(modes.IsSet("c"), Equals, true)
	c.Check(modes.IsSet("d"), Equals, false)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("dbc *!*@host1 10")
	c.Check(modes.IsSet("a"), Equals, true)
	c.Check(modes.IsSet("b *!*@host1"), Equals, false)
	c.Check(modes.IsSet("b *!*@host2"), Equals, true)
	c.Check(modes.IsSet("c"), Equals, false)
	c.Check(modes.IsSet("d"), Equals, false)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("bad *!*@not.host1")
	c.Check(modes.IsSet("a"), Equals, false)
	c.Check(modes.IsSet("b"), Equals, true)
	c.Check(modes.IsSet("c"), Equals, true)
	c.Check(modes.IsSet("d"), Equals, false)

	modes = CreateModeset(testKinds)
	modes.Set("a", "b *!*@host1", "b *!*@host2", "c 10", "d arg")
	modes.Unset("a", "b *!*@not.host1")
	c.Check(modes.IsSet("a"), Equals, false)
	c.Check(modes.IsSet("b"), Equals, true)
	c.Check(modes.IsSet("c"), Equals, true)
	c.Check(modes.IsSet("d"), Equals, true)
}

func (s *s) TestModeset_String(c *C) {
	modes := CreateModeset(testKinds)
	modes.Set("a", "b host1", "b host2", "c 10", "d arg")
	str := modes.String()
	c.Check(str, Matches, `^[abbcd]{5}( arg| 10){2}( host1| host2){2}$`)

	modes = CreateModeset(testKinds)
	modes.Set("xyz")
	str = modes.String()
	c.Check(str, Matches, `^xyz$`)
}
