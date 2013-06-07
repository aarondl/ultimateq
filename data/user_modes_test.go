package data

import (
	. "launchpad.net/gocheck"
)

var testUserModes, _ = CreateUserModesMeta("(ov)@+")

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

func (s *s) TestUserModesMeta_Create(c *C) {
	u, err := CreateUserModesMeta("")
	c.Check(u, IsNil)
	c.Check(err, NotNil)
	u, err = CreateUserModesMeta("a")
	c.Check(u, IsNil)
	c.Check(err, NotNil)
	u, err = CreateUserModesMeta("(a")
	c.Check(u, IsNil)
	c.Check(err, NotNil)

	u, err = CreateUserModesMeta("(ov)@+")
	c.Check(u, NotNil)
	c.Check(err, IsNil)
	c.Check(u.modeInfo[0], Equals, [2]rune{'o', '@'})
	c.Check(u.modeInfo[1], Equals, [2]rune{'v', '+'})
}

func (s *s) TestUserModesMeta_GetSymbol(c *C) {
	u, err := CreateUserModesMeta("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetSymbol('o'), Equals, '@')
	c.Check(u.GetSymbol(' '), Equals, rune(0))
}

func (s *s) TestUserModesMeta_GetMode(c *C) {
	u, err := CreateUserModesMeta("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetMode('@'), Equals, 'o')
	c.Check(u.GetMode(' '), Equals, rune(0))
}

func (s *s) TestUserModesMeta_Update(c *C) {
	u, err := CreateUserModesMeta("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetModeBit('o'), Not(Equals), 0)
	err = u.UpdateModes("(v)+")
	c.Check(err, IsNil)
	c.Check(u.GetModeBit('o'), Equals, 0)

	u, err = CreateUserModesMeta("(ov)@+")
	err = u.UpdateModes("")
	c.Check(err, NotNil)
}
