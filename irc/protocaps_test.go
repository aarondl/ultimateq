package irc

import . "launchpad.net/gocheck"

func (s *s) TestSetServerCaps(c *C) {
	capabilites := []string{"(ov)@+", "@+", "b,k,l,imnpstrDdRcC", "#&~"}
	caps := &ProtoCaps{
		capabilites[0],
		capabilites[1],
		capabilites[2],
		"",
		nil,
	}
	c.Assert(caps.Prefix, Equals, capabilites[0])
	c.Assert(caps.Statusmsg, Equals, capabilites[1])
	c.Assert(caps.Chanmodes, Equals, capabilites[2])

	caps.SetChantypes(capabilites[3])
	c.Assert(caps.chantypes, Equals, capabilites[3])
	c.Assert(caps.chantypesRegex, NotNil)
	c.Assert(caps.chantypesRegex.MatchString("#channel"), Equals, true)
}

func (s *s) TestSetChanTypes_Security(c *C) {
	p := &ProtoCaps{}
	err := p.SetChantypes(`*+?[]()-^`)
	c.Assert(err, IsNil)
	c.Assert(p.chantypesRegex.MatchString(")hello"), Equals, true)
}
