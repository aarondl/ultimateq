package data

import (
	. "launchpad.net/gocheck"
)

var testUserModes, _ = CreateUserModes("(ov)@+")

func (s *s) TestChannelUser_Create(c *C) {
	ch := CreateChannel("", testKinds)
	user := CreateUser("nick")

	cu := CreateChannelUser(
		ch,
		user,
		testUserModes,
	)

	c.Check(cu, NotNil)
	c.Check(cu.Channel, Equals, ch)
	c.Check(cu.User, Equals, user)
	c.Check(cu.UserModes, Equals, testUserModes)
}

func (s *s) TestChannelUser_Modes(c *C) {
	cu := CreateChannelUser(nil, nil, testUserModes)
	c.Check(cu.HasMode('o'), Equals, false)
	c.Check(cu.HasMode('v'), Equals, false)

	cu.SetMode('o')
	c.Check(cu.HasMode('o'), Equals, true)
	c.Check(cu.HasMode('v'), Equals, false)
	cu.SetMode('v')
	c.Check(cu.HasMode('o'), Equals, true)
	c.Check(cu.HasMode('v'), Equals, true)

	cu.UnsetMode('v')
	c.Check(cu.HasMode('o'), Equals, true)
	c.Check(cu.HasMode('v'), Equals, false)
	cu.UnsetMode('o')
	c.Check(cu.HasMode('o'), Equals, false)
	c.Check(cu.HasMode('v'), Equals, false)
}
