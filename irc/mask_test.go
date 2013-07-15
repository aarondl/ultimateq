package irc

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestMask(c *C) {
	var mask Mask = "nick!user@host"

	c.Check(mask.GetNick(), Equals, "nick")
	c.Check(mask.GetUsername(), Equals, "user")
	c.Check(mask.GetHost(), Equals, "host")
	c.Check(mask.GetFullhost(), Equals, string(mask))

	mask = "nick@user!host"
	c.Check(mask.GetNick(), Equals, "nick")
	c.Check(mask.GetUsername(), Equals, "")
	c.Check(mask.GetHost(), Equals, "")
	c.Check(mask.GetFullhost(), Equals, string(mask))

	mask = "nick"
	c.Check(mask.GetNick(), Equals, "nick")
	c.Check(mask.GetUsername(), Equals, "")
	c.Check(mask.GetHost(), Equals, "")
	c.Check(mask.GetFullhost(), Equals, string(mask))
}

func (s *s) TestMask_SplitHost(c *C) {
	var nick, user, host string

	nick, user, host = Mask("nick!user@host").Split()
	c.Check(nick, Equals, "nick")
	c.Check(user, Equals, "user")
	c.Check(host, Equals, "host")

	nick, user, host = WildMask("ni ck!user@host").Split()
	c.Check(nick, Equals, "")
	c.Check(user, Equals, "")
	c.Check(host, Equals, "")
}

func (s *s) TestMask_IsValid(c *C) {
	var isValid bool
	isValid = Mask("").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("!@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("nick").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("nick!").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("nick@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("nick@host!user").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("nick!user@host").IsValid()
	c.Check(isValid, Equals, true)
}

func (s *s) TestWildMask_Split(c *C) {
	var nick, user, host string
	nick, user, host = WildMask("n?i*ck!u*ser@h*o?st").Split()
	c.Check(nick, Equals, "n?i*ck")
	c.Check(user, Equals, "u*ser")
	c.Check(host, Equals, "h*o?st")

	nick, user, host = WildMask("n?i* ck!u*ser@h*o?st").Split()
	c.Check(nick, Equals, "")
	c.Check(user, Equals, "")
	c.Check(host, Equals, "")
}

func (s *s) TestWildMask_IsValid(c *C) {
	var isValid bool
	isValid = WildMask("").IsValid()
	c.Check(isValid, Equals, false)

	isValid = WildMask("!@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = WildMask("n?i*ck").IsValid()
	c.Check(isValid, Equals, false)

	isValid = WildMask("n?i*ck!").IsValid()
	c.Check(isValid, Equals, false)

	isValid = WildMask("n?i*ck@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = WildMask("n*i?ck@h*o?st!u*ser").IsValid()
	c.Check(isValid, Equals, false)

	isValid = WildMask("n?i*ck!u*ser@h*o?st").IsValid()
	c.Check(isValid, Equals, true)
}

func (s *s) TestWildMask_Match(c *C) {
	var wmask WildMask
	var mask Mask
	c.Check(wmask.Match(mask), Equals, true)

	c.Check(WildMask("nick!*@*").Match("nick!@"), Equals, true)

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
		if !mask.Match(positiveMasks[i]) {
			c.Errorf("Expected: %v to match %v", mask, positiveMasks[i])
		}
	}

	negativeMasks := []WildMask{
		``, `?nq******c?!*@*`, `nick2!*@*`, `*!*@hostfail`, `*!*@failhost`,
	}

	for i := 0; i < len(negativeMasks); i++ {
		if negativeMasks[i].Match(mask) {
			c.Errorf("Expected: %v not to match %v", negativeMasks[i], mask)
		}
		if mask.Match(negativeMasks[i]) {
			c.Errorf("Expected: %v to match %v", mask, negativeMasks[i])
		}
	}

}
