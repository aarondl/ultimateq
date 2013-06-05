package data

import (
	. "launchpad.net/gocheck"
)

var testUserModes = CreateUserModes("(ov)@+")

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
