package irc

import . "launchpad.net/gocheck"

func (s *s) TestChannelFinder(c *C) {
	finder, err := CreateChannelFinder(`*+?[]()-^`)
	c.Assert(err, IsNil)
	c.Assert(finder.channelRegexp, NotNil)
	c.Assert(len(finder.FindChannels(")channel")), Equals, 1)
	c.Assert(finder.IsChannel(")channel"), Equals, true)
}
