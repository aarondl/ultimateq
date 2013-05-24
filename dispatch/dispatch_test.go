package dispatch

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
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
	h1 := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		msg1 = m
		s1 = s
	}}
	h2 := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		msg2 = m
		s2 = s
	}}
	h3 := testHandler{func(m *irc.IrcMessage, s irc.Sender) {
		msg3 = m
		s3 = s
	}}

	d := CreateDispatcher()
	send := testSender{}

	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.PRIVMSG, h2)
	d.Register(irc.QUIT, h3)

	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	quitmsg := &irc.IrcMessage{Name: irc.QUIT}
	d.Dispatch(privmsg, send)
	d.WaitForCompletion()
	c.Assert(msg1.Name, Equals, irc.PRIVMSG)
	c.Assert(msg1, Equals, msg2)
	c.Assert(s1, NotNil)
	c.Assert(s1, Equals, s2)
	c.Assert(msg3, IsNil)
	d.Dispatch(quitmsg, send)
	d.WaitForCompletion()
	c.Assert(msg3.Name, Equals, irc.QUIT)
}

func (s *s) TestDispatcher_RawDispatch(c *C) {
	var msg1, msg2 *irc.IrcMessage
	h1 := testHandler{func(m *irc.IrcMessage, send irc.Sender) {
		msg1 = m
	}}
	h2 := testHandler{func(m *irc.IrcMessage, send irc.Sender) {
		msg2 = m
	}}

	d := CreateDispatcher()
	send := testSender{}
	d.Register(irc.PRIVMSG, h1)
	d.Register(irc.RAW, h2)

	privmsg := &irc.IrcMessage{Name: irc.PRIVMSG}
	d.Dispatch(privmsg, send)
	d.WaitForCompletion()
	c.Assert(msg1, Equals, privmsg)
	c.Assert(msg1, Equals, msg2)
}

// ================================
// Testing types
// ================================
type testCallbackMsg func(*irc.Message, irc.Sender)

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
func (t testPrivmsgHandler) Privmsg(msg *irc.Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testPrivmsgUserHandler) PrivmsgUser(
	msg *irc.Message, sender irc.Sender) {

	t.callback(msg, sender)
}
func (t testPrivmsgChannelHandler) PrivmsgChannel(
	msg *irc.Message, sender irc.Sender) {

	t.callback(msg, sender)
}
func (t testPrivmsgAllHandler) Privmsg(
	msg *irc.Message, sender irc.Sender) {

	t.testCallbackNormal(msg, sender)
}
func (t testPrivmsgAllHandler) PrivmsgUser(
	msg *irc.Message, sender irc.Sender) {

	t.testCallbackUser(msg, sender)
}
func (t testPrivmsgAllHandler) PrivmsgChannel(
	msg *irc.Message, sender irc.Sender) {

	t.testCallbackChannel(msg, sender)
}
func (t testNoticeHandler) Notice(msg *irc.Message, sender irc.Sender) {
	t.callback(msg, sender)
}
func (t testNoticeUserHandler) NoticeUser(
	msg *irc.Message, sender irc.Sender) {

	t.callback(msg, sender)
}
func (t testNoticeChannelHandler) NoticeChannel(
	msg *irc.Message, sender irc.Sender) {

	t.callback(msg, sender)
}
func (t testNoticeAllHandler) Notice(
	msg *irc.Message, sender irc.Sender) {

	t.testCallbackNormal(msg, sender)
}
func (t testNoticeAllHandler) NoticeUser(
	msg *irc.Message, sender irc.Sender) {

	t.testCallbackUser(msg, sender)
}
func (t testNoticeAllHandler) NoticeChannel(
	msg *irc.Message, sender irc.Sender) {

	t.testCallbackChannel(msg, sender)
}

