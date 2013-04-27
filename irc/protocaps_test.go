package irc

import (
	. "launchpad.net/gocheck"
)

func (s *s) TestSetServerCaps(c *C) {
	capabilites := []string{"#&~", "(ov)@+", "@+", "b,k,l,imnpstrDdRcC"}
	caps := &ProtoCaps{
		capabilites[0],
		capabilites[1],
		capabilites[2],
		capabilites[3],
	}
	c.Assert(caps.Chantypes, Equals, capabilites[0])
	c.Assert(caps.Prefix, Equals, capabilites[1])
	c.Assert(caps.Statusmsg, Equals, capabilites[2])
	c.Assert(caps.Chanmodes, Equals, capabilites[3])
}
