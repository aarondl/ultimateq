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

	id := d.Register(PRIVMSG, cb)
	c.Assert(id, Not(Equals), 0)
	id2 := d.Register(PRIVMSG, cb)
	c.Assert(id2, Not(Equals), id)
	ok := d.Unregister("privmsg", id)
	c.Assert(ok, Equals, true)
	ok = d.Unregister("privmsg", id)
	c.Assert(ok, Equals, false)
}

func (s *s) TestDispatcher_Dispatching(c *C) {
	var msg1, msg2, msg3 *IrcMessage
	h1 := testingHandler{func(m *IrcMessage) {
		msg1 = m
	}}
	h2 := testingHandler{func(m *IrcMessage) {
		msg2 = m
	}}
	h3 := testingHandler{func(m *IrcMessage) {
		msg3 = m
	}}

	d := CreateDispatcher()

	d.Register(PRIVMSG, h1)
	d.Register(PRIVMSG, h2)
	d.Register(QUIT, h3)

	privmsg := &IrcMessage{Name: PRIVMSG}
	quitmsg := &IrcMessage{Name: QUIT}
	d.Dispatch(PRIVMSG, privmsg)
	c.Assert(msg1.Name, Equals, PRIVMSG)
	c.Assert(msg1, Equals, msg2)
	c.Assert(msg3, IsNil)
	d.Dispatch(QUIT, quitmsg)
	c.Assert(msg3.Name, Equals, QUIT)
}

func (s *s) TestDispatcher_RawDispatch(c *C) {
	var msg1, msg2 *IrcMessage
	h1 := testingHandler{func(m *IrcMessage) {
		msg1 = m
	}}
	h2 := testingHandler{func(m *IrcMessage) {
		msg2 = m
	}}

	d := CreateDispatcher()
	d.Register(PRIVMSG, h1)
	d.Register(RAW, h2)

	privmsg := &IrcMessage{Name: PRIVMSG}
	d.Dispatch(PRIVMSG, privmsg)
	c.Assert(msg1, Equals, privmsg)
	c.Assert(msg1, Equals, msg2)
}