var privChanmsg = &irc.IrcMessage{
	Name:   irc.PRIVMSG,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var privUsermsg = &irc.IrcMessage{
	Name:   irc.PRIVMSG,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func (s *s) TestDispatcher_Privmsg(c *C) {
	var p, pu, pc *irc.Message
	ph := testPrivmsgHandler{func(m *irc.Message, _ irc.Sender) {
		p = m
	}}
	puh := testPrivmsgUserHandler{func(m *irc.Message, _ irc.Sender) {
		pu = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Message, _ irc.Sender) {
		pc = m
	}}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, puh)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privUsermsg, nil)
	d.WaitForCompletion()
	c.Assert(p, NotNil)
	c.Assert(pu.Raw, Equals, p.Raw)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForCompletion()
	c.Assert(p, NotNil)
	c.Assert(pc.Raw, Equals, p.Raw)
}

func (s *s) TestDispatcher_PrivmsgMultiple(c *C) {
	var p, pu, pc *irc.Message
	pall := testPrivmsgAllHandler{
		func(m *irc.Message, _ irc.Sender) {
			p = m
		},
		func(m *irc.Message, _ irc.Sender) {
			pu = m
		},
		func(m *irc.Message, _ irc.Sender) {
			pc = m
		},
	}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, pall)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privChanmsg, nil)
	d.WaitForCompletion()
	c.Assert(p, IsNil)
	c.Assert(pu, IsNil)
	c.Assert(pc, NotNil)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privUsermsg, nil)
	d.WaitForCompletion()
	c.Assert(p, IsNil)
	c.Assert(pu, NotNil)
	c.Assert(pc, IsNil)

	d = CreateDispatcher()
	d.Dispatch(privChanmsg, nil)
	d.Register(irc.PRIVMSG, pall)

	p, pu, pc = nil, nil, nil
	d.Dispatch(privUsermsg, nil)
	d.WaitForCompletion()
	c.Assert(p, NotNil)
	c.Assert(pu, IsNil)
	c.Assert(pc, IsNil)
}

var noticeChanmsg = &irc.IrcMessage{
	Name:   irc.NOTICE,
	Args:   []string{"#chan", "msg"},
	Sender: "nick!user@host.com",
}
var noticeUsermsg = &irc.IrcMessage{
	Name:   irc.NOTICE,
	Args:   []string{"user", "msg"},
	Sender: "nick!user@host.com",
}

func (s *s) TestDispatcher_Notice(c *C) {
	var n, nu, nc *irc.Message
	nh := testNoticeHandler{func(m *irc.Message, _ irc.Sender) {
		n = m
	}}
	nuh := testNoticeUserHandler{func(m *irc.Message, _ irc.Sender) {
		nu = m
	}}
	nch := testNoticeChannelHandler{func(m *irc.Message, _ irc.Sender) {
		nc = m
	}}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, nh)
	d.Register(irc.NOTICE, nuh)
	d.Register(irc.NOTICE, nch)

	d.Dispatch(noticeUsermsg, nil)
	d.WaitForCompletion()
	c.Assert(n, NotNil)
	c.Assert(nu.Raw, Equals, n.Raw)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForCompletion()
	c.Assert(n, NotNil)
	c.Assert(nc.Raw, Equals, n.Raw)
}

func (s *s) TestDispatcher_NoticeMultiple(c *C) {
	var n, nu, nc *irc.Message
	nall := testNoticeAllHandler{
		func(m *irc.Message, _ irc.Sender) {
			n = m
		},
		func(m *irc.Message, _ irc.Sender) {
			nu = m
		},
		func(m *irc.Message, _ irc.Sender) {
			nc = m
		},
	}

	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, nall)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeChanmsg, nil)
	d.WaitForCompletion()
	c.Assert(n, IsNil)
	c.Assert(nu, IsNil)
	c.Assert(nc, NotNil)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeUsermsg, nil)
	d.WaitForCompletion()
	c.Assert(n, IsNil)
	c.Assert(nu, NotNil)
	c.Assert(nc, IsNil)

	d = CreateDispatcher()
	d.Dispatch(noticeChanmsg, nil)
	d.Register(irc.NOTICE, nall)

	n, nu, nc = nil, nil, nil
	d.Dispatch(noticeUsermsg, nil)
	d.WaitForCompletion()
	c.Assert(n, NotNil)
	c.Assert(nu, IsNil)
	c.Assert(nc, IsNil)
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
	chanmsg2 := &irc.IrcMessage{
		Name:   irc.PRIVMSG,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var p, pc *irc.Message
	ph := testPrivmsgHandler{func(m *irc.Message, _ irc.Sender) {
		p = m
	}}
	pch := testPrivmsgChannelHandler{func(m *irc.Message, _ irc.Sender) {
		pc = m
	}}

	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	d.Register(irc.PRIVMSG, ph)
	d.Register(irc.PRIVMSG, pch)

	d.Dispatch(privChanmsg, nil)
	d.WaitForCompletion()
	c.Assert(p, NotNil)
	c.Assert(pc.Raw, Equals, p.Raw)

	p, pc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForCompletion()
	c.Assert(p, NotNil)
	c.Assert(pc, IsNil)
}

