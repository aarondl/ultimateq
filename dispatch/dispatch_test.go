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

//===========================================================
// Set up a type that can be used to mock irc.Sender
//===========================================================
type testSender struct {
}

func (tsender testSender) Writeln(s string) error {
	return nil
}

func (tsender testSender) GetKey() string {
	return ""
}

//===========================================================
// Set up a type that can be used to mock a callback for raw.
//===========================================================
type testCallback func(msg *irc.IrcMessage, sender irc.Sender)

type testHandler struct {
	callback testCallback
}

func (handler testHandler) HandleRaw(msg *irc.IrcMessage, sender irc.Sender) {
	if handler.callback != nil {
		handler.callback(msg, sender)
	}
}

//===========================================================
// Tests
//===========================================================
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
	cb := testHandler{}

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
	var s1, s2, s3 irc.Sender
	waiter := sync.WaitGroup{}
	h1 := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		msg1 = m
		s1 = s
		waiter.Done()
	}}
	h2 := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		msg2 = m
		s2 = s
		waiter.Done()
	}}
	h3 := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		msg3 = m
		s3 = s
		waiter.Done()
	}}

	d := CreateDispatcher()
	send := testSender{}

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	waiter.Add(2)
	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	quitmsg := &irc.IrcMessage{Name: irc.QUIT}
	d.Dispatch(privmsg, send)
	waiter.Wait()
	c.Assert(msg1.Name, Equals, irc.PRIVMSG)
	c.Assert(msg1, Equals, msg2)
	c.Assert(s1, NotNil)
	c.Assert(s1, Equals, s2)
	c.Assert(msg3, IsNil)
	waiter.Add(1)
	d.Dispatch(quitmsg, send)
	waiter.Wait()
	c.Assert(msg3.Name, Equals, irc.QUIT)
}

func (s *s) TestDispatcher_RawDispatch(c *C) {
	var msg1, msg2 *irc.IrcMessage
	waiter := sync.WaitGroup{}
	h1 := testHandler{func(m *irc.IrcMessage, send irc.Sender) {
		msg1 = m
		waiter.Done()
	}}
	h2 := testHandler{func(m *irc.IrcMessage, send irc.Sender) {
		msg2 = m
		waiter.Done()
	}}

	d := CreateDispatcher()
	send := testSender{}
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	waiter.Add(2)
	d.Dispatch(privmsg, send)
	waiter.Wait()
	c.Assert(msg1, Equals, privmsg)
	c.Assert(msg1, Equals, msg2)
}

// ================================
// Testing types
// ================================
type testPrivmsgHandler struct {
	callback func(*Message, irc.Sender)
}
type testPrivmsgUserHandler struct {
	callback func(*Message, irc.Sender)
}
type testPrivmsgChannelHandler struct {
	callback func(*Message, irc.Sender)
}
type testNoticeHandler struct {
	callback func(*Message, irc.Sender)
}
type testNoticeUserHandler struct {
	callback func(*Message, irc.Sender)
}
type testNoticeChannelHandler struct {
	callback func(*Message, irc.Sender)
}

// ================================
// Testing Callbacks
// ================================
func (t testPrivmsgHandler) Privmsg(msg *Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testPrivmsgUserHandler) PrivmsgUser(msg *Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testPrivmsgChannelHandler) PrivmsgChannel(msg *Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testNoticeHandler) Notice(msg *Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testNoticeUserHandler) NoticeUser(msg *Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testNoticeChannelHandler) NoticeChannel(msg *Message, sender irc.Sender) {
	t.callback(msg, sender)
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
	ph := testPrivmsgHandler{func(m *Message, _ irc.Sender) {
		p = m
		waiter.Done()
	}}
	puh := testPrivmsgUserHandler{func(m *Message, _ irc.Sender) {
		pu = m
		waiter.Done()
	}}
	pch := testPrivmsgChannelHandler{func(m *Message, _ irc.Sender) {
		pc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, puh)
	d.Register(irc.PRIVMSG, pch)

	waiter.Add(2)
	d.Dispatch(usermsg, nil)
	waiter.Wait()
	c.Assert(p, NotNil)
	c.Assert(pu.Raw, Equals, p.Raw)

	p, pu, pc = nil, nil, nil
	waiter.Add(2)
	d.Dispatch(chanmsg, nil)
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
	nh := testNoticeHandler{func(m *Message, _ irc.Sender) {
		n = m
		waiter.Done()
	}}
	nuh := testNoticeUserHandler{func(m *Message, _ irc.Sender) {
		nu = m
		waiter.Done()
	}}
	nch := testNoticeChannelHandler{func(m *Message, _ irc.Sender) {
		nc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, nh)
	d.Register(irc.NOTICE, nuh)
	d.Register(irc.NOTICE, nch)

	waiter.Add(2)
	d.Dispatch(usermsg, nil)
	waiter.Wait()
	c.Assert(n, NotNil)
	c.Assert(nu.Raw, Equals, n.Raw)

	n, nu, nc = nil, nil, nil
	waiter.Add(2)
	d.Dispatch(chanmsg, nil)
	waiter.Wait()
	c.Assert(n, NotNil)
	c.Assert(nc.Raw, Equals, n.Raw)
}

func (s *s) TestDispatcher_Sender(c *C) {
	d := CreateDispatcher()
	send := testSender{}

	msg := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan", "msg"},
		Sender: "nick!user@host.com",
	}

	d.Register(irc.PRIVMSG, func(msg *irc.IrcMessage, sender irc.Sender) {
		c.Assert(sender.GetKey(), NotNil)
		c.Assert(sender.Writeln(""), IsNil)
	})
	d.Dispatch(msg, send)
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
	ph := testPrivmsgHandler{func(m *Message, _ irc.Sender) {
		p = m
		waiter.Done()
	}}
	pch := testPrivmsgChannelHandler{func(m *Message, _ irc.Sender) {
		pc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, pch)

	waiter.Add(2)
	d.Dispatch(chanmsg, nil)
	waiter.Wait()
	c.Assert(p, NotNil)
	c.Assert(pc.Raw, Equals, p.Raw)

	p, pc = nil, nil
	waiter.Add(1)
	d.Dispatch(chanmsg2, nil)
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
	uh := testNoticeHandler{func(m *Message, _ irc.Sender) {
		u = m
		waiter.Done()
	}}
	uch := testNoticeChannelHandler{func(m *Message, _ irc.Sender) {
		uc = m
		waiter.Done()
	}}

	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, uh)
	d.Register(irc.NOTICE, uch)

	waiter.Add(2)
	d.Dispatch(chanmsg, nil)
	waiter.Wait()
	c.Assert(u, NotNil)
	c.Assert(uc.Raw, Equals, u.Raw)

	u, uc = nil, nil
	waiter.Add(1)
	d.Dispatch(chanmsg2, nil)
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
