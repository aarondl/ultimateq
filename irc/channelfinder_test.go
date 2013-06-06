package irc

import . "launchpad.net/gocheck"

func (s *s) TestChannelFinder(c *C) {
	finder, err := CreateChannelFinder(`~&*+?[]()-^`)
	c.Check(err, IsNil)
	c.Check(finder.channelRegexp, NotNil)
	c.Check(len(finder.FindChannels(")channel")), Equals, 1)
	c.Check(finder.IsChannel(")channel"), Equals, true)
}

func (s *s) TestChannelFinder_Error(c *C) {
	_, err := CreateChannelFinder(`H`)
	c.Check(err, NotNil)
}
