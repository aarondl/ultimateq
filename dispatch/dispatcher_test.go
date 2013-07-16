package dispatch

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
)

var core, _ = CreateDispatchCore(irc.CreateProtoCaps())

//===========================================================
// Set up a type that can be used to mock irc.Endpoint
//===========================================================
type testPoint struct {
	*irc.Helper
}

func (tsender testPoint) GetKey() string {
	return ""
}

//===========================================================
// Set up a type that can be used to mock a callback for raw.
//===========================================================
type testCallback func(msg *irc.Message, ep irc.Endpoint)

type testHandler struct {
	callback testCallback
}

func (handler testHandler) HandleRaw(msg *irc.Message, ep irc.Endpoint) {
	if handler.callback != nil {
		handler.callback(msg, ep)
	}
}

//===========================================================
// Tests
//===========================================================
func (s *s) TestDispatcher(c *C) {
	d := CreateDispatcher(core)
	c.Check(d, NotNil)
	c.Check(d.events, NotNil)
}

func (s *s) TestDispatcher_Registration(c *C) {
	d := CreateDispatcher(core)
	cb := testHandler{}

	id := d.Register(irc.PRIVMSG, cb)
	c.Check(id, Not(Equals), 0)
	id2 := d.Register(irc.PRIVMSG, cb)
	c.Check(id2, Not(Equals), id)
	ok := d.Unregister("privmsg", id)
	c.Check(ok, Equals, true)
	ok = d.Unregister("privmsg", id)
	c.Check(ok, Equals, false)
}

func (s *s) TestDispatcher_Dispatching(c *C) {
	var msg1, msg2, msg3 *irc.Message
	var s1, s2, s3 irc.Endpoint
	h1 := testHandler{func(m *irc.Message, s irc.Endpoint) {
		msg1 = m
		s1 = s
	}}
	h2 := testHandler{func(m *irc.Message, s irc.Endpoint) {
		msg2 = m
		s2 = s
	}}
	h3 := testHandler{func(m *irc.Message, s irc.Endpoint) {
		msg3 = m
		s3 = s
	}}

	d := CreateDispatcher(core)
	send := testPoint{&irc.Helper{}}

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	privmsg := &irc.Message{Name: irc.PRIVMSG}
	quitmsg := &irc.Message{Name: irc.QUIT}
	d.Dispatch(privmsg, send)
	d.WaitForHandlers()
	c.Check(msg1.Name, Equals, irc.PRIVMSG)
	c.Check(msg1, Equals, msg2)
	c.Check(s1, NotNil)
	c.Check(s1, Equals, s2)
	c.Check(msg3, IsNil)
	d.Dispatch(quitmsg, send)
	d.WaitForHandlers()
	c.Check(msg3.Name, Equals, irc.QUIT)
}

func (s *s) TestDispatcher_RawDispatch(c *C) {
	var msg1, msg2 *irc.Message
	h1 := testHandler{func(m *irc.Message, send irc.Endpoint) {
		msg1 = m
	}}
	h2 := testHandler{func(m *irc.Message, send irc.Endpoint) {
		msg2 = m
	}}

	d := CreateDispatcher(core)
	send := testPoint{&irc.Helper{}}
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.Message{Name: irc.PRIVMSG}
	d.Dispatch(privmsg, send)
	d.WaitForHandlers()
	c.Check(msg1, Equals, privmsg)
	c.Check(msg1, Equals, msg2)
}

// ================================
// Testing types
// ================================
type testCallbackMsg func(*irc.Message, irc.Endpoint)

type testPrivmsgHandler struct {
	callback testCallbackMsg
}
type testPrivmsgUserHandler struct {
	callback testCallbackMsg
}
type testPrivmsgChannelHandler struct {
	callback testCallbackMsg
}
type testPrivmsgAllHandler struct {
	testCallbackNormal, testCallbackUser, testCallbackChannel testCallbackMsg
}
type testNoticeHandler struct {
	callback testCallbackMsg
}
type testNoticeUserHandler struct {
	callback testCallbackMsg
}
type testNoticeChannelHandler struct {
	callback testCallbackMsg
}
type testNoticeAllHandler struct {
	testCallbackNormal, testCallbackUser, testCallbackChannel testCallbackMsg
}

// ================================
// Testing Callbacks
// ================================
func (t testPrivmsgHandler) Privmsg(msg *irc.Message, ep irc.Endpoint) {
	t.callback(msg, ep)
}
func (t testPrivmsgUserHandler) PrivmsgUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testPrivmsgChannelHandler) PrivmsgChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testPrivmsgAllHandler) Privmsg(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackNormal(msg, ep)
}
func (t testPrivmsgAllHandler) PrivmsgUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackUser(msg, ep)
}
func (t testPrivmsgAllHandler) PrivmsgChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackChannel(msg, ep)
}
func (t testNoticeHandler) Notice(msg *irc.Message, ep irc.Endpoint) {
	t.callback(msg, ep)
}
func (t testNoticeUserHandler) NoticeUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testNoticeChannelHandler) NoticeChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.callback(msg, ep)
}
func (t testNoticeAllHandler) Notice(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackNormal(msg, ep)
}
func (t testNoticeAllHandler) NoticeUser(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackUser(msg, ep)
}
func (t testNoticeAllHandler) NoticeChannel(
	msg *irc.Message, ep irc.Endpoint) {

	t.testCallbackChannel(msg, ep)
}

