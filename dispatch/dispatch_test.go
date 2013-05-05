package dispatch

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"sync"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package

type s struct{}

var _ = Suite(&s{})

type testingCallback func(msg *irc.IrcMessage)

type testingHandler struct {
	callback testingCallback
}

func (handler testingHandler) HandleRaw(msg *irc.IrcMessage) {
	if handler.callback != nil {
		handler.callback(msg)
	}
}

func (s *s) TestDispatcher(c *C) {
	d := CreateDispatcher()
	c.Assert(d, NotNil)
	c.Assert(d.events, NotNil)
	d, err := CreateRichDispatcher(nil, nil)
	c.Assert(err, Equals, errProtoCapsMissing)
	d, err = CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "H"}, nil)
	c.Assert(err, NotNil)
}

func (s *s) TestDispatcher_Registration(c *C) {
	d := CreateDispatcher()
	cb := testingHandler{}

	id := d.Register(irc.PRIVMSG, cb)
	c.Assert(id, Not(Equals), 0)
	id2 := d.Register(irc.PRIVMSG, cb)
	c.Assert(id2, Not(Equals), id)
	ok := d.Unregister("privmsg", id)
	c.Assert(ok, Equals, true)
	ok = d.Unregister("privmsg", id)
	c.Assert(ok, Equals, false)
}

func (s *s) TestDispatcher_Dispatching(c *C) {
	var msg1, msg2, msg3 *irc.IrcMessage
	waiter := sync.WaitGroup{}
	h1 := testingHandler{func(m *irc.IrcMessage) {
		msg1 = m
		waiter.Done()
	}}
	h2 := testingHandler{func(m *irc.IrcMessage) {
		msg2 = m
		waiter.Done()
	}}
	h3 := testingHandler{func(m *irc.IrcMessage) {
		msg3 = m
		waiter.Done()
	}}

	d := CreateDispatcher()

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	waiter.Add(2)
	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	quitmsg := &irc.IrcMessage{Name: irc.QUIT}
	d.Dispatch(irc.PRIVMSG, privmsg)
	waiter.Wait()
	c.Assert(msg1.Name, Equals, irc.PRIVMSG)
	c.Assert(msg1, Equals, msg2)
	c.Assert(msg3, IsNil)
	waiter.Add(1)
	d.Dispatch(irc.QUIT, quitmsg)
	waiter.Wait()
	c.Assert(msg3.Name, Equals, irc.QUIT)
}

func (s *s) TestDispatcher_RawDispatch(c *C) {
	var msg1, msg2 *irc.IrcMessage
	waiter := sync.WaitGroup{}
	h1 := testingHandler{func(m *irc.IrcMessage) {
		msg1 = m
		waiter.Done()
	}}
	h2 := testingHandler{func(m *irc.IrcMessage) {
		msg2 = m
		waiter.Done()
	}}

	d := CreateDispatcher()
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	waiter.Add(2)
	d.Dispatch(irc.PRIVMSG, privmsg)
	waiter.Wait()
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
	chanmsg := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}
	usermsg := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"user", "msg"},
		Sender: "nick!user@host.com",
	}

	var p, pu, pc *Message
	waiter := sync.WaitGroup{}
	ph := testingPrivmsgHandler{func(m *Message) {
		p = m
		waiter.Done()
	}}
	puh := testingPrivmsgUserHandler{func(m *Message) {
		pu = m
		waiter.Done()
	}}
	pch := testingPrivmsgChannelHandler{func(m *Message) {
		pc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, puh)
	d.Register(irc.PRIVMSG, pch)

	waiter.Add(2)
	d.Dispatch(irc.PRIVMSG, usermsg)
	waiter.Wait()
	c.Assert(p, NotNil)
	c.Assert(pu.Raw, Equals, p.Raw)

	p, pu, pc = nil, nil, nil
	waiter.Add(2)
	d.Dispatch(irc.PRIVMSG, chanmsg)
	waiter.Wait()
	c.Assert(p, NotNil)
	c.Assert(pc.Raw, Equals, p.Raw)
}

