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

	m.UnsetMode('v')
	c.Check(m.HasMode('o'), Equals, true)
	c.Check(m.HasMode('v'), Equals, false)
	m.UnsetMode('o')
	c.Check(m.HasMode('o'), Equals, false)
	c.Check(m.HasMode('v'), Equals, false)
}
