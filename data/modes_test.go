package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestModeDiff_Create(c *C) {
	diff := CreateModeDiff()
	c.Assert(diff, NotNil)
	c.Assert(diff.pos, NotNil)
	c.Assert(diff.neg, NotNil)
	c.Assert(diff.pos.modes, NotNil)
	c.Assert(diff.neg.modes, NotNil)

	var _ moder = CreateModeDiff()
}

func (s *s) TestModeDiff_Apply(c *C) {
	d := CreateModeDiffFromModestring("+ab-c 10 ", "c")
	c.Assert(d.IsSet("ab"), Equals, true)
	c.Assert(d.IsSet("a 10"), Equals, false)
	c.Assert(d.IsSet("b 10"), Equals, false)
	c.Assert(d.IsSet("c"), Equals, false)
	c.Assert(d.IsUnset("c"), Equals, true)
	c.Assert(d.IsUnset("c 10"), Equals, true)

	d.Apply(" +ab-c 10", "b")
	c.Assert(d.IsSet("a"), Equals, true)
	c.Assert(d.IsSet("b 10"), Equals, true)
	c.Assert(d.IsUnset("a"), Equals, false)
	c.Assert(d.IsUnset("b 10"), Equals, false)
	c.Assert(d.IsSet("c"), Equals, false)
	c.Assert(d.IsUnset("c"), Equals, true)

	d.Apply("-b 10", "b")
	c.Assert(d.IsSet("a"), Equals, true)
	c.Assert(d.IsSet("b"), Equals, false)
	c.Assert(d.IsUnset("b"), Equals, true)

	d.Apply("x-y+z", "")
	c.Assert(d.IsSet("x"), Equals, true)
	c.Assert(d.IsUnset("y"), Equals, true)
	c.Assert(d.IsSet("z"), Equals, true)
	c.Assert(d.IsUnset("x"), Equals, false)
	c.Assert(d.IsSet("y"), Equals, false)
	c.Assert(d.IsUnset("z"), Equals, false)
}

func (s *s) TestModeDiff_String(c *C) {
	diff := CreateModeDiff()
	diff.pos.Set("a", "b 2", "c 1")
	diff.neg.Set("x", "y -2", "z -1")
	str := diff.String()
	c.Assert(str, Matches, `^\+[abc]{3}-[xyz]{3}( 1| 2){2}( -1| -2){2}$`)
}

func (s *s) TestModeset_Create(c *C) {
	modes := CreateModeset()
	c.Assert(modes, NotNil)
	c.Assert(modes.modes, NotNil)

	var _ moder = CreateModeset()
}

func (s *s) TestModeset_Apply(c *C) {
	m := CreateModesetFromModestring("+ab-c 10 ", "c")
	c.Assert(m.IsSet("ab"), Equals, true)
	c.Assert(m.IsSet("a 10"), Equals, false)
	c.Assert(m.IsSet("b 10"), Equals, false)
	c.Assert(m.IsSet("c"), Equals, false)

	m.Apply(" +ab-c 10", "b")
	c.Assert(m.IsSet("a"), Equals, true)
	c.Assert(m.IsSet("b 10"), Equals, true)
	c.Assert(m.IsSet("c"), Equals, false)

	m.Apply("-b 10", "b")
	c.Assert(m.IsSet("a"), Equals, true)
	c.Assert(m.IsSet("b"), Equals, false)

	m.Apply("x-y+z", "")
	c.Assert(m.IsSet("x"), Equals, true)
	c.Assert(m.IsSet("y"), Equals, false)
	c.Assert(m.IsSet("z"), Equals, true)
}

func (s *s) TestModeset_ApplyDiff(c *C) {
	m := CreateModesetFromModestring("+ab-c 10 ", "c")
	c.Assert(m.IsSet("ab"), Equals, true)
	c.Assert(m.IsSet("a 10"), Equals, false)
	c.Assert(m.IsSet("b 10"), Equals, false)
	c.Assert(m.IsSet("c"), Equals, false)

	m.ApplyDiff(CreateModeDiffFromModestring(" +ab-c 10", "b"))
	c.Assert(m.IsSet("a"), Equals, true)
	c.Assert(m.IsSet("b 10"), Equals, true)
	c.Assert(m.IsSet("c"), Equals, false)

	m.ApplyDiff(CreateModeDiffFromModestring("-b 10", "b"))
	c.Assert(m.IsSet("a"), Equals, true)
	c.Assert(m.IsSet("b"), Equals, false)

	m.ApplyDiff(CreateModeDiffFromModestring("x-y+z", ""))
	c.Assert(m.IsSet("x"), Equals, true)
	c.Assert(m.IsSet("y"), Equals, false)
	c.Assert(m.IsSet("z"), Equals, true)
}

