package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestUserModes_Create(c *C) {
	u, err := CreateUserModes("")
	c.Check(u, IsNil)
	c.Check(err, NotNil)
	u, err = CreateUserModes("a")
	c.Check(u, IsNil)
	c.Check(err, NotNil)
	u, err = CreateUserModes("(a")
	c.Check(u, IsNil)
	c.Check(err, NotNil)

	u, err = CreateUserModes("(ov)@+")
	c.Check(u, NotNil)
	c.Check(err, IsNil)
	c.Check(u.modes[0], Equals, [2]rune{'o', '@'})
	c.Check(u.modes[1], Equals, [2]rune{'v', '+'})
}

func (s *s) TestUserModes_GetSymbol(c *C) {
	u, err := CreateUserModes("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetSymbol('o'), Equals, '@')
	c.Check(u.GetSymbol(' '), Equals, rune(0))
}

func (s *s) TestUserModes_GetMode(c *C) {
	u, err := CreateUserModes("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetMode('@'), Equals, 'o')
	c.Check(u.GetMode(' '), Equals, rune(0))
}

func (s *s) TestUserModes_Update(c *C) {
	u, err := CreateUserModes("(ov)@+")
	c.Check(err, IsNil)
	c.Check(u.GetModeBit('o'), Not(Equals), 0)
	err = u.UpdateModes("(v)+")
	c.Check(err, IsNil)
	c.Check(u.GetModeBit('o'), Equals, 0)

	u, err = CreateUserModes("(ov)@+")
	err = u.UpdateModes("")
	c.Check(err, NotNil)
}
