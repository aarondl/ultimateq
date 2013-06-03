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

func (s *s) TestWildMask_Match(c *C) {
	var wmask WildMask = ""
	var mask Mask = ""
	c.Assert(wmask.Match(mask), Equals, true)

	c.Assert(WildMask("nick!*@*").Match("nick!@"), Equals, true)

	mask = "nick!user@host"

	positiveMasks := []WildMask{
		// Default
		`nick!user@host`,
		// *'s
		`*`, `*!*@*`, `**!**@**`, `*@host`, `**@host`,
		`nick!*`, `nick!**`, `*nick!user@host`, `**nick!user@host`,
		`nick!user@host*`, `nick!user@host**`,
		// ?'s
		`ni?k!us?r@ho?st`, `ni??k!us??r@ho??st`, `????!????@????`,
		`?ick!user@host`, `??ick!user@host`, `?nick!user@host`,
		`??nick!user@host`, `nick!user@hos?`, `nick!user@hos??`,
		`nick!user@host?`, `nick!user@host??`,
		// Combination
		`?*nick!user@host`, `*?nick!user@host`, `??**nick!user@host`,
		`**??nick!user@host`,
		`nick!user@host?*`, `nick!user@host*?`, `nick!user@host??**`,
		`nick!user@host**??`, `nick!u?*?ser@host`, `nick!u?*?ser@host`,
	}

	for i := 0; i < len(positiveMasks); i++ {
		if !positiveMasks[i].Match(mask) {
			c.Errorf("Expected: %v to match %v", positiveMasks[i], mask)
		}
	}

	negativeMasks := []WildMask{
		``, `?nq******c?!*@*`, `nick2!*@*`, `*!*@hostfail`, `*!*@failhost`,
	}

	for i := 0; i < len(negativeMasks); i++ {
		if negativeMasks[i].Match(mask) {
			c.Errorf("Expected: %v not to match %v", negativeMasks[i], mask)
		}
	}

}
