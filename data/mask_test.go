package data

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestMask(c *C) {
	var mask Mask = "nick!user@host"

	c.Assert(mask.GetNick(), Equals, "nick")
	c.Assert(mask.GetUsername(), Equals, "user")
	c.Assert(mask.GetHost(), Equals, "host")
	c.Assert(mask.GetFullhost(), Equals, string(mask))

	mask = "nick@user!host"
	c.Assert(mask.GetNick(), Equals, "nick")
	c.Assert(mask.GetUsername(), Equals, "")
	c.Assert(mask.GetHost(), Equals, "")
	c.Assert(mask.GetFullhost(), Equals, string(mask))

	mask = "nick"
	c.Assert(mask.GetNick(), Equals, "nick")
	c.Assert(mask.GetUsername(), Equals, "")
	c.Assert(mask.GetHost(), Equals, "")
	c.Assert(mask.GetFullhost(), Equals, string(mask))
}

func (s *s) TestMask_SplitHost(c *C) {
	var nick, user, host string

	nick, user, host = Mask("").SplitFullhost()
	c.Assert(nick, Equals, "")
	c.Assert(user, Equals, "")
	c.Assert(host, Equals, "")

	nick, user, host = Mask("nick").SplitFullhost()
	c.Assert(nick, Equals, "nick")
	c.Assert(user, Equals, "")
	c.Assert(host, Equals, "")

	nick, user, host = Mask("nick!").SplitFullhost()
	c.Assert(nick, Equals, "nick")
	c.Assert(user, Equals, "")
	c.Assert(host, Equals, "")

	nick, user, host = Mask("nick@").SplitFullhost()
	c.Assert(nick, Equals, "nick")
	c.Assert(user, Equals, "")
	c.Assert(host, Equals, "")

	nick, user, host = Mask("nick@host!user").SplitFullhost()
	c.Assert(nick, Equals, "nick")
	c.Assert(user, Equals, "")
	c.Assert(host, Equals, "")

	nick, user, host = Mask("nick!user@host").SplitFullhost()
	c.Assert(nick, Equals, "nick")
	c.Assert(user, Equals, "user")
	c.Assert(host, Equals, "host")
}