var privChanmsg = &irc.Message{
	Name:   irc.PRIVMSG,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var privUsermsg = &irc.Message{
	Name:   irc.PRIVMSG,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func (s *s) TestDispatcher_Privmsg(c *C) {
	var p, pu, pc *irc.Message
	ph := testPrivmsgHandler{func(m *irc.Message, _ irc.Endpoint) {
		p = m
	}}
	puh := testPrivmsgUserHandler{func(m *irc.Message, _ irc.Endpoint) {
		pu = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		pc = m
	}}

	d := CreateDispatcher(core)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, puh)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privUsermsg, nil)
	d.WaitForHandlers()
	c.Check(p, NotNil)
	c.Check(pu, Equals, p)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForHandlers()
	c.Check(p, NotNil)
	c.Check(pc, Equals, p)
}

func (s *s) TestDispatcher_PrivmsgMultiple(c *C) {
	var p, pu, pc *irc.Message
	pall := testPrivmsgAllHandler{
		func(m *irc.Message, _ irc.Endpoint) {
			p = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			pu = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			pc = m
		},
	}

	d := CreateDispatcher(core)
	d.Register(irc.PRIVMSG, pall)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForHandlers()
	c.Check(p, IsNil)
	c.Check(pu, IsNil)
	c.Check(pc, NotNil)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privUsermsg, nil)
	d.WaitForHandlers()
	c.Check(p, IsNil)
	c.Check(pu, NotNil)
	c.Check(pc, IsNil)
}

var noticeChanmsg = &irc.Message{
	Name:   irc.NOTICE,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var noticeUsermsg = &irc.Message{
	Name:   irc.NOTICE,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func (s *s) TestDispatcher_Notice(c *C) {
	var n, nu, nc *irc.Message
	nh := testNoticeHandler{func(m *irc.Message, _ irc.Endpoint) {
		n = m
	}}
	nuh := testNoticeUserHandler{func(m *irc.Message, _ irc.Endpoint) {
		nu = m
	}}
	nch := testNoticeChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		nc = m
	}}

	d := CreateDispatcher(core)
	d.Register(irc.NOTICE, nh)
	d.Register(irc.NOTICE, nuh)
	d.Register(irc.NOTICE, nch)

	d.Dispatch(noticeUsermsg, nil)
	d.WaitForHandlers()
	c.Check(n, NotNil)
	c.Check(nu, Equals, n)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForHandlers()
	c.Check(n, NotNil)
	c.Check(nc, Equals, n)
}

func (s *s) TestDispatcher_NoticeMultiple(c *C) {
	var n, nu, nc *irc.Message
	nall := testNoticeAllHandler{
		func(m *irc.Message, _ irc.Endpoint) {
			n = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			nu = m
		},
		func(m *irc.Message, _ irc.Endpoint) {
			nc = m
		},
	}

	d := CreateDispatcher(core)
	d.Register(irc.NOTICE, nall)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForHandlers()
	c.Check(n, IsNil)
	c.Check(nu, IsNil)
	c.Check(nc, NotNil)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeUsermsg, nil)
	d.WaitForHandlers()
	c.Check(n, IsNil)
	c.Check(nu, NotNil)
	c.Check(nc, IsNil)
}

func (s *s) TestDispatcher_Sender(c *C) {
	d := CreateDispatcher(core)
	send := testPoint{&irc.Helper{}}

	msg := &irc.Message{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}

	d.Register(irc.PRIVMSG, func(msg *irc.Message, point irc.Endpoint) {
		c.Check(point.GetKey(), NotNil)
		c.Check(point.Sendln(""), IsNil)
	})
	d.Dispatch(msg, send)
}

func (s *s) TestDispatcher_FilterPrivmsgChannels(c *C) {
	chanmsg2 := &irc.Message{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var p, pc *irc.Message
	ph := testPrivmsgHandler{func(m *irc.Message, _ irc.Endpoint) {
		p = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		pc = m
	}}

	dcore, err := CreateDispatchCore(irc.CreateProtoCaps(), "#CHAN")
	c.Check(err, IsNil)
	d := CreateDispatcher(dcore)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privChanmsg, nil)
	d.WaitForHandlers()
	c.Check(p, NotNil)
	c.Check(pc, Equals, p)

	p, pc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForHandlers()
	c.Check(p, NotNil)
	c.Check(pc, IsNil)
}

func (s *s) TestDispatcher_FilterNoticeChannels(c *C) {
	chanmsg2 := &irc.Message{
		Name:   irc.NOTICE,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var u, uc *irc.Message
	uh := testNoticeHandler{func(m *irc.Message, _ irc.Endpoint) {
		u = m
	}}
	uch := testNoticeChannelHandler{func(m *irc.Message, _ irc.Endpoint) {
		uc = m
	}}

	dcore, err := CreateDispatchCore(irc.CreateProtoCaps(), "#CHAN")
	c.Check(err, IsNil)
	d := CreateDispatcher(dcore)
	d.Register(irc.NOTICE, uh)
	d.Register(irc.NOTICE, uch)

	d.Dispatch(noticeChanmsg, nil)
	d.WaitForHandlers()
	c.Check(u, NotNil)
	c.Check(uc, Equals, u)

	u, uc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForHandlers()
	c.Check(u, NotNil)
	c.Check(uc, IsNil)
}

func (s *s) TestDispatchCore_ShouldDispatch(c *C) {
	d := CreateDispatcher(core)

	var should bool
	should = d.shouldDispatch(true, "#chan")
	c.Check(should, Equals, true)
	should = d.shouldDispatch(false, "#chan2")
	c.Check(should, Equals, false)

	should = d.shouldDispatch(true, "chan")
	c.Check(should, Equals, false)
	should = d.shouldDispatch(false, "chan")
	c.Check(should, Equals, true)
}