func (s *s) TestModeset_IsSet(c *C) {
	modes := CreateModeset()
	modes.modes['a'] = ""
	modes.modes['b'] = "*!*@aol.com"
	modes.modes['c'] = "10"

	check(modes, c)
}

func (s *s) TestModeset_Set(c *C) {
	modes := CreateModeset()

	modes.Set()
	modes.Set("")
	modes.Set(" ")
	modes.Set("a")
	modes.Set("b *!*@aol.com")
	modes.Set("c 10")
	check(modes, c)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	check(modes, c)

	modes = CreateModeset()
	modes.Set("abc *!*@aol.com 10")
	check(modes, c)
}

func (s *s) TestModeset_Unset(c *C) {
	modes := CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset()
	modes.Unset("")
	modes.Unset("ab")
	c.Assert(modes.IsSet("a"), Equals, false)
	c.Assert(modes.IsSet("b"), Equals, false)
	c.Assert(modes.IsSet("c"), Equals, true)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("a", "b")
	c.Assert(modes.IsSet("a"), Equals, false)
	c.Assert(modes.IsSet("b"), Equals, false)
	c.Assert(modes.IsSet("c"), Equals, true)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("b *!*@aol.com", "c 10")
	c.Assert(modes.IsSet("a"), Equals, true)
	c.Assert(modes.IsSet("b"), Equals, false)
	c.Assert(modes.IsSet("c"), Equals, false)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("bc *!*@aol.com 10")
	c.Assert(modes.IsSet("a"), Equals, true)
	c.Assert(modes.IsSet("b"), Equals, false)
	c.Assert(modes.IsSet("c"), Equals, false)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("ab *!*@not.aol.com")
	c.Assert(modes.IsSet("a"), Equals, false)
	c.Assert(modes.IsSet("b"), Equals, true)
	c.Assert(modes.IsSet("c"), Equals, true)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("a", "b *!*@not.aol.com")
	c.Assert(modes.IsSet("a"), Equals, false)
	c.Assert(modes.IsSet("b"), Equals, true)
	c.Assert(modes.IsSet("c"), Equals, true)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("ba *!*@not.aol.com")
	c.Assert(modes.IsSet("a"), Equals, true)
	c.Assert(modes.IsSet("b"), Equals, false)
	c.Assert(modes.IsSet("c"), Equals, true)

	modes = CreateModeset()
	modes.Set("a", "b *!*@aol.com", "c 10")
	modes.Unset("b", "a *!*@not.aol.com")
	c.Assert(modes.IsSet("a"), Equals, true)
	c.Assert(modes.IsSet("b"), Equals, false)
	c.Assert(modes.IsSet("c"), Equals, true)
}

func (s *s) TestModeset_String(c *C) {
	modes := CreateModeset()
	modes.Set("a", "b 15", "c 10")
	str := modes.String()
	c.Assert(str, Matches, `^[abc]{3}( 15| 10){2}$`)
}

var check = func(modes *Modeset, c *C) {
	c.Assert(modes.IsSet(), Equals, false)
	c.Assert(modes.IsSet(""), Equals, false)
	c.Assert(modes.IsSet(" "), Equals, false)

	c.Assert(modes.IsSet("a"), Equals, true)
	c.Assert(modes.IsSet("b"), Equals, true)
	c.Assert(modes.IsSet("a "), Equals, true)
	c.Assert(modes.IsSet(" a"), Equals, true)
	c.Assert(modes.IsSet(" a "), Equals, true)
	c.Assert(modes.IsSet("a", "b"), Equals, true)
	c.Assert(modes.IsSet("a", "z"), Equals, false)
	c.Assert(modes.IsSet("z", "a"), Equals, false)

	c.Assert(modes.IsSet("a"), Equals, true)
	c.Assert(modes.IsSet("b"), Equals, true)
	c.Assert(modes.IsSet("a "), Equals, true)
	c.Assert(modes.IsSet(" a"), Equals, true)
	c.Assert(modes.IsSet(" a "), Equals, true)
	c.Assert(modes.IsSet("a", "b"), Equals, true)
	c.Assert(modes.IsSet("a", "z"), Equals, false)
	c.Assert(modes.IsSet("z", "a"), Equals, false)

	c.Assert(modes.IsSet("z"), Equals, false)
	c.Assert(modes.IsSet("b *!*@aol.com"), Equals, true)
	c.Assert(modes.IsSet("b other@mask"), Equals, false)
	c.Assert(modes.IsSet("z 20"), Equals, false)

	c.Assert(modes.IsSet("a", "c 10"), Equals, true)
	c.Assert(modes.IsSet("c 10", "a"), Equals, true)
	c.Assert(modes.IsSet("a", "c 20"), Equals, false)

	c.Assert(modes.IsSet("abc 10"), Equals, true)
	c.Assert(modes.IsSet("acb 10"), Equals, false)
	c.Assert(modes.IsSet("acb 10 *!*@aol.com"), Equals, true)
}
