package proto

import (
	. "launchpad.net/gocheck"
)

// Tests that capabilites can be get and set.
func (s *testSuite) TestSetServerCaps(c *C) {
	caps := &ProtoCaps{"#&~", "(ov)@+", "@+", "b,k,l,imnpstrDdRcC"}
	SetCaps(caps)
	gotcaps := GetCaps()
	c.Assert(gotcaps.Chantypes, Equals, caps.Chantypes)
	c.Assert(gotcaps.Prefix, Equals, caps.Prefix)
	c.Assert(gotcaps.Statusmsg, Equals, caps.Statusmsg)
	c.Assert(gotcaps.Chanmodes, Equals, caps.Chanmodes)
}
