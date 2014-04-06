package irc

import (
	. "gopkg.in/check.v1"
)

func (s *s) TestHost(c *C) {
	var host Host = "nick!user@host"

	c.Check(host.Nick(), Equals, "nick")
	c.Check(host.Username(), Equals, "user")
	c.Check(host.Hostname(), Equals, "host")
	c.Check(host.String(), Equals, string(host))

	host = "nick@user!host"
	c.Check(host.Nick(), Equals, "nick")
	c.Check(host.Username(), Equals, "")
	c.Check(host.Hostname(), Equals, "")
	c.Check(host.String(), Equals, string(host))

	host = "nick"
	c.Check(host.Nick(), Equals, "nick")
	c.Check(host.Username(), Equals, "")
	c.Check(host.Hostname(), Equals, "")
	c.Check(host.String(), Equals, string(host))
}

func (s *s) TestHost_SplitHost(c *C) {
	var nick, user, hostname string

	nick, user, hostname = Host("nick!user@host").Split()
	c.Check(nick, Equals, "nick")
	c.Check(user, Equals, "user")
	c.Check(hostname, Equals, "host")

	nick, user, hostname = Host("ni ck!user@host").Split()
	c.Check(nick, Equals, "")
	c.Check(user, Equals, "")
	c.Check(hostname, Equals, "")
}

func (s *s) TestHost_IsValid(c *C) {
	var isValid bool
	isValid = Host("").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Host("!@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Host("nick").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Host("nick!").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Host("nick@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Host("nick@host!user").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Host("nick!user@host").IsValid()
	c.Check(isValid, Equals, true)
}

func (s *s) TestMask_Split(c *C) {
	var nick, user, host string
	nick, user, host = Mask("n?i*ck!u*ser@h*o?st").Split()
	c.Check(nick, Equals, "n?i*ck")
	c.Check(user, Equals, "u*ser")
	c.Check(host, Equals, "h*o?st")

	nick, user, host = Mask("n?i* ck!u*ser@h*o?st").Split()
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

	isValid = Mask("n?i*ck").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("n?i*ck!").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("n?i*ck@").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("n*i?ck@h*o?st!u*ser").IsValid()
	c.Check(isValid, Equals, false)

	isValid = Mask("n?i*ck!u*ser@h*o?st").IsValid()
	c.Check(isValid, Equals, true)
}

func (s *s) TestMask_Match(c *C) {
	var mask Mask
	var host Host
	c.Check(mask.Match(host), Equals, true)

	c.Check(Mask("nick!*@*").Match("nick!@"), Equals, true)

	host = "nick!user@host"

	positiveMasks := []Mask{
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
		if !positiveMasks[i].Match(host) {
			c.Errorf("Expected: %v to match %v", positiveMasks[i], host)
		}
		if !host.Match(positiveMasks[i]) {
			c.Errorf("Expected: %v to match %v", host, positiveMasks[i])
		}
	}

	negativeMasks := []Mask{
		``, `?nq******c?!*@*`, `nick2!*@*`, `*!*@hostfail`, `*!*@failhost`,
	}

	for i := 0; i < len(negativeMasks); i++ {
		if negativeMasks[i].Match(host) {
			c.Errorf("Expected: %v not to match %v", negativeMasks[i], host)
		}
		if host.Match(negativeMasks[i]) {
			c.Errorf("Expected: %v to match %v", host, negativeMasks[i])
		}
	}

}
