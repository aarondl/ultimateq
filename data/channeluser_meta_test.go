package data

import (
	. "launchpad.net/gocheck"
)

var modes = new(int)

func (s *s) TestChannelUser(c *C) {
	user := CreateUser("nick")
	modes := CreateUserModes(testUserKinds)

	cu := CreateChannelUser(
		user,
		modes,
	)

	c.Check(cu, NotNil)
	c.Check(cu.User, Equals, user)
	c.Check(cu.UserModes, Equals, modes)
}

func (s *s) TestUserChannel(c *C) {
	ch := CreateChannel("", testChannelKinds, testUserKinds)
	modes := CreateUserModes(testUserKinds)

	uc := CreateUserChannel(
		ch,
		modes,
	)

	c.Check(uc, NotNil)
	c.Check(uc.Channel, Equals, ch)
	c.Check(uc.UserModes, Equals, modes)
}
