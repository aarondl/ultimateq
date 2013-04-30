package irc

import . "launchpad.net/gocheck"

func (s *s) TestProtoCaps(c *C) {
	capabilities := []string{"#&~", "(ov)@+", "@+", "b,k,l,imnpstrDdRcC"}
	caps := &ProtoCaps{
		capabilities[0],
		capabilities[1],
		capabilities[2],
		capabilities[3],
	}
	c.Assert(caps.Chantypes, Equals, capabilities[0])
	c.Assert(caps.Prefix, Equals, capabilities[1])
	c.Assert(caps.Statusmsg, Equals, capabilities[2])
	c.Assert(caps.Chanmodes, Equals, capabilities[3])
}
