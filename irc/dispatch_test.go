package irc

import . "launchpad.net/gocheck"

type testingCallback func(msg *IrcMessage)

type testingHandler struct {
	callback testingCallback
}

func (handler testingHandler) HandleRaw(msg *IrcMessage) {
	if handler.callback != nil {
		handler.callback(msg)
	}
}

func (s *s) TestDispatcher(c *C) {
	d := CreateDispatcher()
	c.Assert(d, NotNil)
	c.Assert(d.events, NotNil)
}

func (s *s) TestDispatcher_Registration(c *C) {
	d := CreateDispatcher()
	cb := testingHandler{}

	id := d.Register("PRIVMSG", cb)
	c.Assert(id, Not(Equals), 0)
	id2 := d.Register("PRIVMSG", cb)
	c.Assert(id2, Not(Equals), id)
	ok := d.Unregister("privmsg", id)
	c.Assert(ok, Equals, true)
	ok = d.Unregister("privmsg", id)
	c.Assert(ok, Equals, false)
}
