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
	d, err := CreateRichDispatcher(nil)
	c.Assert(err, Equals, errProtoCapsMissing)
	d, err = CreateRichDispatcher(&ProtoCaps{Chantypes: "H"})
	c.Assert(err, NotNil)
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

// ================================
// Testing types
// ================================
type testingPrivmsgHandler struct {
	callback func(*Message)
}
type testingPrivmsgUserHandler struct {
	callback func(*Message)
}
type testingPrivmsgChannelHandler struct {
	callback func(*Message)
}
type testingNoticeHandler struct {
	callback func(*Message)
}
type testingNoticeUserHandler struct {
	callback func(*Message)
}
type testingNoticeChannelHandler struct {
	callback func(*Message)
}

// ================================
// Testing Callbacks
// ================================
func (t testingPrivmsgHandler) Privmsg(msg *Message) {
	t.callback(msg)
}
func (t testingPrivmsgUserHandler) PrivmsgUser(msg *Message) {
	t.callback(msg)
}
func (t testingPrivmsgChannelHandler) PrivmsgChannel(msg *Message) {
	t.callback(msg)
}
func (t testingNoticeHandler) Notice(msg *Message) {
	t.callback(msg)
}
func (t testingNoticeUserHandler) NoticeUser(msg *Message) {
	t.callback(msg)
}
func (t testingNoticeChannelHandler) NoticeChannel(msg *Message) {
	t.callback(msg)
}

func (s *s) TestDispatcher_Privmsg(c *C) {
	chanmsg := &IrcMessage{
		Name:   PRIVMSG,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}
	usermsg := &IrcMessage{
		Name:   PRIVMSG,
		Args:   []string{"user", "msg"},
		Sender: "nick!user@host.com",
	}

	var p, pu, pc *Message
	ph := testingPrivmsgHandler{func(m *Message) {
		p = m
	}}
	puh := testingPrivmsgUserHandler{func(m *Message) {
		pu = m
	}}
	pch := testingPrivmsgChannelHandler{func(m *Message) {
		pc = m
	}}

	d, err := CreateRichDispatcher(&ProtoCaps{Chantypes: "#"})
	c.Assert(err, IsNil)
	d.Register(PRIVMSG, ph)
	d.Register(PRIVMSG, puh)
	d.Register(PRIVMSG, pch)

	d.Dispatch(PRIVMSG, usermsg)
	c.Assert(p, NotNil)
	c.Assert(pu.Raw, Equals, p.Raw)

	p, pu, pc = nil, nil, nil
	d.Dispatch(PRIVMSG, chanmsg)
	c.Assert(p, NotNil)
	c.Assert(pc.Raw, Equals, p.Raw)
}

func (s *s) TestDispatcher_Notice(c *C) {
	chanmsg := &IrcMessage{
		Name:   NOTICE,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}
	usermsg := &IrcMessage{
		Name:   NOTICE,
		Args:   []string{"user", "msg"},
		Sender: "nick!user@host.com",
	}

	var n, nu, nc *Message
	nh := testingNoticeHandler{func(m *Message) {
		n = m
	}}
	nuh := testingNoticeUserHandler{func(m *Message) {
		nu = m
	}}
	nch := testingNoticeChannelHandler{func(m *Message) {
		nc = m
	}}

	d, err := CreateRichDispatcher(&ProtoCaps{Chantypes: "#"})
	c.Assert(err, IsNil)
	d.Register(NOTICE, nh)
	d.Register(NOTICE, nuh)
	d.Register(NOTICE, nch)

	d.Dispatch(NOTICE, usermsg)
	c.Assert(n, NotNil)
	c.Assert(nu.Raw, Equals, n.Raw)

	n, nu, nc = nil, nil, nil
	d.Dispatch(NOTICE, chanmsg)
	c.Assert(n, NotNil)
	c.Assert(nc.Raw, Equals, n.Raw)
}
