package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestUserModes_Create(c *C) {
	u, err := CreateUserModes("")
	c.Assert(u, IsNil)
	c.Assert(err, NotNil)
	u, err = CreateUserModes("a")
	c.Assert(u, IsNil)
	c.Assert(err, NotNil)
	u, err = CreateUserModes("(a")
	c.Assert(u, IsNil)
	c.Assert(err, NotNil)

	u, err = CreateUserModes("(ov)@+")
	c.Assert(u, NotNil)
	c.Assert(err, IsNil)
	c.Assert(u.modes[0], Equals, [2]rune{'o', '@'})
	c.Assert(u.modes[1], Equals, [2]rune{'v', '+'})
}

func (s *s) TestUserModes_GetSymbol(c *C) {
	u, err := CreateUserModes("(ov)@+")
	c.Assert(err, IsNil)
	c.Assert(u.GetSymbol('o'), Equals, '@')
	c.Assert(u.GetSymbol(' '), Equals, rune(0))
}

func (s *s) TestUserModes_GetMode(c *C) {
	u, err := CreateUserModes("(ov)@+")
	c.Assert(err, IsNil)
	c.Assert(u.GetMode('@'), Equals, 'o')
	c.Assert(u.GetMode(' '), Equals, rune(0))
}

func (s *s) TestUserModes_Update(c *C) {
	u, err := CreateUserModes("(ov)@+")
	c.Assert(err, IsNil)
	c.Assert(u.GetModeBit('o'), Not(Equals), 0)
	err = u.UpdateModes("(v)+")
	c.Assert(err, IsNil)
	c.Assert(u.GetModeBit('o'), Equals, 0)

	u, err = CreateUserModes("(ov)@+")
	err = u.UpdateModes("")
	c.Assert(err, NotNil)
}
