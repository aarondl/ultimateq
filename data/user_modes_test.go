package data

import (
	. "launchpad.net/gocheck"
)

var testUserModes, _ = CreateUserModeKinds("(ov)@+")

func (s *s) TestUserModes(c *C) {
	m := CreateUserModes(testUserModes)
	c.Check(m.HasMode('o'), Equals, false)
	c.Check(m.HasMode('v'), Equals, false)

	m.SetMode('o')
	c.Check(m.HasMode('o'), Equals, true)
	c.Check(m.HasMode('v'), Equals, false)
	m.SetMode('v')
	c.Check(m.HasMode('o'), Equals, true)
	c.Check(m.HasMode('v'), Equals, true)

	m.UnsetMode('o')
	c.Check(m.HasMode('o'), Equals, false)
	c.Check(m.HasMode('v'), Equals, true)
	m.UnsetMode('v')
	c.Check(m.HasMode('o'), Equals, false)
	c.Check(m.HasMode('v'), Equals, false)
}

func (s *s) TestUserModes_String(c *C) {
	m := CreateUserModes(testUserModes)
	c.Check(m.String(), Equals, "")
	c.Check(m.StringSymbols(), Equals, "")
	m.SetMode('o')
	c.Check(m.String(), Equals, "o")
	c.Check(m.StringSymbols(), Equals, "@")
	m.SetMode('v')
	c.Check(m.String(), Equals, "ov")
	c.Check(m.StringSymbols(), Equals, "@+")
	m.UnsetMode('o')
	c.Check(m.String(), Equals, "v")
	c.Check(m.StringSymbols(), Equals, "+")
}
