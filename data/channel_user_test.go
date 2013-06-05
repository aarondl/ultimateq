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

	c.Assert(cu, NotNil)
	c.Assert(cu.Channel, Equals, ch)
	c.Assert(cu.User, Equals, user)
	c.Assert(cu.UserModes, Equals, testUserModes)
}

func (s *s) TestChannelUser_Modes(c *C) {
	cu := CreateChannelUser(nil, nil, testUserModes)
	c.Assert(cu.HasMode('o'), Equals, false)
	c.Assert(cu.HasMode('v'), Equals, false)

	cu.SetMode('o')
	c.Assert(cu.HasMode('o'), Equals, true)
	c.Assert(cu.HasMode('v'), Equals, false)
	cu.SetMode('v')
	c.Assert(cu.HasMode('o'), Equals, true)
	c.Assert(cu.HasMode('v'), Equals, true)

	cu.UnsetMode('v')
	c.Assert(cu.HasMode('o'), Equals, true)
	c.Assert(cu.HasMode('v'), Equals, false)
	cu.UnsetMode('o')
	c.Assert(cu.HasMode('o'), Equals, false)
	c.Assert(cu.HasMode('v'), Equals, false)
}
