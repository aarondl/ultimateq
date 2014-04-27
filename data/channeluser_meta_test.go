package data

import (
	. "gopkg.in/check.v1"
)

var modes = new(int)

func (s *s) TestChannelUser(c *C) {
	user := NewUser("nick")
	modes := NewUserModes(testUserKinds)

	cu := NewChannelUser(
		user,
		modes,
	)

	c.Check(cu, NotNil)
	c.Check(cu.User, Equals, user)
	c.Check(cu.UserModes, Equals, modes)
}

func (s *s) TestUserChannel(c *C) {
	ch := NewChannel("", testChannelKinds, testUserKinds)
	modes := NewUserModes(testUserKinds)

	uc := NewUserChannel(
		ch,
		modes,
	)

	c.Check(uc, NotNil)
	c.Check(uc.Channel, Equals, ch)
	c.Check(uc.UserModes, Equals, modes)
}