func (s *s) TestDispatcher_Notice(c *C) {
	chanmsg := &irc.IrcMessage{
		Name:   irc.NOTICE,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}
	usermsg := &irc.IrcMessage{
		Name:   irc.NOTICE,
		Args:   []string{"user", "msg"},
		Sender: "nick!user@host.com",
	}

	var n, nu, nc *Message
	waiter := sync.WaitGroup{}
	nh := testingNoticeHandler{func(m *Message) {
		n = m
		waiter.Done()
	}}
	nuh := testingNoticeUserHandler{func(m *Message) {
		nu = m
		waiter.Done()
	}}
	nch := testingNoticeChannelHandler{func(m *Message) {
		nc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, nh)
	d.Register(irc.NOTICE, nuh)
	d.Register(irc.NOTICE, nch)

	waiter.Add(2)
	d.Dispatch(irc.NOTICE, usermsg)
	waiter.Wait()
	c.Assert(n, NotNil)
	c.Assert(nu.Raw, Equals, n.Raw)

	n, nu, nc = nil, nil, nil
	waiter.Add(2)
	d.Dispatch(irc.NOTICE, chanmsg)
	waiter.Wait()
	c.Assert(n, NotNil)
	c.Assert(nc.Raw, Equals, n.Raw)
}

func (s *s) TestDispatcher_FilterPrivmsgChannels(c *C) {
	chanmsg := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}
	chanmsg2 := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var p, pc *Message
	waiter := sync.WaitGroup{}
	ph := testingPrivmsgHandler{func(m *Message) {
		p = m
		waiter.Done()
	}}
	pch := testingPrivmsgChannelHandler{func(m *Message) {
		pc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, pch)

	waiter.Add(2)
	d.Dispatch(irc.PRIVMSG, chanmsg)
	waiter.Wait()
	c.Assert(p, NotNil)
	c.Assert(pc.Raw, Equals, p.Raw)

	p, pc = nil, nil
	waiter.Add(1)
	d.Dispatch(irc.PRIVMSG, chanmsg2)
	waiter.Wait()
	c.Assert(p, NotNil)
	c.Assert(pc, IsNil)
}

func (s *s) TestDispatcher_FilterNoticeChannels(c *C) {
	chanmsg := &irc.IrcMessage{
		Name:   irc.NOTICE,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}
	chanmsg2 := &irc.IrcMessage{
		Name:   irc.NOTICE,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var u, uc *Message
	waiter := sync.WaitGroup{}
	uh := testingNoticeHandler{func(m *Message) {
		u = m
		waiter.Done()
	}}
	uch := testingNoticeChannelHandler{func(m *Message) {
		uc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, uh)
	d.Register(irc.NOTICE, uch)

	waiter.Add(2)
	d.Dispatch(irc.NOTICE, chanmsg)
	waiter.Wait()
	c.Assert(u, NotNil)
	c.Assert(uc.Raw, Equals, u.Raw)

	u, uc = nil, nil
	waiter.Add(1)
	d.Dispatch(irc.NOTICE, chanmsg2)
	waiter.Wait()
	c.Assert(u, NotNil)
	c.Assert(uc, IsNil)
}

func (s *s) TestDispatcher_shouldDispatch(c *C) {
	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)

	var should bool
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Assert(should, Equals, true)
	should = d.shouldDispatch(false, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Assert(should, Equals, false)

	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"chan"}})
	c.Assert(should, Equals, false)
	should = d.shouldDispatch(false, &irc.IrcMessage{Args: []string{"chan"}})
	c.Assert(should, Equals, true)
}

func (s *s) TestDispatcher_filterChannelDispatch(c *C) {
	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	c.Assert(d.chans, NotNil)

	var should bool
	should = d.checkChannels(&irc.IrcMessage{Args: []string{"#chan"}})
	c.Assert(should, Equals, true)
	should = d.checkChannels(&irc.IrcMessage{Args: []string{"#chan2"}})
	c.Assert(should, Equals, false)
}
