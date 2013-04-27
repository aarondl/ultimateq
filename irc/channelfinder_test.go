package irc

import . "launchpad.net/gocheck"

func (s *s) TestChannelFinder(c *C) {
	finder := &ChannelFinder{}
	err := finder.BuildRegex(`*+?[]()-^`)
	c.Assert(err, IsNil)
	c.Assert(finder.channelRegexp, NotNil)
	c.Assert(len(finder.FindChannels(")channel")), Equals, 1)
}