func (s *s) TestDispatcher_FilterNoticeChannels(c *C) {
	chanmsg2 := &irc.IrcMessage{
		Name:   irc.NOTICE,
		Args:   []string{"#chan2", "msg"},
		Sender: "nick!user@host.com",
	}

	var u, uc *irc.Message
	uh := testNoticeHandler{func(m *irc.Message, _ irc.Sender) {
		u = m
	}}
	uch := testNoticeChannelHandler{func(m *irc.Message, _ irc.Sender) {
		uc = m
	}}

	d, err := CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#"}, []string{"#CHAN"})
	c.Assert(err, IsNil)
	d.Register(irc.NOTICE, uh)
	d.Register(irc.NOTICE, uch)

	d.Dispatch(noticeChanmsg, nil)
	d.WaitForCompletion()
	c.Assert(u, NotNil)
	c.Assert(uc.Raw, Equals, u.Raw)

	u, uc = nil, nil
	d.Dispatch(chanmsg2, nil)
	d.WaitForCompletion()
	c.Assert(u, NotNil)
	c.Assert(uc, IsNil)
}

func (s *s) TestDispatcher_AddRemoveChannels(c *C) {
	chans := []string{"#chan1", "#chan2", "#chan3"}
	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, chans)
	c.Assert(err, IsNil)

	c.Assert(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Assert(d.chans[i], Equals, v)
	}

	d.RemoveChannels(chans...)
	c.Assert(d.chans, IsNil)
	d.RemoveChannels(chans...)
	c.Assert(d.chans, IsNil)
	d.RemoveChannels()
	c.Assert(d.chans, IsNil)

	d.Channels(chans)
	d.RemoveChannels(chans[1:]...)
	c.Assert(len(d.chans), Equals, len(chans)-2)
	for i, v := range chans[:1] {
		c.Assert(d.chans[i], Equals, v)
	}
	d.AddChannels(chans[1:]...)
	c.Assert(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Assert(d.chans[i], Equals, v)
	}
	d.AddChannels(chans[0])
	d.AddChannels()
	c.Assert(len(d.chans), Equals, len(chans))
	d.RemoveChannels(chans...)
	d.AddChannels(chans...)
	c.Assert(len(d.chans), Equals, len(chans))
}

func (s *s) TestDispatcher_UpdateChannels(c *C) {
	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)
	c.Assert(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Assert(d.chans[i], Equals, v)
	}
	d.Channels([]string{})
	c.Assert(len(d.chans), Equals, 0)
	d.Channels(chans)
	c.Assert(len(d.chans), Equals, len(chans))
	d.Channels(nil)
	c.Assert(len(d.chans), Equals, 0)
}

func (s *s) TestDispatcher_UpdateProtoCaps(c *C) {
	d, err := CreateRichDispatcher(&irc.ProtoCaps{Chantypes: "#"}, nil)
	c.Assert(err, IsNil)
	var should bool
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Assert(should, Equals, true)
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"&chan"}})
	c.Assert(should, Equals, false)

	err = d.Protocaps(&irc.ProtoCaps{Chantypes: "&"})
	c.Assert(err, IsNil)
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"#chan"}})
	c.Assert(should, Equals, false)
	should = d.shouldDispatch(true, &irc.IrcMessage{Args: []string{"&chan"}})
	c.Assert(should, Equals, true)
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
